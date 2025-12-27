package models

import (
	"time"

	"github.com/google/uuid"
)

type TransferRequest struct {
	ID                 uuid.UUID      `json:"id" db:"id"`
	WalletID           uuid.UUID      `json:"wallet_id" db:"wallet_id"`
	RequestedByUserID  uuid.UUID      `json:"requested_by_user_id" db:"requested_by_user_id"`
	RecipientAddress   string         `json:"recipient_address" db:"recipient_address"`
	AmountString       string         `json:"amount_string" db:"amount_string"`
	Coin               string         `json:"coin" db:"coin"`
	TransferType       WalletType     `json:"transfer_type" db:"transfer_type"`
	Status             TransferStatus `json:"status" db:"status"`
	BitgoTransferID    *string        `json:"bitgo_transfer_id" db:"bitgo_transfer_id"`
	TransactionHash    *string        `json:"transaction_hash" db:"transaction_hash"`
	RequiredApprovals  int            `json:"required_approvals" db:"required_approvals"`
	ReceivedApprovals  int            `json:"received_approvals" db:"received_approvals"`
	Memo               *string        `json:"memo" db:"memo"`
	FeeString          *string        `json:"fee_string" db:"fee_string"`
	EstimatedFeeString *string        `json:"estimated_fee_string" db:"estimated_fee_string"`
	SubmittedAt        *time.Time     `json:"submitted_at" db:"submitted_at"`
	ApprovedAt         *time.Time     `json:"approved_at" db:"approved_at"`
	CompletedAt        *time.Time     `json:"completed_at" db:"completed_at"`
	FailedAt           *time.Time     `json:"failed_at" db:"failed_at"`
	CreatedAt          time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at" db:"updated_at"`
}

type TransferStatus string

const (
	TransferStatusDraft           TransferStatus = "draft"
	TransferStatusSubmitted       TransferStatus = "submitted"
	TransferStatusPendingApproval TransferStatus = "pending_approval"
	TransferStatusApproved        TransferStatus = "approved"
	TransferStatusSigned          TransferStatus = "signed"
	TransferStatusBroadcast       TransferStatus = "broadcast"
	TransferStatusConfirmed       TransferStatus = "confirmed"
	TransferStatusCompleted       TransferStatus = "completed"
	TransferStatusFailed          TransferStatus = "failed"
	TransferStatusRejected        TransferStatus = "rejected"
	TransferStatusCancelled       TransferStatus = "cancelled"
)
