package bitgo

import (
	"fmt"
	"strings"
	"time"
)

// CanonicalTransferStatus represents our normalized transfer status
type CanonicalTransferStatus string

const (
	// Core statuses
	CanonicalStatusPending   CanonicalTransferStatus = "pending"
	CanonicalStatusConfirmed CanonicalTransferStatus = "confirmed"
	CanonicalStatusFailed    CanonicalTransferStatus = "failed"
	CanonicalStatusRejected  CanonicalTransferStatus = "rejected"
	CanonicalStatusCanceled  CanonicalTransferStatus = "canceled"

	// Intermediate statuses
	CanonicalStatusBuilding        CanonicalTransferStatus = "building"
	CanonicalStatusSigning         CanonicalTransferStatus = "signing"
	CanonicalStatusSubmitting      CanonicalTransferStatus = "submitting"
	CanonicalStatusBroadcast       CanonicalTransferStatus = "broadcast"
	CanonicalStatusWaitingApproval CanonicalTransferStatus = "waiting_approval"

	// Unknown status for unmappable states
	CanonicalStatusUnknown CanonicalTransferStatus = "unknown"
)

// CanonicalWalletType represents our normalized wallet type
type CanonicalWalletType string

const (
	CanonicalWalletTypeWarm     CanonicalWalletType = "warm"     // Custodial/hot wallet
	CanonicalWalletTypeCold     CanonicalWalletType = "cold"     // Cold storage
	CanonicalWalletTypeMultisig CanonicalWalletType = "multisig" // Multi-signature wallet
	CanonicalWalletTypeUnknown  CanonicalWalletType = "unknown"
)

// StatusMapper handles the mapping between BitGo statuses and our canonical statuses
type StatusMapper struct{}

// NewStatusMapper creates a new status mapper
func NewStatusMapper() *StatusMapper {
	return &StatusMapper{}
}

// NormalizeTransferStatus converts BitGo transfer status to canonical status
func (sm *StatusMapper) NormalizeTransferStatus(bitgoStatus TransferStatus, transfer *Transfer) CanonicalTransferStatus {
	switch bitgoStatus {
	case TransferStatusConfirmed:
		return CanonicalStatusConfirmed

	case TransferStatusPending:
		// Check if it's waiting for approval
		if transfer != nil && len(transfer.History) > 0 {
			for _, hist := range transfer.History {
				if strings.Contains(strings.ToLower(hist.Action), "approval") &&
					strings.Contains(strings.ToLower(hist.Action), "pending") {
					return CanonicalStatusWaitingApproval
				}
			}
		}

		// Check confirmations to determine if it's broadcast but unconfirmed
		if transfer != nil && transfer.Confirmations == 0 && transfer.TxID != "" {
			return CanonicalStatusBroadcast
		}

		return CanonicalStatusPending

	case TransferStatusSigning:
		return CanonicalStatusSigning

	case TransferStatusSubmitted:
		return CanonicalStatusSubmitting

	case TransferStatusFailed:
		return CanonicalStatusFailed

	case TransferStatusRejected:
		return CanonicalStatusRejected

	case TransferStatusCanceled:
		return CanonicalStatusCanceled

	default:
		return CanonicalStatusUnknown
	}
}

// NormalizeWalletType converts BitGo wallet information to canonical wallet type
func (sm *StatusMapper) NormalizeWalletType(wallet *Wallet) CanonicalWalletType {
	if wallet == nil {
		return CanonicalWalletTypeUnknown
	}

	// Check for explicit wallet type
	switch wallet.Type {
	case WalletTypeCold:
		return CanonicalWalletTypeCold
	case WalletTypeHot, WalletTypeWarm:
		return CanonicalWalletTypeWarm
	}

	// Infer from wallet properties
	if wallet.Multisig {
		// Multisig wallets with high threshold are typically cold
		if wallet.Threshold >= 2 {
			return CanonicalWalletTypeCold
		}
		return CanonicalWalletTypeMultisig
	}

	// Check for custodial indicators
	if len(wallet.ClientFlags) > 0 {
		for _, flag := range wallet.ClientFlags {
			if strings.Contains(strings.ToLower(flag), "custodial") {
				return CanonicalWalletTypeWarm
			}
			if strings.Contains(strings.ToLower(flag), "cold") {
				return CanonicalWalletTypeCold
			}
		}
	}

	// Default to warm for hot wallets
	return CanonicalWalletTypeWarm
}

// TransferRisk represents the risk level of a transfer
type TransferRisk string

const (
	TransferRiskLow    TransferRisk = "low"
	TransferRiskMedium TransferRisk = "medium"
	TransferRiskHigh   TransferRisk = "high"
)

// AssessTransferRisk evaluates the risk level of a transfer
func (sm *StatusMapper) AssessTransferRisk(req *BuildTransferRequest, walletType CanonicalWalletType) TransferRisk {
	if req == nil {
		return TransferRiskMedium
	}

	// Calculate total value
	totalValue := int64(0)
	for _, recipient := range req.Recipients {
		if recipient.Amount > 0 {
			totalValue += recipient.Amount
		}
	}

	// Risk thresholds (in base units, adjust based on coin)
	highValueThreshold := int64(100000000000)  // 1000 units in satoshis/wei
	mediumValueThreshold := int64(10000000000) // 100 units

	// Cold wallet transfers are inherently higher risk
	if walletType == CanonicalWalletTypeCold {
		if totalValue > mediumValueThreshold {
			return TransferRiskHigh
		}
		return TransferRiskMedium
	}

	// High value transfers
	if totalValue > highValueThreshold {
		return TransferRiskHigh
	}

	// Medium value transfers
	if totalValue > mediumValueThreshold {
		return TransferRiskMedium
	}

	// Multiple recipients increase risk
	if len(req.Recipients) > 3 {
		return TransferRiskMedium
	}

	return TransferRiskLow
}

// TransferSLA represents expected SLA for different transfer types
type TransferSLA struct {
	WalletType          CanonicalWalletType `json:"walletType"`
	ExpectedConfirmTime time.Duration       `json:"expectedConfirmTime"`
	MaxWaitTime         time.Duration       `json:"maxWaitTime"`
	RequiresApproval    bool                `json:"requiresApproval"`
	ApprovalSLA         time.Duration       `json:"approvalSLA,omitempty"`
}

// GetTransferSLA returns expected SLA for a transfer type
func (sm *StatusMapper) GetTransferSLA(walletType CanonicalWalletType, risk TransferRisk) TransferSLA {
	switch walletType {
	case CanonicalWalletTypeWarm:
		sla := TransferSLA{
			WalletType:          walletType,
			ExpectedConfirmTime: 15 * time.Minute, // Typical block time + safety margin
			MaxWaitTime:         2 * time.Hour,
			RequiresApproval:    false,
		}

		// High risk warm transfers may require approval
		if risk == TransferRiskHigh {
			sla.RequiresApproval = true
			sla.ApprovalSLA = 4 * time.Hour
			sla.ExpectedConfirmTime = 6 * time.Hour // Including approval time
		}

		return sla

	case CanonicalWalletTypeCold:
		return TransferSLA{
			WalletType:          walletType,
			ExpectedConfirmTime: 24 * time.Hour, // Manual process
			MaxWaitTime:         72 * time.Hour,
			RequiresApproval:    true,
			ApprovalSLA:         48 * time.Hour,
		}

	case CanonicalWalletTypeMultisig:
		return TransferSLA{
			WalletType:          walletType,
			ExpectedConfirmTime: 2 * time.Hour, // Time for signers to respond
			MaxWaitTime:         24 * time.Hour,
			RequiresApproval:    true,
			ApprovalSLA:         12 * time.Hour,
		}

	default:
		return TransferSLA{
			WalletType:          CanonicalWalletTypeUnknown,
			ExpectedConfirmTime: time.Hour,
			MaxWaitTime:         24 * time.Hour,
			RequiresApproval:    false,
		}
	}
}

// IsTransferStale determines if a transfer has exceeded expected timelines
func (sm *StatusMapper) IsTransferStale(transfer *Transfer, walletType CanonicalWalletType) bool {
	if transfer == nil {
		return false
	}

	canonicalStatus := sm.NormalizeTransferStatus(transfer.State, transfer)

	// Only check stale status for pending/in-progress transfers
	if canonicalStatus == CanonicalStatusConfirmed ||
		canonicalStatus == CanonicalStatusFailed ||
		canonicalStatus == CanonicalStatusRejected ||
		canonicalStatus == CanonicalStatusCanceled {
		return false
	}

	risk := TransferRiskMedium // Default assumption
	sla := sm.GetTransferSLA(walletType, risk)

	elapsed := time.Since(transfer.CreatedTime)
	return elapsed > sla.MaxWaitTime
}

// GetTransferStatusDescription returns a human-readable description of the transfer status
func (sm *StatusMapper) GetTransferStatusDescription(status CanonicalTransferStatus, transfer *Transfer) string {
	switch status {
	case CanonicalStatusPending:
		return "Transfer is being processed"
	case CanonicalStatusConfirmed:
		if transfer != nil && transfer.Confirmations > 0 {
			return fmt.Sprintf("Transfer confirmed with %d confirmations", transfer.Confirmations)
		}
		return "Transfer has been confirmed on the blockchain"
	case CanonicalStatusFailed:
		return "Transfer failed to complete"
	case CanonicalStatusRejected:
		return "Transfer was rejected"
	case CanonicalStatusCanceled:
		return "Transfer was canceled"
	case CanonicalStatusBuilding:
		return "Transfer is being prepared"
	case CanonicalStatusSigning:
		return "Transfer is being signed"
	case CanonicalStatusSubmitting:
		return "Transfer is being submitted to the network"
	case CanonicalStatusBroadcast:
		return "Transfer has been broadcast and is awaiting confirmation"
	case CanonicalStatusWaitingApproval:
		return "Transfer is waiting for approval"
	case CanonicalStatusUnknown:
		return "Transfer status is unknown"
	default:
		return string(status)
	}
}

// NormalizedTransfer represents a transfer with normalized status and metadata
type NormalizedTransfer struct {
	*Transfer
	CanonicalStatus   CanonicalTransferStatus `json:"canonicalStatus"`
	StatusDescription string                  `json:"statusDescription"`
	IsStale           bool                    `json:"isStale"`
	Risk              TransferRisk            `json:"risk"`
	SLA               TransferSLA             `json:"sla"`
	WalletType        CanonicalWalletType     `json:"walletType"`
}

// NormalizeTransfer converts a BitGo transfer to our normalized format
func (sm *StatusMapper) NormalizeTransfer(transfer *Transfer, wallet *Wallet, buildReq *BuildTransferRequest) *NormalizedTransfer {
	if transfer == nil {
		return nil
	}

	walletType := sm.NormalizeWalletType(wallet)
	canonicalStatus := sm.NormalizeTransferStatus(transfer.State, transfer)
	statusDesc := sm.GetTransferStatusDescription(canonicalStatus, transfer)
	isStale := sm.IsTransferStale(transfer, walletType)

	var risk TransferRisk = TransferRiskMedium
	if buildReq != nil {
		risk = sm.AssessTransferRisk(buildReq, walletType)
	}

	sla := sm.GetTransferSLA(walletType, risk)

	return &NormalizedTransfer{
		Transfer:          transfer,
		CanonicalStatus:   canonicalStatus,
		StatusDescription: statusDesc,
		IsStale:           isStale,
		Risk:              risk,
		SLA:               sla,
		WalletType:        walletType,
	}
}
