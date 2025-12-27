package api

import (
	"net/http"
	"strconv"

	"bitgo-wallets-api/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CreateWalletRequest struct {
	BitgoWalletID string            `json:"bitgo_wallet_id" binding:"required"`
	Label         string            `json:"label" binding:"required"`
	Coin          string            `json:"coin" binding:"required"`
	WalletType    models.WalletType `json:"wallet_type" binding:"required"`
	MultisigType  *string           `json:"multisig_type"`
	Threshold     *int              `json:"threshold"`
	Tags          []string          `json:"tags"`
	Metadata      models.JSON       `json:"metadata"`
}

type UpdateWalletRequest struct {
	Label                  string      `json:"label"`
	BalanceString          string      `json:"balance_string"`
	ConfirmedBalanceString string      `json:"confirmed_balance_string"`
	SpendableBalanceString string      `json:"spendable_balance_string"`
	Frozen                 bool        `json:"frozen"`
	Tags                   []string    `json:"tags"`
	Metadata               models.JSON `json:"metadata"`
}

func (s *Server) createWallet(c *gin.Context) {
	var req CreateWalletRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get default organization (for now, using a hardcoded ID)
	// In a real implementation, you'd get this from the user context
	orgID := uuid.New() // This should come from the database

	wallet := &models.Wallet{
		OrganizationID:         orgID,
		BitgoWalletID:          req.BitgoWalletID,
		Label:                  req.Label,
		Coin:                   req.Coin,
		WalletType:             req.WalletType,
		BalanceString:          "0",
		ConfirmedBalanceString: "0",
		SpendableBalanceString: "0",
		IsActive:               true,
		Frozen:                 false,
		MultisigType:           req.MultisigType,
		Threshold:              2, // default
		Tags:                   req.Tags,
		Metadata:               req.Metadata,
	}

	if req.Threshold != nil {
		wallet.Threshold = *req.Threshold
	}

	if err := s.walletRepo.Create(wallet); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create wallet"})
		return
	}

	c.JSON(http.StatusCreated, wallet)
}

func (s *Server) listWallets(c *gin.Context) {
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

	// For demo, use a hardcoded organization ID
	orgID := uuid.New() // This should come from user context

	wallets, err := s.walletRepo.List(orgID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list wallets"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"wallets": wallets,
		"count":   len(wallets),
		"limit":   limit,
		"offset":  offset,
	})
}

func (s *Server) getWallet(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid wallet ID"})
		return
	}

	wallet, err := s.walletRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wallet"})
		return
	}

	if wallet == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Wallet not found"})
		return
	}

	c.JSON(http.StatusOK, wallet)
}

func (s *Server) updateWallet(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid wallet ID"})
		return
	}

	var req UpdateWalletRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get existing wallet
	wallet, err := s.walletRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wallet"})
		return
	}

	if wallet == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Wallet not found"})
		return
	}

	// Update fields
	if req.Label != "" {
		wallet.Label = req.Label
	}
	if req.BalanceString != "" {
		wallet.BalanceString = req.BalanceString
	}
	if req.ConfirmedBalanceString != "" {
		wallet.ConfirmedBalanceString = req.ConfirmedBalanceString
	}
	if req.SpendableBalanceString != "" {
		wallet.SpendableBalanceString = req.SpendableBalanceString
	}
	wallet.Frozen = req.Frozen
	if req.Tags != nil {
		wallet.Tags = req.Tags
	}
	if req.Metadata != nil {
		wallet.Metadata = req.Metadata
	}

	if err := s.walletRepo.Update(wallet); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update wallet"})
		return
	}

	c.JSON(http.StatusOK, wallet)
}

func (s *Server) deleteWallet(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid wallet ID"})
		return
	}

	if err := s.walletRepo.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete wallet"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Wallet deleted successfully"})
}
