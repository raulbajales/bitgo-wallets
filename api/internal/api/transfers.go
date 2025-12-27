package api

import (
	"net/http"
	"strconv"

	"bitgo-wallets-api/internal/models"

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

	c.JSON(http.StatusCreated, transferRequest)
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
