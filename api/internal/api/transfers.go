package api

import (
	"context"
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

	// Verify wallet exists
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

	// Create transfer request in our database first
	transferRequest := &models.TransferRequest{
		WalletID:          walletID,
		RequestedByUserID: userID,
		RecipientAddress:  req.RecipientAddress,
		AmountString:      req.AmountString,
		Coin:              req.Coin,
		TransferType:      req.TransferType,
		Status:            models.TransferStatusDraft,
		RequiredApprovals: 1,
		ReceivedApprovals: 0,
		Memo:              req.Memo,
	}

	// Set different approval requirements based on transfer type
	if req.TransferType == models.WalletTypeCold {
		transferRequest.RequiredApprovals = 2 // Cold transfers need more approvals
	}

	if err := s.transferRequestRepo.Create(transferRequest); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create transfer request"})
		return
	}

	// Try to build the transfer with BitGo
	ctx := context.Background()
	buildRequest := bitgo.BuildTransferRequest{
		Recipients: []bitgo.TransferRecipient{
			{
				Address: req.RecipientAddress,
				Amount:  req.AmountString,
			},
		},
		Memo: req.Memo,
	}

	// Get idempotent transfer builder
	idempotentBuilder := bitgo.NewIdempotentTransferBuilder(
		s.bitgoClient,
		s.bitgoClient.GetIdempotencyService(),
	)

	// Build transfer with idempotency
	buildResponse, err := idempotentBuilder.BuildTransferIdempotent(
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
	transferRequest.Status = models.TransferStatusPendingApproval
	transferRequest.BitgoTxid = &buildResponse.TxId
	transferRequest.Fee = buildResponse.Fee
	transferRequest.FeeRate = buildResponse.FeeRate

	if err := s.transferRequestRepo.Update(transferRequest); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update transfer request"})
		return
	}

	// Return the transfer request with BitGo transaction details
	response := gin.H{
		"transfer_request": transferRequest,
		"bitgo_tx_preview": buildResponse,
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
		TxId: *transfer.BitgoTxid,
		// In a real implementation, you would include the signed transaction
		// This would come from the approval process
	}

	// Get idempotent transfer builder
	idempotentBuilder := bitgo.NewIdempotentTransferBuilder(
		s.bitgoClient,
		s.bitgoClient.GetIdempotencyService(),
	)

	// Submit transfer with idempotency
	ctx := context.Background()
	submitResponse, err := idempotentBuilder.SubmitTransferIdempotent(
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
	transfer.TransactionHash = &submitResponse.Transfer.TxHash
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
		if string(transfer.Status) != canonicalStatus {
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
