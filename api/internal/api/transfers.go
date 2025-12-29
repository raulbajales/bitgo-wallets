package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"bitgo-wallets-api/internal/bitgo"
	"bitgo-wallets-api/internal/models"
	"bitgo-wallets-api/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CreateTransferRequest struct {
	RecipientAddress string            `json:"recipient_address" binding:"required"`
	AmountString     string            `json:"amount_string" binding:"required"`
	Coin             string            `json:"coin" binding:"required"`
	TransferType     models.WalletType `json:"transfer_type" binding:"required"`
	Memo             *string           `json:"memo"`

	// Additional fields for warm/cold transfers
	BusinessPurpose string `json:"business_purpose,omitempty"`
	RequestorName   string `json:"requestor_name,omitempty"`
	RequestorEmail  string `json:"requestor_email,omitempty"`
	UrgencyLevel    string `json:"urgency_level,omitempty"`
	AutoProcess     bool   `json:"auto_process,omitempty"` // For warm transfers
}

type UpdateTransferStatusRequest struct {
	Status models.TransferStatus `json:"status" binding:"required"`
}

func (s *Server) createTransfer(c *gin.Context) {
	// Get wallet ID from path
	walletIDParam := c.Param("id")
	walletID, err := uuid.Parse(walletIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid wallet ID"})
		return
	}

	var req CreateTransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify wallet exists and get its type
	wallet, err := s.walletRepo.GetByID(walletID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wallet"})
		return
	}

	if wallet == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Wallet not found"})
		return
	}

	// Get current user ID
	userID := s.getCurrentUserID(c)
	ctx := context.Background()

	// Delegate to appropriate service based on wallet type
	switch wallet.WalletType {
	case models.WalletTypeCold:
		// Create cold transfer request
		coldReq := services.ColdTransferRequest{
			WalletID:         walletID,
			RecipientAddress: req.RecipientAddress,
			AmountString:     req.AmountString,
			Coin:             req.Coin,
			BusinessPurpose:  req.BusinessPurpose,
			RequestorName:    req.RequestorName,
			RequestorEmail:   req.RequestorEmail,
			UrgencyLevel:     req.UrgencyLevel,
		}
		if req.Memo != nil {
			coldReq.Memo = *req.Memo
		}

		transfer, err := s.coldWalletSvc.CreateColdTransferRequest(ctx, coldReq, userID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"transfer": transfer,
			"message":  "Cold transfer request created successfully",
			"type":     "cold",
		})

	case models.WalletTypeWarm:
		// Create warm transfer request
		warmReq := services.WarmTransferRequest{
			WalletID:         walletID,
			RecipientAddress: req.RecipientAddress,
			AmountString:     req.AmountString,
			Coin:             req.Coin,
			BusinessPurpose:  req.BusinessPurpose,
			RequestorName:    req.RequestorName,
			RequestorEmail:   req.RequestorEmail,
			UrgencyLevel:     req.UrgencyLevel,
			AutoProcess:      req.AutoProcess,
		}
		if req.Memo != nil {
			warmReq.Memo = *req.Memo
		}

		transfer, err := s.warmWalletSvc.CreateWarmTransferRequest(ctx, warmReq, userID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"transfer": transfer,
			"message":  "Warm transfer request created successfully",
			"type":     "warm",
		})

	case models.WalletTypeHot:
		// For hot wallets, use the original immediate processing logic
		s.createHotTransfer(c, walletID, wallet, req, userID)

	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Unsupported wallet type: %s", wallet.WalletType),
		})
	}
}

// createHotTransfer handles immediate processing for hot wallets
func (s *Server) createHotTransfer(c *gin.Context, walletID uuid.UUID, wallet *models.Wallet, req CreateTransferRequest, userID uuid.UUID) {
	// Create transfer request in our database first
	transferRequest := &models.TransferRequest{
		WalletID:          walletID,
		RequestedByUserID: userID,
		RecipientAddress:  req.RecipientAddress,
		AmountString:      req.AmountString,
		Coin:              req.Coin,
		TransferType:      models.WalletTypeHot,
		Status:            models.TransferStatusDraft,
		RequiredApprovals: 0, // Hot transfers require no approvals
		ReceivedApprovals: 0,
		Memo:              req.Memo,
	}

	if err := s.transferRequestRepo.Create(transferRequest); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create transfer request"})
		return
	}

	// Try to build the transfer with BitGo immediately
	ctx := context.Background()
	memoStr := ""
	if req.Memo != nil {
		memoStr = *req.Memo
	}

	buildRequest := bitgo.BuildTransferRequest{
		Recipients: []bitgo.TransferRecipient{
			{
				Address:      req.RecipientAddress,
				AmountString: req.AmountString,
			},
		},
		Memo: memoStr,
	}

	// Build transfer with BitGo
	buildResponse, err := s.bitgoClient.BuildTransfer(
		ctx,
		wallet.BitgoWalletID,
		wallet.Coin,
		buildRequest,
	)

	if err != nil {
		// Update transfer request status to failed
		transferRequest.Status = models.TransferStatusFailed
		s.transferRequestRepo.Update(transferRequest)

		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Failed to build transfer with BitGo",
			"details": err.Error(),
		})
		return
	}

	// Update transfer request with BitGo transaction info
	transferRequest.Status = models.TransferStatusSigned // Hot transfers go directly to signed
	if buildResponse.Transfer != nil {
		transferRequest.BitgoTxid = &buildResponse.Transfer.TxID
	}
	if buildResponse.FeeInfo != nil {
		transferRequest.Fee = &buildResponse.FeeInfo.FeeString
		feeRateStr := fmt.Sprintf("%d", buildResponse.FeeInfo.FeeRate)
		transferRequest.FeeRate = &feeRateStr
	}

	if err := s.transferRequestRepo.Update(transferRequest); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update transfer request"})
		return
	}

	// Return the transfer request with BitGo transaction details
	response := gin.H{
		"transfer": transferRequest,
		"message":  "Hot transfer created and ready for broadcast",
		"type":     "hot",
	}

	c.JSON(http.StatusCreated, response)
}

func (s *Server) listTransfers(c *gin.Context) {
	// Get wallet ID from path
	walletIDParam := c.Param("id")
	walletID, err := uuid.Parse(walletIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid wallet ID"})
		return
	}

	// Get pagination parameters
	limit := 25
	offset := 0

	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 500 {
			limit = parsed
		}
	}

	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	transfers, err := s.transferRequestRepo.List(walletID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list transfers"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"transfers": transfers,
		"count":     len(transfers),
		"limit":     limit,
		"offset":    offset,
	})
}

func (s *Server) getTransfer(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transfer ID"})
		return
	}

	transfer, err := s.transferRequestRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get transfer"})
		return
	}

	if transfer == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transfer not found"})
		return
	}

	c.JSON(http.StatusOK, transfer)
}

func (s *Server) updateTransfer(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transfer ID"})
		return
	}

	// Get existing transfer
	transfer, err := s.transferRequestRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get transfer"})
		return
	}

	if transfer == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transfer not found"})
		return
	}

	// For now, just return the transfer as-is
	// In a real implementation, you'd handle BitGo integration here
	c.JSON(http.StatusOK, transfer)
}

func (s *Server) updateTransferStatus(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transfer ID"})
		return
	}

	var req UpdateTransferStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.transferRequestRepo.UpdateStatus(id, req.Status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update transfer status"})
		return
	}

	// Get updated transfer
	transfer, err := s.transferRequestRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get updated transfer"})
		return
	}

	c.JSON(http.StatusOK, transfer)
}

// submitTransfer submits an approved transfer to BitGo for execution
func (s *Server) submitTransfer(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transfer ID"})
		return
	}

	// Get transfer request
	transfer, err := s.transferRequestRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get transfer"})
		return
	}

	if transfer == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transfer not found"})
		return
	}

	// Check if transfer is in a valid state for submission
	if transfer.Status != models.TransferStatusApproved {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":          "Transfer must be approved before submission",
			"current_status": transfer.Status,
		})
		return
	}

	// Get wallet details
	wallet, err := s.walletRepo.GetByID(transfer.WalletID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wallet"})
		return
	}

	// Build submit request
	submitRequest := bitgo.SubmitTransferRequest{
		TxHex: *transfer.BitgoTxid, // Using TxHex instead of TxId
		// In a real implementation, you would include the signed transaction
		// This would come from the approval process
	}

	// Submit transfer directly
	ctx := context.Background()
	submitResponse, err := s.bitgoClient.SubmitTransfer(
		ctx,
		wallet.BitgoWalletID,
		wallet.Coin,
		submitRequest,
	)

	if err != nil {
		// Update transfer status to failed
		transfer.Status = models.TransferStatusFailed
		now := time.Now()
		transfer.FailedAt = &now
		s.transferRequestRepo.Update(transfer)

		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Failed to submit transfer to BitGo",
			"details": err.Error(),
		})
		return
	}

	// Update transfer request with submission details
	transfer.Status = models.TransferStatusBroadcast
	transfer.BitgoTransferID = &submitResponse.Transfer.ID
	transfer.TransactionHash = &submitResponse.Transfer.TxID
	now := time.Now()
	transfer.SubmittedAt = &now

	if err := s.transferRequestRepo.Update(transfer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update transfer"})
		return
	}

	response := gin.H{
		"transfer_request": transfer,
		"bitgo_response":   submitResponse,
	}

	c.JSON(http.StatusOK, response)
}

// getTransferStatus gets the current status of a transfer from BitGo
func (s *Server) getTransferStatus(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transfer ID"})
		return
	}

	// Get transfer request
	transfer, err := s.transferRequestRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get transfer"})
		return
	}

	if transfer == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transfer not found"})
		return
	}

	// If transfer has been submitted, get status from BitGo
	if transfer.BitgoTransferID != nil {
		wallet, err := s.walletRepo.GetByID(transfer.WalletID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wallet"})
			return
		}

		ctx := context.Background()
		bitgoTransfer, err := s.bitgoClient.GetTransfer(ctx, wallet.BitgoWalletID, wallet.Coin, *transfer.BitgoTransferID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to get transfer status from BitGo",
				"details": err.Error(),
			})
			return
		}

		// Normalize the status from BitGo
		statusMapper := bitgo.NewStatusMapper()
		canonicalStatus := statusMapper.NormalizeTransferStatus(bitgoTransfer.State, bitgoTransfer)

		// Update our local record if status changed
		if transfer.Status != models.TransferStatus(canonicalStatus) {
			transfer.Status = models.TransferStatus(canonicalStatus)

			// Update completion timestamps based on status
			now := time.Now()
			switch canonicalStatus {
			case "confirmed":
				if transfer.CompletedAt == nil {
					transfer.CompletedAt = &now
				}
			case "failed":
				if transfer.FailedAt == nil {
					transfer.FailedAt = &now
				}
			}

			s.transferRequestRepo.Update(transfer)
		}

		response := gin.H{
			"transfer_request": transfer,
			"bitgo_transfer":   bitgoTransfer,
			"canonical_status": canonicalStatus,
		}

		c.JSON(http.StatusOK, response)
		return
	}

	// Return local transfer if not submitted yet
	c.JSON(http.StatusOK, gin.H{
		"transfer_request": transfer,
		"bitgo_transfer":   nil,
		"canonical_status": string(transfer.Status),
	})
}

// createColdTransfer creates a new cold storage transfer request
func (s *Server) createColdTransfer(c *gin.Context) {
	var req services.ColdTransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get current user ID
	userID := s.getCurrentUserID(c)

	// Create cold transfer request
	ctx := context.Background()
	transfer, err := s.coldWalletSvc.CreateColdTransferRequest(ctx, req, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Failed to create cold transfer request",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"transfer_request": transfer,
		"message":          "Cold transfer request created successfully. This request requires manual approval and may take up to 72 hours to process.",
	})
}

// getColdTransfersSLA gets SLA status for cold transfers
func (s *Server) getColdTransfersSLA(c *gin.Context) {
	ctx := context.Background()
	slaStatus, err := s.coldWalletSvc.GetColdTransfersSLAStatus(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get cold transfers SLA status",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, slaStatus)
}

// updateOfflineWorkflowState updates the offline workflow state for a cold transfer
func (s *Server) updateOfflineWorkflowState(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transfer ID"})
		return
	}

	var req struct {
		State services.OfflineWorkflowState `json:"state" binding:"required"`
		Notes string                        `json:"notes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := context.Background()
	if err := s.coldWalletSvc.UpdateOfflineWorkflowState(ctx, id, req.State, req.Notes); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Failed to update offline workflow state",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Offline workflow state updated successfully",
		"state":   req.State,
		"notes":   req.Notes,
	})
}

// getColdTransfersAdminQueue gets cold transfers for admin review
func (s *Server) getColdTransfersAdminQueue(c *gin.Context) {
	// Get pagination parameters
	limit := 50
	offset := 0

	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 500 {
			limit = parsed
		}
	}

	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// Get cold transfers that need attention
	coldStatuses := []models.TransferStatus{
		models.TransferStatusSubmitted,
		models.TransferStatusPendingApproval,
		models.TransferStatusApproved,
	}

	transfers, err := s.transferRequestRepo.GetTransfersByStatuses(coldStatuses, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get cold transfers"})
		return
	}

	// Filter only cold transfers
	coldTransfers := make([]*models.TransferRequest, 0)
	for _, transfer := range transfers {
		if transfer.TransferType == models.WalletTypeCold {
			coldTransfers = append(coldTransfers, transfer)
		}
	}

	// Get SLA status for context
	ctx := context.Background()
	slaStatus, _ := s.coldWalletSvc.GetColdTransfersSLAStatus(ctx)

	response := gin.H{
		"transfers":   coldTransfers,
		"count":       len(coldTransfers),
		"sla_summary": slaStatus,
		"pagination": gin.H{
			"limit":  limit,
			"offset": offset,
		},
	}

	c.JSON(http.StatusOK, response)
}

// verifyAddress verifies if a blockchain address is valid
func (s *Server) verifyAddress(c *gin.Context) {
	var req struct {
		Address string `json:"address" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Use BitGo client to verify address format
	ctx := context.Background()
	isValid, err := s.bitgoClient.ValidateAddress(ctx, req.Address)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"valid": false,
			"error": "Address validation failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":   isValid,
		"address": req.Address,
	})
}

// getApprovers returns list of available approvers for transfers
func (s *Server) getApprovers(c *gin.Context) {
	// In a real implementation, this would come from a user management system
	// For now, return a static list of mock approvers
	approvers := []string{
		"admin@company.com",
		"compliance@company.com",
		"cfo@company.com",
		"security@company.com",
		"operations@company.com",
	}

	c.JSON(http.StatusOK, gin.H{
		"approvers": approvers,
	})
}

// WARM TRANSFER ENDPOINTS

// createWarmTransfer creates a new warm storage transfer request
func (s *Server) createWarmTransfer(c *gin.Context) {
	var req services.WarmTransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user ID from context (this would come from JWT token in real implementation)
	userID := uuid.New() // Mock user ID
	ctx := context.Background()

	transfer, err := s.warmWalletSvc.CreateWarmTransferRequest(ctx, req, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"transfer": transfer,
		"message":  "Warm transfer request created successfully",
	})
}

// getWarmTransfersSLA gets SLA status for warm transfers
func (s *Server) getWarmTransfersSLA(c *gin.Context) {
	ctx := context.Background()
	slaStatus, err := s.warmWalletSvc.GetWarmTransfersSLAStatus(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get warm transfers SLA status"})
		return
	}

	c.JSON(http.StatusOK, slaStatus)
}

// getWarmTransfersAnalytics gets analytics and metrics for warm transfers
func (s *Server) getWarmTransfersAnalytics(c *gin.Context) {
	ctx := context.Background()

	// Get basic SLA status
	slaStatus, err := s.warmWalletSvc.GetWarmTransfersSLAStatus(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get analytics"})
		return
	}

	// Get all warm transfers for additional analytics
	warmStatuses := []models.TransferStatus{
		models.TransferStatusSubmitted,
		models.TransferStatusPendingApproval,
		models.TransferStatusApproved,
		models.TransferStatusSigned,
		models.TransferStatusBroadcast,
		models.TransferStatusCompleted,
	}

	transfers, err := s.transferRequestRepo.GetTransfersByStatuses(warmStatuses, 1000)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get transfers"})
		return
	}

	// Filter warm transfers
	warmTransfers := make([]*models.TransferRequest, 0)
	for _, transfer := range transfers {
		if transfer.TransferType == models.WalletTypeWarm {
			warmTransfers = append(warmTransfers, transfer)
		}
	}

	// Calculate additional metrics
	totalVolume := 0.0
	avgProcessingTime := 0.0
	statusBreakdown := make(map[models.TransferStatus]int)

	for _, transfer := range warmTransfers {
		// Parse amount for volume calculation
		if amount, err := parseAmountFloat(transfer.AmountString); err == nil {
			totalVolume += amount
		}

		// Status breakdown
		statusBreakdown[transfer.Status]++

		// Processing time calculation (simplified)
		if transfer.Status == models.TransferStatusCompleted && !transfer.UpdatedAt.IsZero() {
			processingTime := transfer.UpdatedAt.Sub(transfer.CreatedAt).Hours()
			avgProcessingTime += processingTime
		}
	}

	if len(warmTransfers) > 0 {
		avgProcessingTime = avgProcessingTime / float64(len(warmTransfers))
	}

	analytics := map[string]interface{}{
		"sla_status":           slaStatus,
		"total_volume":         totalVolume,
		"avg_processing_hours": avgProcessingTime,
		"status_breakdown":     statusBreakdown,
		"transfer_count":       len(warmTransfers),
	}

	c.JSON(http.StatusOK, analytics)
}

// processWarmTransfer manually processes a warm transfer (for admin override)
func (s *Server) processWarmTransfer(c *gin.Context) {
	transferID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transfer ID"})
		return
	}

	var req struct {
		Action string `json:"action" binding:"required"` // "approve", "reject", "process"
		Notes  string `json:"notes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get the transfer
	transfer, err := s.transferRequestRepo.GetByID(transferID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transfer not found"})
		return
	}

	if transfer.TransferType != models.WalletTypeWarm {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Transfer is not a warm storage transfer"})
		return
	}

	switch req.Action {
	case "approve":
		transfer.Status = models.TransferStatusApproved
		transfer.ReceivedApprovals = transfer.RequiredApprovals
	case "reject":
		transfer.Status = models.TransferStatusRejected
	case "process":
		// Trigger automated processing
		if transfer.Status == models.TransferStatusApproved {
			// This would trigger the actual BitGo processing
			// For now, we'll just update the status
			transfer.Status = models.TransferStatusSigned
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Transfer must be approved before processing"})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid action. Must be 'approve', 'reject', or 'process'"})
		return
	}

	if err := s.transferRequestRepo.Update(transfer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update transfer"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"transfer": transfer,
		"message":  fmt.Sprintf("Transfer %s successfully", req.Action),
		"notes":    req.Notes,
	})
}

// Helper function to parse amount as float
func parseAmountFloat(amountStr string) (float64, error) {
	var amount float64
	_, err := fmt.Sscanf(amountStr, "%f", &amount)
	return amount, err
}
