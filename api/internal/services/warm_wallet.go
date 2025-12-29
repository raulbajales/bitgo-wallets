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

// WarmWalletService handles warm wallet specific operations
type WarmWalletService struct {
	bitgoClient     *bitgo.Client
	walletRepo      repository.WalletRepository
	transferRepo    repository.TransferRequestRepository
	notificationSvc NotificationService
	logger          Logger
	config          WarmWalletConfig
}

// WarmWalletConfig contains configuration for warm wallet operations
type WarmWalletConfig struct {
	// Validation settings
	MaxDailyTransferLimit  string   `json:"maxDailyTransferLimit"`
	MaxSingleTransferLimit string   `json:"maxSingleTransferLimit"`
	AllowedAddressPatterns []string `json:"allowedAddressPatterns"`
	RequiredApprovals      int      `json:"requiredApprovals"`
	ApprovalTimeoutHours   int      `json:"approvalTimeoutHours"`

	// SLA settings (faster than cold)
	InitialResponseSLA time.Duration `json:"initialResponseSLA"`
	ProcessingSLA      time.Duration `json:"processingSLA"`
	CompletionSLA      time.Duration `json:"completionSLA"`

	// Automated workflow settings
	AutoProcessThreshold  string        `json:"autoProcessThreshold"`
	ManualReviewThreshold string        `json:"manualReviewThreshold"`
	RiskScoringEnabled    bool          `json:"riskScoringEnabled"`
	MaxRiskScore          float64       `json:"maxRiskScore"`
	VelocityCheckEnabled  bool          `json:"velocityCheckEnabled"`
	EscalationThreshold   time.Duration `json:"escalationThreshold"`
}

// DefaultWarmWalletConfig returns sensible defaults for warm wallet operations
func DefaultWarmWalletConfig() WarmWalletConfig {
	return WarmWalletConfig{
		MaxDailyTransferLimit:  "100.0",          // 100 BTC or equivalent (higher than cold)
		MaxSingleTransferLimit: "25.0",           // 25 BTC or equivalent (higher than cold)
		AllowedAddressPatterns: []string{},       // Empty = no restrictions
		RequiredApprovals:      1,                // Only 1 approval needed for warm
		ApprovalTimeoutHours:   24,               // 1 day (faster than cold)
		InitialResponseSLA:     15 * time.Minute, // 15 minutes for initial response
		ProcessingSLA:          2 * time.Hour,    // 2 hours for processing
		CompletionSLA:          12 * time.Hour,   // 12 hours total completion
		AutoProcessThreshold:   "5.0",            // Auto-process up to 5 BTC
		ManualReviewThreshold:  "10.0",           // Manual review for 10+ BTC
		RiskScoringEnabled:     true,             // Enable risk scoring
		MaxRiskScore:           0.7,              // Max acceptable risk score
		VelocityCheckEnabled:   true,             // Enable velocity checks
		EscalationThreshold:    6 * time.Hour,    // Escalate after 6 hours
	}
}

// WarmTransferRequest represents a warm storage transfer request
type WarmTransferRequest struct {
	WalletID         uuid.UUID `json:"walletId"`
	RecipientAddress string    `json:"recipientAddress"`
	AmountString     string    `json:"amountString"`
	Coin             string    `json:"coin"`
	BusinessPurpose  string    `json:"businessPurpose"`
	RequestorName    string    `json:"requestorName"`
	RequestorEmail   string    `json:"requestorEmail"`
	UrgencyLevel     string    `json:"urgencyLevel"`
	Memo             string    `json:"memo,omitempty"`
	AutoProcess      bool      `json:"autoProcess,omitempty"` // Allow automatic processing
}

// WarmTransferValidationError represents validation errors for warm transfers
type WarmTransferValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e WarmTransferValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// RiskAssessmentResult represents the result of risk assessment
type RiskAssessmentResult struct {
	Score       float64           `json:"score"`
	Factors     map[string]string `json:"factors"`
	Approved    bool              `json:"approved"`
	Reason      string            `json:"reason"`
	Suggestions []string          `json:"suggestions"`
}

// AutomatedWorkflowState represents the state of automated warm wallet workflows
type AutomatedWorkflowState string

const (
	AutomatedStateSubmitted      AutomatedWorkflowState = "submitted"
	AutomatedStateRiskAssessment AutomatedWorkflowState = "risk_assessment"
	AutomatedStateVelocityCheck  AutomatedWorkflowState = "velocity_check"
	AutomatedStateAutoApproved   AutomatedWorkflowState = "auto_approved"
	AutomatedStateManualReview   AutomatedWorkflowState = "manual_review"
	AutomatedStatePendingSign    AutomatedWorkflowState = "pending_sign"
	AutomatedStateExecuting      AutomatedWorkflowState = "executing"
	AutomatedStateCompleted      AutomatedWorkflowState = "completed"
	AutomatedStateRejected       AutomatedWorkflowState = "rejected"
)

// NewWarmWalletService creates a new warm wallet service
func NewWarmWalletService(
	bitgoClient *bitgo.Client,
	walletRepo repository.WalletRepository,
	transferRepo repository.TransferRequestRepository,
	notificationSvc NotificationService,
	logger Logger,
	config WarmWalletConfig,
) *WarmWalletService {
	return &WarmWalletService{
		bitgoClient:     bitgoClient,
		walletRepo:      walletRepo,
		transferRepo:    transferRepo,
		notificationSvc: notificationSvc,
		logger:          logger,
		config:          config,
	}
}

// ValidateWarmTransferRequest performs comprehensive validation for warm transfers
func (wws *WarmWalletService) ValidateWarmTransferRequest(ctx context.Context, request WarmTransferRequest) []WarmTransferValidationError {
	var errors []WarmTransferValidationError

	// Validate wallet exists and is warm type
	wallet, err := wws.walletRepo.GetByID(request.WalletID)
	if err != nil {
		errors = append(errors, WarmTransferValidationError{
			Field:   "walletId",
			Message: "Wallet not found",
		})
		return errors
	}

	if wallet.WalletType != models.WalletTypeWarm {
		errors = append(errors, WarmTransferValidationError{
			Field:   "walletId",
			Message: "Wallet is not a warm storage wallet",
		})
	}

	// Validate recipient address format and allowlist
	if err := wws.validateRecipientAddress(request.RecipientAddress, request.Coin); err != nil {
		errors = append(errors, WarmTransferValidationError{
			Field:   "recipientAddress",
			Message: err.Error(),
		})
	}

	// Validate transfer amounts
	if err := wws.validateTransferAmount(request.AmountString, request.Coin, wallet); err != nil {
		errors = append(errors, WarmTransferValidationError{
			Field:   "amountString",
			Message: err.Error(),
		})
	}

	// Business purpose is less strict for warm wallets but still recommended
	if strings.TrimSpace(request.BusinessPurpose) == "" && wws.requiresManualReview(request.AmountString) {
		errors = append(errors, WarmTransferValidationError{
			Field:   "businessPurpose",
			Message: "Business purpose is required for high-value warm storage transfers",
		})
	}

	// Validate requestor information (less strict than cold)
	if strings.TrimSpace(request.RequestorName) == "" {
		errors = append(errors, WarmTransferValidationError{
			Field:   "requestorName",
			Message: "Requestor name is required",
		})
	}

	if !wws.isValidEmail(request.RequestorEmail) {
		errors = append(errors, WarmTransferValidationError{
			Field:   "requestorEmail",
			Message: "Valid requestor email is required",
		})
	}

	// Validate urgency level
	validUrgencyLevels := []string{"low", "normal", "high", "critical"}
	if !wws.contains(validUrgencyLevels, request.UrgencyLevel) {
		errors = append(errors, WarmTransferValidationError{
			Field:   "urgencyLevel",
			Message: "Urgency level must be one of: low, normal, high, critical",
		})
	}

	return errors
}

// CreateWarmTransferRequest creates a new warm storage transfer request with automated processing
func (wws *WarmWalletService) CreateWarmTransferRequest(ctx context.Context, request WarmTransferRequest, requestedBy uuid.UUID) (*models.TransferRequest, error) {
	// Validate the request
	validationErrors := wws.ValidateWarmTransferRequest(ctx, request)
	if len(validationErrors) > 0 {
		return nil, fmt.Errorf("validation failed: %v", validationErrors)
	}

	// Perform risk assessment
	riskResult, err := wws.assessTransferRisk(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("risk assessment failed: %w", err)
	}

	// Determine required approvals based on risk and amount
	requiredApprovals := wws.calculateRequiredApprovals(request.AmountString, riskResult.Score)

	// Create transfer request with warm-specific settings
	transferRequest := &models.TransferRequest{
		WalletID:          request.WalletID,
		RequestedByUserID: requestedBy,
		RecipientAddress:  request.RecipientAddress,
		AmountString:      request.AmountString,
		Coin:              request.Coin,
		TransferType:      models.WalletTypeWarm,
		Status:            models.TransferStatusSubmitted,
		RequiredApprovals: requiredApprovals,
		ReceivedApprovals: 0,
		Memo:              &request.Memo,
	}

	// Create the transfer request in the database
	if err := wws.transferRepo.Create(transferRequest); err != nil {
		return nil, fmt.Errorf("failed to create warm transfer request: %w", err)
	}

	// Start automated processing if eligible
	if wws.canAutoProcess(request.AmountString, riskResult.Score) && request.AutoProcess {
		go wws.processAutomatedTransfer(ctx, transferRequest, riskResult)
	} else {
		// Send notifications for manual review
		wws.notifyWarmTransferCreated(transferRequest, request, riskResult)
	}

	// Log the creation
	wws.logger.Info("Warm transfer request created",
		"transfer_id", transferRequest.ID,
		"wallet_id", request.WalletID,
		"amount", request.AmountString,
		"coin", request.Coin,
		"risk_score", riskResult.Score,
		"auto_process", request.AutoProcess,
		"urgency", request.UrgencyLevel,
	)

	return transferRequest, nil
}

// ProcessAutomatedTransfer handles automated processing for eligible warm transfers
func (wws *WarmWalletService) processAutomatedTransfer(ctx context.Context, transfer *models.TransferRequest, riskResult *RiskAssessmentResult) {
	wws.logger.Info("Starting automated processing for warm transfer",
		"transfer_id", transfer.ID,
		"risk_score", riskResult.Score,
	)

	// Update status to auto-approved
	transfer.Status = models.TransferStatusApproved
	transfer.ReceivedApprovals = transfer.RequiredApprovals
	if err := wws.transferRepo.Update(transfer); err != nil {
		wws.logger.Error("Failed to update transfer status", "error", err)
		return
	}

	// In a real implementation, this would trigger the actual BitGo transfer
	// For now, we'll simulate the process
	time.Sleep(2 * time.Second) // Simulate processing time

	// Update to signed status
	transfer.Status = models.TransferStatusSigned
	if err := wws.transferRepo.Update(transfer); err != nil {
		wws.logger.Error("Failed to update transfer to signed", "error", err)
		return
	}

	// Simulate broadcast
	time.Sleep(1 * time.Second)
	transfer.Status = models.TransferStatusBroadcast
	if err := wws.transferRepo.Update(transfer); err != nil {
		wws.logger.Error("Failed to update transfer to broadcast", "error", err)
		return
	}

	wws.logger.Info("Automated warm transfer processing completed",
		"transfer_id", transfer.ID,
	)
}

// AssessTransferRisk performs risk assessment for warm transfers
func (wws *WarmWalletService) assessTransferRisk(ctx context.Context, request WarmTransferRequest) (*RiskAssessmentResult, error) {
	result := &RiskAssessmentResult{
		Factors: make(map[string]string),
		Score:   0.0,
	}

	// Amount-based risk scoring
	amount, err := parseAmount(request.AmountString)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}

	if amount > 10.0 {
		result.Score += 0.3
		result.Factors["high_amount"] = "Transfer amount is above 10.0"
	}

	// Address reputation check (simplified)
	if wws.isHighRiskAddress(request.RecipientAddress) {
		result.Score += 0.5
		result.Factors["high_risk_address"] = "Recipient address flagged as high risk"
	}

	// Velocity check
	if wws.config.VelocityCheckEnabled {
		velocityRisk, err := wws.checkTransferVelocity(ctx, request.WalletID, amount)
		if err == nil && velocityRisk > 0 {
			result.Score += velocityRisk
			result.Factors["velocity_risk"] = fmt.Sprintf("High transfer velocity detected (score: %.2f)", velocityRisk)
		}
	}

	// Urgency-based risk adjustment
	if request.UrgencyLevel == "critical" {
		result.Score += 0.1
		result.Factors["urgent_request"] = "Critical urgency level increases risk"
	}

	// Final assessment
	result.Approved = result.Score <= wws.config.MaxRiskScore
	if result.Approved {
		result.Reason = "Risk score within acceptable limits"
	} else {
		result.Reason = fmt.Sprintf("Risk score (%.2f) exceeds maximum allowed (%.2f)", result.Score, wws.config.MaxRiskScore)
		result.Suggestions = []string{
			"Consider manual review",
			"Verify recipient address",
			"Check transfer purpose",
		}
	}

	return result, nil
}

// GetWarmTransfersSLAStatus returns SLA status for warm transfers
func (wws *WarmWalletService) GetWarmTransfersSLAStatus(ctx context.Context) (map[string]interface{}, error) {
	// Get all warm transfers in progress
	warmStatuses := []models.TransferStatus{
		models.TransferStatusSubmitted,
		models.TransferStatusPendingApproval,
		models.TransferStatusApproved,
		models.TransferStatusSigned,
	}

	transfers, err := wws.transferRepo.GetTransfersByStatuses(warmStatuses, 1000)
	if err != nil {
		return nil, fmt.Errorf("failed to get warm transfers: %w", err)
	}

	// Filter only warm transfers
	warmTransfers := make([]*models.TransferRequest, 0)
	for _, transfer := range transfers {
		if transfer.TransferType == models.WalletTypeWarm {
			warmTransfers = append(warmTransfers, transfer)
		}
	}

	now := time.Now()
	slaBreached := 0
	atRisk := 0
	escalated := 0
	automated := 0

	for _, transfer := range warmTransfers {
		// Calculate time since creation
		elapsed := now.Sub(transfer.CreatedAt)

		// Check SLA status
		if elapsed > wws.config.CompletionSLA {
			slaBreached++
		} else if elapsed > wws.config.CompletionSLA/2 {
			atRisk++
		}

		// Check if escalated
		if elapsed > wws.config.EscalationThreshold {
			escalated++
		}

		// Check if automatically processed
		if transfer.ReceivedApprovals == transfer.RequiredApprovals && transfer.RequiredApprovals == 0 {
			automated++
		}
	}

	return map[string]interface{}{
		"totalWarmTransfers": len(warmTransfers),
		"slaBreached":        slaBreached,
		"atRisk":             atRisk,
		"escalated":          escalated,
		"automated":          automated,
		"automationRate":     float64(automated) / float64(len(warmTransfers)) * 100,
		"config": map[string]interface{}{
			"initialResponseSLA": wws.config.InitialResponseSLA.String(),
			"processingSLA":      wws.config.ProcessingSLA.String(),
			"completionSLA":      wws.config.CompletionSLA.String(),
		},
	}, nil
}

// Helper methods

func (wws *WarmWalletService) validateRecipientAddress(address, coin string) error {
	if strings.TrimSpace(address) == "" {
		return fmt.Errorf("recipient address is required")
	}

	// Check allowlist if configured
	if len(wws.config.AllowedAddressPatterns) > 0 {
		allowed := false
		for _, pattern := range wws.config.AllowedAddressPatterns {
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

func (wws *WarmWalletService) validateTransferAmount(amountStr, coin string, wallet *models.Wallet) error {
	// Parse amount
	amount, err := parseAmount(amountStr)
	if err != nil {
		return fmt.Errorf("invalid amount format")
	}

	if amount <= 0 {
		return fmt.Errorf("amount must be greater than zero")
	}

	// Check against limits
	maxSingle, _ := parseAmount(wws.config.MaxSingleTransferLimit)
	if amount > maxSingle {
		return fmt.Errorf("amount exceeds single transfer limit of %s %s", wws.config.MaxSingleTransferLimit, coin)
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

func (wws *WarmWalletService) canAutoProcess(amountStr string, riskScore float64) bool {
	amount, err := parseAmount(amountStr)
	if err != nil {
		return false
	}

	threshold, err := parseAmount(wws.config.AutoProcessThreshold)
	if err != nil {
		return false
	}

	return amount <= threshold && riskScore <= wws.config.MaxRiskScore
}

func (wws *WarmWalletService) requiresManualReview(amountStr string) bool {
	amount, err := parseAmount(amountStr)
	if err != nil {
		return true // Default to manual review on parsing error
	}

	threshold, err := parseAmount(wws.config.ManualReviewThreshold)
	if err != nil {
		return true
	}

	return amount >= threshold
}

func (wws *WarmWalletService) calculateRequiredApprovals(amountStr string, riskScore float64) int {
	amount, err := parseAmount(amountStr)
	if err != nil {
		return wws.config.RequiredApprovals
	}

	// Higher amounts or risk scores require more approvals
	if amount > 50.0 || riskScore > 0.8 {
		return 2
	} else if amount > 20.0 || riskScore > 0.5 {
		return 1
	}

	return 0 // Can be auto-processed
}

func (wws *WarmWalletService) isHighRiskAddress(address string) bool {
	// In a real implementation, this would check against known bad addresses
	// For now, just a simple mock
	highRiskPrefixes := []string{"1BadAddr", "0xBadAddr"}
	for _, prefix := range highRiskPrefixes {
		if strings.HasPrefix(address, prefix) {
			return true
		}
	}
	return false
}

func (wws *WarmWalletService) checkTransferVelocity(ctx context.Context, walletID uuid.UUID, amount float64) (float64, error) {
	// Get recent transfers for this wallet (last 24 hours)
	// This is a simplified implementation
	// In reality, you'd query the database for recent transfers

	// Mock velocity check
	if amount > 20.0 {
		return 0.2, nil // Some velocity risk for large amounts
	}
	return 0.0, nil
}

func (wws *WarmWalletService) isValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

func (wws *WarmWalletService) contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func (wws *WarmWalletService) notifyWarmTransferCreated(transfer *models.TransferRequest, request WarmTransferRequest, riskResult *RiskAssessmentResult) {
	// Send notification about new warm transfer
	wws.notificationSvc.SendTransferCreatedNotification(transfer)

	// Additional warm-specific notifications would go here
	// e.g., risk alerts if high risk score
	if riskResult.Score > 0.5 {
		wws.logger.Warn("High risk warm transfer created",
			"transfer_id", transfer.ID,
			"risk_score", riskResult.Score,
			"factors", riskResult.Factors,
		)
	}
}
