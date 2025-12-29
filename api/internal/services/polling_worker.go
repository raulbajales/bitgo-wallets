package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"bitgo-wallets-api/internal/bitgo"
	"bitgo-wallets-api/internal/models"
	"bitgo-wallets-api/internal/repository"
)

// Logger interface for the worker service
type Logger interface {
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
	Debug(msg string, fields ...interface{})
}

// PollingWorkerConfig configures the polling worker
type PollingWorkerConfig struct {
	PollInterval      time.Duration // How often to poll for updates
	BatchSize         int           // Number of transfers to process per batch
	MaxRetries        int           // Max retries for failed polling attempts
	StaleThreshold    time.Duration // How old a transfer can be before considered stale
	ConcurrentWorkers int           // Number of concurrent workers
	ShutdownTimeout   time.Duration // Timeout for graceful shutdown
}

// DefaultPollingWorkerConfig returns sensible defaults
func DefaultPollingWorkerConfig() PollingWorkerConfig {
	return PollingWorkerConfig{
		PollInterval:      30 * time.Second,
		BatchSize:         50,
		MaxRetries:        3,
		StaleThreshold:    24 * time.Hour,
		ConcurrentWorkers: 3,
		ShutdownTimeout:   30 * time.Second,
	}
}

// TransferPollingWorker polls BitGo for transfer status updates
type TransferPollingWorker struct {
	config          PollingWorkerConfig
	logger          Logger
	bitgoClient     *bitgo.Client
	approvalService *bitgo.ApprovalService
	transferRepo    repository.TransferRequestRepository
	walletRepo      repository.WalletRepository
	notificationSvc NotificationService

	// Control channels
	ctx       context.Context
	cancel    context.CancelFunc
	shutdown  chan struct{}
	stopped   chan struct{}
	wg        sync.WaitGroup
	isRunning bool
	mu        sync.RWMutex
}

// NewTransferPollingWorker creates a new polling worker
func NewTransferPollingWorker(
	config PollingWorkerConfig,
	logger Logger,
	bitgoClient *bitgo.Client,
	transferRepo repository.TransferRequestRepository,
	walletRepo repository.WalletRepository,
	notificationSvc NotificationService,
) *TransferPollingWorker {
	ctx, cancel := context.WithCancel(context.Background())

	approvalService := bitgo.NewApprovalService(bitgoClient, logger)

	return &TransferPollingWorker{
		config:          config,
		logger:          logger,
		bitgoClient:     bitgoClient,
		approvalService: approvalService,
		transferRepo:    transferRepo,
		walletRepo:      walletRepo,
		notificationSvc: notificationSvc,
		ctx:             ctx,
		cancel:          cancel,
		shutdown:        make(chan struct{}),
		stopped:         make(chan struct{}),
	}
}

// Start begins the polling worker
func (w *TransferPollingWorker) Start() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.isRunning {
		return fmt.Errorf("worker is already running")
	}

	w.isRunning = true
	w.logger.Info("Starting transfer polling worker",
		"poll_interval", w.config.PollInterval,
		"batch_size", w.config.BatchSize,
		"concurrent_workers", w.config.ConcurrentWorkers,
	)

	// Start main polling loop
	w.wg.Add(1)
	go w.pollingLoop()

	// Start concurrent worker goroutines
	for i := 0; i < w.config.ConcurrentWorkers; i++ {
		w.wg.Add(1)
		go w.worker(i)
	}

	return nil
}

// Stop gracefully stops the polling worker
func (w *TransferPollingWorker) Stop() error {
	w.mu.Lock()
	if !w.isRunning {
		w.mu.Unlock()
		return fmt.Errorf("worker is not running")
	}
	w.isRunning = false
	w.mu.Unlock()

	w.logger.Info("Stopping transfer polling worker")

	// Signal shutdown
	close(w.shutdown)
	w.cancel()

	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		w.logger.Info("Transfer polling worker stopped gracefully")
	case <-time.After(w.config.ShutdownTimeout):
		w.logger.Warn("Transfer polling worker shutdown timed out")
	}

	close(w.stopped)
	return nil
}

// IsRunning returns whether the worker is currently running
func (w *TransferPollingWorker) IsRunning() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.isRunning
}

// pollingLoop is the main polling loop
func (w *TransferPollingWorker) pollingLoop() {
	defer w.wg.Done()

	ticker := time.NewTicker(w.config.PollInterval)
	defer ticker.Stop()

	// Run initial poll
	w.pollTransfers()

	for {
		select {
		case <-ticker.C:
			w.pollTransfers()
		case <-w.shutdown:
			w.logger.Info("Polling loop shutting down")
			return
		case <-w.ctx.Done():
			w.logger.Info("Polling loop context cancelled")
			return
		}
	}
}

// pollTransfers gets transfers that need status updates
func (w *TransferPollingWorker) pollTransfers() {
	// Get transfers that need polling
	statuses := []models.TransferStatus{
		models.TransferStatusSubmitted,
		models.TransferStatusPendingApproval,
		models.TransferStatusApproved,
		models.TransferStatusSigned,
		models.TransferStatusBroadcast,
	}

	transfers, err := w.transferRepo.GetTransfersByStatuses(statuses, w.config.BatchSize)
	if err != nil {
		w.logger.Error("Failed to get transfers for polling", "error", err)
		return
	}

	if len(transfers) == 0 {
		w.logger.Debug("No transfers need status polling")
		return
	}

	w.logger.Info("Found transfers to poll", "count", len(transfers))

	// Distribute transfers to workers via channel
	transferChan := make(chan *models.TransferRequest, len(transfers))
	for _, transfer := range transfers {
		transferChan <- transfer
	}
	close(transferChan)

	// Workers will process from the channel
}

// worker processes transfers from the work queue
func (w *TransferPollingWorker) worker(workerID int) {
	defer w.wg.Done()

	w.logger.Debug("Starting worker", "worker_id", workerID)

	for {
		select {
		case <-w.shutdown:
			w.logger.Debug("Worker shutting down", "worker_id", workerID)
			return
		case <-w.ctx.Done():
			w.logger.Debug("Worker context cancelled", "worker_id", workerID)
			return
		default:
			// This would normally read from a work channel
			// For now, just sleep to avoid busy waiting
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// processTransfer handles status polling for a single transfer
func (w *TransferPollingWorker) processTransfer(transfer *models.TransferRequest) {
	ctx, cancel := context.WithTimeout(w.ctx, 30*time.Second)
	defer cancel()

	w.logger.Debug("Processing transfer",
		"transfer_id", transfer.ID,
		"current_status", transfer.Status,
		"bitgo_transfer_id", transfer.BitgoTransferID,
	)

	// Get wallet information
	wallet, err := w.walletRepo.GetByID(transfer.WalletID)
	if err != nil {
		w.logger.Error("Failed to get wallet for transfer",
			"transfer_id", transfer.ID,
			"wallet_id", transfer.WalletID,
			"error", err,
		)
		return
	}

	// Update transfer status based on current state
	updated, err := w.updateTransferStatus(ctx, transfer, wallet)
	if err != nil {
		w.logger.Error("Failed to update transfer status",
			"transfer_id", transfer.ID,
			"error", err,
		)
		return
	}

	// Check for pending approvals if needed
	if transfer.Status == models.TransferStatusPendingApproval {
		w.checkPendingApprovals(ctx, transfer, wallet)
	}

	if updated {
		w.logger.Info("Transfer status updated",
			"transfer_id", transfer.ID,
			"old_status", transfer.Status,
			"new_status", transfer.Status,
		)
	}
}

// updateTransferStatus checks and updates transfer status from BitGo
func (w *TransferPollingWorker) updateTransferStatus(ctx context.Context, transfer *models.TransferRequest, wallet *models.Wallet) (bool, error) {
	// Only poll transfers that have been submitted to BitGo
	if transfer.BitgoTransferID == nil {
		return false, nil
	}

	// Get transfer status from BitGo
	bitgoTransfer, err := w.bitgoClient.GetTransfer(ctx, wallet.BitgoWalletID, wallet.Coin, *transfer.BitgoTransferID)
	if err != nil {
		return false, fmt.Errorf("failed to get BitGo transfer: %w", err)
	}

	// Normalize status using status mapper
	statusMapper := bitgo.NewStatusMapper()
	canonicalStatus := statusMapper.NormalizeTransferStatus(bitgoTransfer.State, bitgoTransfer)
	newStatus := models.TransferStatus(canonicalStatus)

	// Check if status changed
	if transfer.Status == newStatus {
		return false, nil // No change
	}

	// Update transfer with new status
	oldStatus := transfer.Status
	transfer.Status = newStatus

	// Update timestamps based on status
	now := time.Now()
	switch newStatus {
	case models.TransferStatusConfirmed:
		if transfer.CompletedAt == nil {
			transfer.CompletedAt = &now
		}
	case models.TransferStatusFailed:
		if transfer.FailedAt == nil {
			transfer.FailedAt = &now
		}
	}

	// Save to database
	if err := w.transferRepo.Update(transfer); err != nil {
		return false, fmt.Errorf("failed to update transfer in database: %w", err)
	}

	// Send notification about status change
	w.notificationSvc.SendTransferStatusNotification(transfer, oldStatus, newStatus)

	return true, nil
}

// checkPendingApprovals checks for pending approvals and sends notifications
func (w *TransferPollingWorker) checkPendingApprovals(ctx context.Context, transfer *models.TransferRequest, wallet *models.Wallet) {
	if transfer.BitgoTxid == nil {
		return
	}

	// Get approval status for this transfer
	approvalStatus, err := w.approvalService.GetTransferApprovalStatus(
		ctx,
		wallet.BitgoWalletID,
		wallet.Coin,
		*transfer.BitgoTxid,
		"", // No specific user context in worker
	)
	if err != nil {
		w.logger.Error("Failed to get approval status",
			"transfer_id", transfer.ID,
			"error", err,
		)
		return
	}

	if approvalStatus == nil {
		return // No pending approval
	}

	// Send pending approval notifications
	w.notificationSvc.SendPendingApprovalNotification(transfer, approvalStatus)

	w.logger.Info("Checked pending approvals",
		"transfer_id", transfer.ID,
		"approval_id", approvalStatus.ID,
		"required_approvals", approvalStatus.RequiredApprovals,
		"received_approvals", approvalStatus.ReceivedApprovals,
	)
}

// GetStats returns worker statistics
func (w *TransferPollingWorker) GetStats() map[string]interface{} {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return map[string]interface{}{
		"is_running":         w.isRunning,
		"poll_interval":      w.config.PollInterval.String(),
		"batch_size":         w.config.BatchSize,
		"concurrent_workers": w.config.ConcurrentWorkers,
		"stale_threshold":    w.config.StaleThreshold.String(),
	}
}

// HealthCheck returns the health status of the worker
func (w *TransferPollingWorker) HealthCheck() map[string]interface{} {
	w.mu.RLock()
	isRunning := w.isRunning
	w.mu.RUnlock()

	status := "stopped"
	if isRunning {
		status = "running"
	}

	return map[string]interface{}{
		"status":        status,
		"last_check":    time.Now().UTC(),
		"configuration": w.GetStats(),
	}
}
