package services

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"bitgo-wallets-api/internal/bitgo"
	"bitgo-wallets-api/internal/models"
	"bitgo-wallets-api/internal/repository"

	"github.com/google/uuid"
)

// ColdWalletService handles cold wallet specific operations
type ColdWalletService struct {
	bitgoClient     *bitgo.Client
	walletRepo      repository.WalletRepository
	transferRepo    repository.TransferRequestRepository
	notificationSvc NotificationService
	logger          Logger
	config          ColdWalletConfig
}

// ColdWalletConfig contains configuration for cold wallet operations
type ColdWalletConfig struct {
	// Validation settings
	MaxDailyTransferLimit  string   `json:"maxDailyTransferLimit"`
	MaxSingleTransferLimit string   `json:"maxSingleTransferLimit"`
	AllowedAddressPatterns []string `json:"allowedAddressPatterns"`
	RequiredApprovals      int      `json:"requiredApprovals"`
	ApprovalTimeoutHours   int      `json:"approvalTimeoutHours"`

	// SLA settings
	InitialResponseSLA time.Duration `json:"initialResponseSLA"`
	ProcessingSLA      time.Duration `json:"processingSLA"`
	CompletionSLA      time.Duration `json:"completionSLA"`

	// Offline workflow settings
	ManualReviewThreshold    string        `json:"manualReviewThreshold"`
	OperatorNotificationList []string      `json:"operatorNotificationList"`
	EscalationThreshold      time.Duration `json:"escalationThreshold"`
}

// DefaultColdWalletConfig returns sensible defaults for cold wallet operations
func DefaultColdWalletConfig() ColdWalletConfig {
	return ColdWalletConfig{
		MaxDailyTransferLimit:  "10.0",         // 10 BTC or equivalent
		MaxSingleTransferLimit: "5.0",          // 5 BTC or equivalent
		AllowedAddressPatterns: []string{},     // Empty = no restrictions
		RequiredApprovals:      3,              // Minimum 3 approvals
		ApprovalTimeoutHours:   72,             // 3 days
		InitialResponseSLA:     2 * time.Hour,  // 2 hours for initial response
		ProcessingSLA:          24 * time.Hour, // 24 hours for processing
		CompletionSLA:          72 * time.Hour, // 72 hours total completion
		ManualReviewThreshold:  "1.0",          // Manual review for 1+ BTC
		EscalationThreshold:    48 * time.Hour, // Escalate after 48 hours
	}
}

// ColdTransferRequest represents a cold storage transfer request
type ColdTransferRequest struct {
	WalletID         uuid.UUID `json:"walletId"`
	RecipientAddress string    `json:"recipientAddress"`
	AmountString     string    `json:"amountString"`
	Coin             string    `json:"coin"`
	BusinessPurpose  string    `json:"businessPurpose"`
	RequestorName    string    `json:"requestorName"`
	RequestorEmail   string    `json:"requestorEmail"`
	UrgencyLevel     string    `json:"urgencyLevel"`
	Memo             string    `json:"memo,omitempty"`
}

// ColdTransferValidationError represents validation errors for cold transfers
type ColdTransferValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e ColdTransferValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// OfflineWorkflowState represents the state of offline custody workflows
type OfflineWorkflowState string

const (
	OfflineStateSubmitted        OfflineWorkflowState = "submitted"
	OfflineStateSecurityReview   OfflineWorkflowState = "security_review"
	OfflineStateComplianceCheck  OfflineWorkflowState = "compliance_check"
	OfflineStateOperatorQueued   OfflineWorkflowState = "operator_queued"
	OfflineStateManualProcessing OfflineWorkflowState = "manual_processing"
	OfflineStateAwaitingHSM      OfflineWorkflowState = "awaiting_hsm"
	OfflineStateReadyToExecute   OfflineWorkflowState = "ready_to_execute"
	OfflineStateExecuted         OfflineWorkflowState = "executed"
	OfflineStateEscalated        OfflineWorkflowState = "escalated"
)

// NewColdWalletService creates a new cold wallet service
func NewColdWalletService(
	bitgoClient *bitgo.Client,
	walletRepo repository.WalletRepository,
	transferRepo repository.TransferRequestRepository,
	notificationSvc NotificationService,
	logger Logger,
	config ColdWalletConfig,
) *ColdWalletService {
	return &ColdWalletService{
		bitgoClient:     bitgoClient,
		walletRepo:      walletRepo,
		transferRepo:    transferRepo,
		notificationSvc: notificationSvc,
		logger:          logger,
		config:          config,
	}
}

// ValidateColdTransferRequest performs comprehensive validation for cold transfers
func (cws *ColdWalletService) ValidateColdTransferRequest(ctx context.Context, request ColdTransferRequest) []ColdTransferValidationError {
	var errors []ColdTransferValidationError

	// Validate wallet exists and is cold type
	wallet, err := cws.walletRepo.GetByID(request.WalletID)
	if err != nil {
		errors = append(errors, ColdTransferValidationError{
			Field:   "walletId",
			Message: "Wallet not found",
		})
		return errors
	}

	if wallet.WalletType != models.WalletTypeCold {
		errors = append(errors, ColdTransferValidationError{
			Field:   "walletId",
			Message: "Wallet is not a cold storage wallet",
		})
	}

	// Validate recipient address format and allowlist
	if err := cws.validateRecipientAddress(request.RecipientAddress, request.Coin); err != nil {
		errors = append(errors, ColdTransferValidationError{
			Field:   "recipientAddress",
			Message: err.Error(),
		})
	}

	// Validate transfer amounts
	if err := cws.validateTransferAmount(request.AmountString, request.Coin, wallet); err != nil {
		errors = append(errors, ColdTransferValidationError{
			Field:   "amountString",
			Message: err.Error(),
		})
	}

	// Validate business purpose
	if strings.TrimSpace(request.BusinessPurpose) == "" {
		errors = append(errors, ColdTransferValidationError{
			Field:   "businessPurpose",
			Message: "Business purpose is required for cold storage transfers",
		})
	}

	// Validate requestor information
	if strings.TrimSpace(request.RequestorName) == "" {
		errors = append(errors, ColdTransferValidationError{
			Field:   "requestorName",
			Message: "Requestor name is required",
		})
	}

	if !cws.isValidEmail(request.RequestorEmail) {
		errors = append(errors, ColdTransferValidationError{
			Field:   "requestorEmail",
			Message: "Valid requestor email is required",
		})
	}

	// Validate urgency level
	validUrgencyLevels := []string{"low", "normal", "high", "critical"}
	if !cws.contains(validUrgencyLevels, request.UrgencyLevel) {
		errors = append(errors, ColdTransferValidationError{
			Field:   "urgencyLevel",
			Message: "Urgency level must be one of: low, normal, high, critical",
		})
	}

	return errors
}

// CreateColdTransferRequest creates a new cold storage transfer request
func (cws *ColdWalletService) CreateColdTransferRequest(ctx context.Context, request ColdTransferRequest, requestedBy uuid.UUID) (*models.TransferRequest, error) {
	// Validate the request
	validationErrors := cws.ValidateColdTransferRequest(ctx, request)
	if len(validationErrors) > 0 {
		return nil, fmt.Errorf("validation failed: %v", validationErrors)
	}

	// Get wallet for additional checks
	wallet, err := cws.walletRepo.GetByID(request.WalletID)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet: %w", err)
	}

	// Create transfer request with cold-specific settings
	transferRequest := &models.TransferRequest{
		WalletID:          request.WalletID,
		RequestedByUserID: requestedBy,
		RecipientAddress:  request.RecipientAddress,
		AmountString:      request.AmountString,
		Coin:              request.Coin,
		TransferType:      models.WalletTypeCold,
		Status:            models.TransferStatusSubmitted,
		RequiredApprovals: cws.config.RequiredApprovals,
		ReceivedApprovals: 0,
		Memo:              &request.Memo,
	}

	// Add cold-specific metadata
	metadata := map[string]interface{}{
		"businessPurpose": request.BusinessPurpose,
		"requestorName":   request.RequestorName,
		"requestorEmail":  request.RequestorEmail,
		"urgencyLevel":    request.UrgencyLevel,
		"offlineState":    OfflineStateSubmitted,
		"slaDeadlines": map[string]time.Time{
			"initialResponse": time.Now().Add(cws.config.InitialResponseSLA),
			"processing":      time.Now().Add(cws.config.ProcessingSLA),
			"completion":      time.Now().Add(cws.config.CompletionSLA),
		},
		"requiresManualReview": cws.requiresManualReview(request.AmountString),
	}

	// Store metadata (this would be stored in a metadata field in a real implementation)
	transferRequest.Metadata = metadata

	// Create the transfer request in the database
	if err := cws.transferRepo.Create(transferRequest); err != nil {
		return nil, fmt.Errorf("failed to create cold transfer request: %w", err)
	}

	// Send notifications to operators
	cws.notifyColdTransferCreated(transferRequest, request)

	// Log the creation
	cws.logger.Info("Cold transfer request created",
		"transfer_id", transferRequest.ID,
		"wallet_id", request.WalletID,
		"amount", request.AmountString,
		"coin", request.Coin,
		"urgency", request.UrgencyLevel,
		"requires_manual_review", metadata["requiresManualReview"],
	)

	return transferRequest, nil
}

// GetColdTransfersSLAStatus returns SLA status for cold transfers
func (cws *ColdWalletService) GetColdTransfersSLAStatus(ctx context.Context) (map[string]interface{}, error) {
	// Get all cold transfers in progress
	coldStatuses := []models.TransferStatus{
		models.TransferStatusSubmitted,
		models.TransferStatusPendingApproval,
		models.TransferStatusApproved,
	}

	transfers, err := cws.transferRepo.GetTransfersByStatuses(coldStatuses, 1000)
	if err != nil {
		return nil, fmt.Errorf("failed to get cold transfers: %w", err)
	}

	// Filter only cold transfers
	coldTransfers := make([]*models.TransferRequest, 0)
	for _, transfer := range transfers {
		if transfer.TransferType == models.WalletTypeCold {
			coldTransfers = append(coldTransfers, transfer)
		}
	}

	now := time.Now()
	slaBreached := 0
	atRisk := 0
	escalated := 0

	for _, transfer := range coldTransfers {
		// Calculate time since creation
		elapsed := now.Sub(transfer.CreatedAt)

		// Check SLA status
		if elapsed > cws.config.CompletionSLA {
			slaBreached++
		} else if elapsed > cws.config.CompletionSLA/2 {
			atRisk++
		}

		// Check if escalated
		if elapsed > cws.config.EscalationThreshold {
			escalated++
		}
	}

	return map[string]interface{}{
		"totalColdTransfers": len(coldTransfers),
		"slaBreached":        slaBreached,
		"atRisk":             atRisk,
		"escalated":          escalated,
		"config": map[string]interface{}{
			"initialResponseSLA": cws.config.InitialResponseSLA.String(),
			"processingSLA":      cws.config.ProcessingSLA.String(),
			"completionSLA":      cws.config.CompletionSLA.String(),
		},
	}, nil
}

// UpdateOfflineWorkflowState updates the offline workflow state for a cold transfer
func (cws *ColdWalletService) UpdateOfflineWorkflowState(ctx context.Context, transferID uuid.UUID, newState OfflineWorkflowState, notes string) error {
	transfer, err := cws.transferRepo.GetByID(transferID)
	if err != nil {
		return fmt.Errorf("failed to get transfer: %w", err)
	}

	if transfer.TransferType != models.WalletTypeCold {
		return fmt.Errorf("transfer is not a cold storage transfer")
	}

	// Update metadata with new offline state
	if transfer.Metadata == nil {
		transfer.Metadata = make(map[string]interface{})
	}

	metadata := transfer.Metadata.(map[string]interface{})
	metadata["offlineState"] = newState
	metadata["stateUpdatedAt"] = time.Now()
	if notes != "" {
		metadata["stateNotes"] = notes
	}

	// Update corresponding transfer status
	switch newState {
	case OfflineStateSecurityReview, OfflineStateComplianceCheck:
		transfer.Status = models.TransferStatusPendingApproval
	case OfflineStateOperatorQueued, OfflineStateManualProcessing:
		transfer.Status = models.TransferStatusApproved
	case OfflineStateAwaitingHSM, OfflineStateReadyToExecute:
		transfer.Status = models.TransferStatusSigned
	case OfflineStateExecuted:
		transfer.Status = models.TransferStatusBroadcast
	case OfflineStateEscalated:
		// Keep current status but mark as escalated
		metadata["escalated"] = true
		metadata["escalatedAt"] = time.Now()
	}

	if err := cws.transferRepo.Update(transfer); err != nil {
		return fmt.Errorf("failed to update transfer: %w", err)
	}

	cws.logger.Info("Cold transfer offline state updated",
		"transfer_id", transferID,
		"old_state", metadata["offlineState"],
		"new_state", newState,
		"notes", notes,
	)

	return nil
}

// Helper methods

func (cws *ColdWalletService) validateRecipientAddress(address, coin string) error {
	if strings.TrimSpace(address) == "" {
		return fmt.Errorf("recipient address is required")
	}

	// Check allowlist if configured
	if len(cws.config.AllowedAddressPatterns) > 0 {
		allowed := false
		for _, pattern := range cws.config.AllowedAddressPatterns {
			if matched, _ := regexp.MatchString(pattern, address); matched {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("recipient address not in allowlist")
		}
	}

	// Basic format validation (simplified)
	switch strings.ToLower(coin) {
	case "btc", "tbtc":
		if len(address) < 26 || len(address) > 62 {
			return fmt.Errorf("invalid Bitcoin address format")
		}
	case "eth":
		if len(address) != 42 || !strings.HasPrefix(address, "0x") {
			return fmt.Errorf("invalid Ethereum address format")
		}
	}

	return nil
}

func (cws *ColdWalletService) validateTransferAmount(amountStr, coin string, wallet *models.Wallet) error {
	// Parse amount
	amount, err := parseAmount(amountStr)
	if err != nil {
		return fmt.Errorf("invalid amount format")
	}

	if amount <= 0 {
		return fmt.Errorf("amount must be greater than zero")
	}

	// Check against limits
	maxSingle, _ := parseAmount(cws.config.MaxSingleTransferLimit)
	if amount > maxSingle {
		return fmt.Errorf("amount exceeds single transfer limit of %s %s", cws.config.MaxSingleTransferLimit, coin)
	}

	// Check spendable balance
	spendableBalance, err := parseAmount(wallet.SpendableBalanceString)
	if err != nil {
		return fmt.Errorf("unable to verify wallet balance")
	}

	if amount > spendableBalance {
		return fmt.Errorf("amount exceeds spendable balance of %s %s", wallet.SpendableBalanceString, coin)
	}

	return nil
}

func (cws *ColdWalletService) requiresManualReview(amountStr string) bool {
	amount, err := parseAmount(amountStr)
	if err != nil {
		return true // Default to manual review on parsing error
	}

	threshold, err := parseAmount(cws.config.ManualReviewThreshold)
	if err != nil {
		return true
	}

	return amount >= threshold
}

func (cws *ColdWalletService) isValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

func (cws *ColdWalletService) contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func (cws *ColdWalletService) notifyColdTransferCreated(transfer *models.TransferRequest, request ColdTransferRequest) {
	// Send notification to operators about new cold transfer
	cws.notificationSvc.SendTransferCreatedNotification(transfer)

	// Additional cold-specific notifications would go here
	// e.g., email to compliance team, Slack to operations channel
}

// parseAmount is a simple amount parser - in production, use decimal library
func parseAmount(amountStr string) (float64, error) {
	// This is a simplified implementation
	// In production, use shopspring/decimal or similar for precise decimal handling
	var amount float64
	_, err := fmt.Sscanf(amountStr, "%f", &amount)
	return amount, err
}
