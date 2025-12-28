package bitgo

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// IdempotencyService handles idempotency for BitGo operations
type IdempotencyService struct {
	cache  map[string]*IdempotencyRecord
	mutex  sync.RWMutex
	logger Logger
	ttl    time.Duration
}

// IdempotencyRecord represents a cached operation result
type IdempotencyRecord struct {
	Key         string            `json:"key"`
	RequestID   string            `json:"requestId"`
	Operation   string            `json:"operation"`
	Status      IdempotencyStatus `json:"status"`
	Request     interface{}       `json:"request"`
	Response    interface{}       `json:"response,omitempty"`
	Error       string            `json:"error,omitempty"`
	CreatedAt   time.Time         `json:"createdAt"`
	ExpiresAt   time.Time         `json:"expiresAt"`
	Attempts    int               `json:"attempts"`
	LastAttempt time.Time         `json:"lastAttempt"`
}

// IdempotencyStatus represents the status of an idempotent operation
type IdempotencyStatus string

const (
	IdempotencyStatusPending   IdempotencyStatus = "pending"
	IdempotencyStatusCompleted IdempotencyStatus = "completed"
	IdempotencyStatusFailed    IdempotencyStatus = "failed"
	IdempotencyStatusExpired   IdempotencyStatus = "expired"
)

// NewIdempotencyService creates a new idempotency service
func NewIdempotencyService(logger Logger, ttl time.Duration) *IdempotencyService {
	if ttl == 0 {
		ttl = 24 * time.Hour // Default 24 hour TTL
	}

	service := &IdempotencyService{
		cache:  make(map[string]*IdempotencyRecord),
		logger: logger,
		ttl:    ttl,
	}

	// Start cleanup routine
	go service.cleanupExpired()

	return service
}

// GenerateKey generates an idempotency key based on operation and request parameters
func (s *IdempotencyService) GenerateKey(operation string, request interface{}) string {
	// Create a deterministic hash of the operation and request
	data := map[string]interface{}{
		"operation": operation,
		"request":   request,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		s.logger.Warn("Failed to marshal request for idempotency key", "error", err)
		return uuid.New().String() // Fallback to random UUID
	}

	hash := sha256.Sum256(jsonData)
	return hex.EncodeToString(hash[:])
}

// CheckOrStore checks if an operation is already in progress or completed
// Returns (record, isNew) where isNew indicates if this is a new operation
func (s *IdempotencyService) CheckOrStore(ctx context.Context, key, operation string, request interface{}) (*IdempotencyRecord, bool, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Check if record exists
	if record, exists := s.cache[key]; exists {
		// Check if expired
		if time.Now().After(record.ExpiresAt) {
			delete(s.cache, key)
			s.logger.Info("Expired idempotency record removed", "key", key)
		} else {
			s.logger.Info("Found existing idempotency record",
				"key", key,
				"status", record.Status,
				"created_at", record.CreatedAt,
			)
			return record, false, nil
		}
	}

	// Create new record
	record := &IdempotencyRecord{
		Key:         key,
		RequestID:   uuid.New().String(),
		Operation:   operation,
		Status:      IdempotencyStatusPending,
		Request:     request,
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(s.ttl),
		Attempts:    1,
		LastAttempt: time.Now(),
	}

	s.cache[key] = record

	s.logger.Info("Created new idempotency record",
		"key", key,
		"operation", operation,
		"request_id", record.RequestID,
	)

	return record, true, nil
}

// UpdateRecord updates an existing idempotency record
func (s *IdempotencyService) UpdateRecord(key string, status IdempotencyStatus, response interface{}, err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	record, exists := s.cache[key]
	if !exists {
		s.logger.Warn("Attempted to update non-existent idempotency record", "key", key)
		return
	}

	record.Status = status
	record.Response = response
	record.LastAttempt = time.Now()

	if err != nil {
		record.Error = err.Error()
	}

	s.logger.Info("Updated idempotency record",
		"key", key,
		"status", status,
		"attempts", record.Attempts,
	)
}

// RetryRecord increments the attempt count for a record
func (s *IdempotencyService) RetryRecord(key string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	record, exists := s.cache[key]
	if !exists {
		s.logger.Warn("Attempted to retry non-existent idempotency record", "key", key)
		return
	}

	record.Attempts++
	record.LastAttempt = time.Now()

	s.logger.Info("Incremented retry count for idempotency record",
		"key", key,
		"attempts", record.Attempts,
	)
}

// GetRecord retrieves a specific idempotency record
func (s *IdempotencyService) GetRecord(key string) (*IdempotencyRecord, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	record, exists := s.cache[key]
	if !exists {
		return nil, false
	}

	// Check if expired
	if time.Now().After(record.ExpiresAt) {
		return nil, false
	}

	return record, true
}

// cleanupExpired removes expired records from the cache
func (s *IdempotencyService) cleanupExpired() {
	ticker := time.NewTicker(time.Hour) // Cleanup every hour
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.performCleanup()
		}
	}
}

// performCleanup removes expired records
func (s *IdempotencyService) performCleanup() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	now := time.Now()
	expiredKeys := make([]string, 0)

	for key, record := range s.cache {
		if now.After(record.ExpiresAt) {
			expiredKeys = append(expiredKeys, key)
		}
	}

	for _, key := range expiredKeys {
		delete(s.cache, key)
	}

	if len(expiredKeys) > 0 {
		s.logger.Info("Cleaned up expired idempotency records", "count", len(expiredKeys))
	}
}

// IdempotentOperation represents a function that can be executed idempotently
type IdempotentOperation func(ctx context.Context) (interface{}, error)

// ExecuteIdempotent executes an operation idempotently
func (s *IdempotencyService) ExecuteIdempotent(ctx context.Context, key, operation string, request interface{}, op IdempotentOperation) (interface{}, error) {
	// Check if operation already exists or is in progress
	record, isNew, err := s.CheckOrStore(ctx, key, operation, request)
	if err != nil {
		return nil, fmt.Errorf("failed to check idempotency: %w", err)
	}

	// If not new, return existing result or wait for completion
	if !isNew {
		switch record.Status {
		case IdempotencyStatusCompleted:
			s.logger.Info("Returning cached result for idempotent operation", "key", key)
			return record.Response, nil

		case IdempotencyStatusFailed:
			s.logger.Info("Returning cached error for idempotent operation", "key", key)
			if record.Error != "" {
				return nil, fmt.Errorf("cached error: %s", record.Error)
			}
			return nil, fmt.Errorf("operation failed previously")

		case IdempotencyStatusPending:
			// Operation is in progress, this is a duplicate request
			s.logger.Warn("Duplicate request detected for pending operation", "key", key)
			return nil, fmt.Errorf("operation already in progress")

		case IdempotencyStatusExpired:
			// Treat as new operation
			s.RetryRecord(key)
		}
	}

	// Execute the operation
	s.logger.Info("Executing idempotent operation", "key", key, "operation", operation)

	result, execErr := op(ctx)

	if execErr != nil {
		s.UpdateRecord(key, IdempotencyStatusFailed, nil, execErr)
		return nil, execErr
	}

	s.UpdateRecord(key, IdempotencyStatusCompleted, result, nil)
	return result, nil
}

// GetStats returns statistics about the idempotency service
func (s *IdempotencyService) GetStats() map[string]interface{} {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	stats := make(map[string]int)
	totalRecords := len(s.cache)

	for _, record := range s.cache {
		stats[string(record.Status)]++
	}

	return map[string]interface{}{
		"total_records":    totalRecords,
		"status_breakdown": stats,
		"ttl_hours":        s.ttl.Hours(),
	}
}

// IdempotentTransferBuilder wraps transfer building with idempotency
type IdempotentTransferBuilder struct {
	client      *Client
	idempotency *IdempotencyService
}

// NewIdempotentTransferBuilder creates a new idempotent transfer builder
func NewIdempotentTransferBuilder(client *Client, idempotency *IdempotencyService) *IdempotentTransferBuilder {
	return &IdempotentTransferBuilder{
		client:      client,
		idempotency: idempotency,
	}
}

// BuildTransferIdempotent builds a transfer with idempotency guarantees
func (itb *IdempotentTransferBuilder) BuildTransferIdempotent(ctx context.Context, walletID, coin string, req BuildTransferRequest) (*BuildTransferResponse, error) {
	// Use provided sequence ID or generate idempotency key
	key := req.SequenceId
	if key == "" {
		key = itb.idempotency.GenerateKey(fmt.Sprintf("build-transfer-%s-%s", walletID, coin), req)
		req.SequenceId = key
	}

	operation := func(ctx context.Context) (interface{}, error) {
		return itb.client.BuildTransfer(ctx, walletID, coin, req)
	}

	result, err := itb.idempotency.ExecuteIdempotent(ctx, key, "build-transfer", req, operation)
	if err != nil {
		return nil, err
	}

	return result.(*BuildTransferResponse), nil
}

// SubmitTransferIdempotent submits a transfer with idempotency guarantees
func (itb *IdempotentTransferBuilder) SubmitTransferIdempotent(ctx context.Context, walletID, coin string, req SubmitTransferRequest) (*SubmitTransferResponse, error) {
	// Generate idempotency key based on transaction hex
	key := itb.idempotency.GenerateKey(fmt.Sprintf("submit-transfer-%s-%s", walletID, coin), req)

	operation := func(ctx context.Context) (interface{}, error) {
		return itb.client.SubmitTransfer(ctx, walletID, coin, req)
	}

	result, err := itb.idempotency.ExecuteIdempotent(ctx, key, "submit-transfer", req, operation)
	if err != nil {
		return nil, err
	}

	return result.(*SubmitTransferResponse), nil
}
