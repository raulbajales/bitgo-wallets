package bitgo

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// ApprovalInfo represents BitGo pending approval information
type ApprovalInfo struct {
	ID                string          `json:"id"`
	Type              ApprovalType    `json:"type"`
	State             ApprovalState   `json:"state"`
	Creator           string          `json:"creator"`
	WalletID          string          `json:"walletId"`
	Enterprise        string          `json:"enterprise"`
	Info              ApprovalDetails `json:"info"`
	ApprovalsRequired int             `json:"approvalsRequired"`
	Approvals         []Approval      `json:"approvals"`
	Created           time.Time       `json:"created"`
	Expires           time.Time       `json:"expires"`
}

// ApprovalType represents the type of approval needed
type ApprovalType string

const (
	ApprovalTypeTransactionRequest ApprovalType = "transactionRequest"
	ApprovalTypePolicyChange       ApprovalType = "policyChange"
	ApprovalTypeUserRequest        ApprovalType = "userRequest"
)

// ApprovalState represents the current state of an approval
type ApprovalState string

const (
	ApprovalStatePending   ApprovalState = "pending"
	ApprovalStateApproved  ApprovalState = "approved"
	ApprovalStateRejected  ApprovalState = "rejected"
	ApprovalStateProcessed ApprovalState = "processed"
)

// ApprovalDetails contains type-specific approval information
type ApprovalDetails struct {
	TransactionRequest *TransactionRequestInfo `json:"transactionRequest,omitempty"`
	PolicyChange       *PolicyChangeInfo       `json:"policyChange,omitempty"`
}

// TransactionRequestInfo contains transaction-specific approval details
type TransactionRequestInfo struct {
	RequestID        string      `json:"requestId"`
	TxRequestID      string      `json:"txRequestId"`
	Coin             string      `json:"coin"`
	ValueString      string      `json:"valueString"`
	Recipients       []Recipient `json:"recipients"`
	Fee              int64       `json:"fee"`
	FeeString        string      `json:"feeString"`
	Message          string      `json:"message"`
	InitiatingUserID string      `json:"initiatingUserId"`
}

// PolicyChangeInfo contains policy-specific approval details
type PolicyChangeInfo struct {
	PolicyID   string `json:"policyId"`
	RuleName   string `json:"ruleName"`
	OldValue   string `json:"oldValue"`
	NewValue   string `json:"newValue"`
	ChangeType string `json:"changeType"`
}

// Recipient represents a transaction recipient
type Recipient struct {
	Address string `json:"address"`
	Amount  string `json:"amount"`
}

// Approval represents a single approval from a user
type Approval struct {
	ID       string    `json:"id"`
	UserID   string    `json:"userId"`
	Username string    `json:"username"`
	State    string    `json:"state"`
	Date     time.Time `json:"date"`
}

// ListApprovalsParams represents parameters for listing pending approvals
type ListApprovalsParams struct {
	Coin       string        `json:"coin,omitempty"`
	Type       ApprovalType  `json:"type,omitempty"`
	State      ApprovalState `json:"state,omitempty"`
	Enterprise string        `json:"enterprise,omitempty"`
	Limit      int           `json:"limit,omitempty"`
	Skip       int           `json:"skip,omitempty"`
}

// ListApprovalsResponse represents the response from listing approvals
type ListApprovalsResponse struct {
	Approvals []ApprovalInfo `json:"pendingApprovals"`
	Count     int            `json:"count"`
}

// ApprovalService handles BitGo approval operations
type ApprovalService struct {
	client *Client
	logger Logger
}

// NewApprovalService creates a new approval service
func NewApprovalService(client *Client, logger Logger) *ApprovalService {
	return &ApprovalService{
		client: client,
		logger: logger,
	}
}

// ListPendingApprovals gets all pending approvals for the enterprise
func (as *ApprovalService) ListPendingApprovals(ctx context.Context, params ListApprovalsParams) (*ListApprovalsResponse, error) {
	path := "/pendingapprovals"

	// Build query parameters
	queryParams := make(map[string]string)
	if params.Coin != "" {
		queryParams["coin"] = params.Coin
	}
	if params.Type != "" {
		queryParams["type"] = string(params.Type)
	}
	if params.State != "" {
		queryParams["state"] = string(params.State)
	}
	if params.Enterprise != "" {
		queryParams["enterprise"] = params.Enterprise
	}
	if params.Limit > 0 {
		queryParams["limit"] = fmt.Sprintf("%d", params.Limit)
	}
	if params.Skip > 0 {
		queryParams["skip"] = fmt.Sprintf("%d", params.Skip)
	}

	// Add query parameters to path
	if len(queryParams) > 0 {
		path += "?"
		first := true
		for key, value := range queryParams {
			if !first {
				path += "&"
			}
			path += fmt.Sprintf("%s=%s", key, value)
			first = false
		}
	}

	resp, err := as.client.makeRequest(ctx, RequestOptions{
		Method: "GET",
		Path:   path,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pending approvals: %w", err)
	}
	defer resp.Body.Close()

	var response ListApprovalsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode list approvals response: %w", err)
	}

	as.logger.Info("Listed pending approvals",
		"count", response.Count,
		"coin", params.Coin,
		"type", params.Type,
	)

	return &response, nil
}

// GetApproval gets a specific approval by ID
func (as *ApprovalService) GetApproval(ctx context.Context, approvalID string) (*ApprovalInfo, error) {
	path := fmt.Sprintf("/pendingapprovals/%s", approvalID)

	resp, err := as.client.makeRequest(ctx, RequestOptions{
		Method: "GET",
		Path:   path,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get approval %s: %w", approvalID, err)
	}
	defer resp.Body.Close()

	var approval ApprovalInfo
	if err := json.NewDecoder(resp.Body).Decode(&approval); err != nil {
		return nil, fmt.Errorf("failed to decode approval response: %w", err)
	}

	as.logger.Info("Retrieved approval info",
		"approval_id", approvalID,
		"type", approval.Type,
		"state", approval.State,
		"wallet_id", approval.WalletID,
	)

	return &approval, nil
}

// GetWalletApprovals gets pending approvals for a specific wallet
func (as *ApprovalService) GetWalletApprovals(ctx context.Context, walletID, coin string) ([]ApprovalInfo, error) {
	params := ListApprovalsParams{
		Coin:  coin,
		Type:  ApprovalTypeTransactionRequest,
		State: ApprovalStatePending,
		Limit: 100,
	}

	response, err := as.ListPendingApprovals(ctx, params)
	if err != nil {
		return nil, err
	}

	// Filter approvals for specific wallet
	var walletApprovals []ApprovalInfo
	for _, approval := range response.Approvals {
		if approval.WalletID == walletID {
			walletApprovals = append(walletApprovals, approval)
		}
	}

	as.logger.Info("Retrieved wallet approvals",
		"wallet_id", walletID,
		"coin", coin,
		"count", len(walletApprovals),
	)

	return walletApprovals, nil
}

// ApprovalStatus represents the UI-friendly approval status
type ApprovalStatus struct {
	ID                string              `json:"id"`
	Type              ApprovalType        `json:"type"`
	State             ApprovalState       `json:"state"`
	WalletID          string              `json:"walletId"`
	RequiredApprovals int                 `json:"requiredApprovals"`
	ReceivedApprovals int                 `json:"receivedApprovals"`
	PendingApprovals  int                 `json:"pendingApprovals"`
	Approvers         []ApproverInfo      `json:"approvers"`
	TransactionInfo   *TransactionSummary `json:"transactionInfo,omitempty"`
	TimeRemaining     time.Duration       `json:"timeRemaining"`
	IsExpired         bool                `json:"isExpired"`
	CanUserApprove    bool                `json:"canUserApprove"`
	Created           time.Time           `json:"created"`
	Expires           time.Time           `json:"expires"`
}

// ApproverInfo represents information about an approver
type ApproverInfo struct {
	UserID       string     `json:"userId"`
	Username     string     `json:"username"`
	State        string     `json:"state"`
	ApprovalDate *time.Time `json:"approvalDate,omitempty"`
}

// TransactionSummary provides a simplified view of transaction details
type TransactionSummary struct {
	Coin       string   `json:"coin"`
	Amount     string   `json:"amount"`
	Recipients []string `json:"recipients"`
	Fee        string   `json:"fee"`
	Message    string   `json:"message,omitempty"`
}

// MapApprovalToUIStatus converts BitGo approval info to UI-friendly status
func (as *ApprovalService) MapApprovalToUIStatus(approval *ApprovalInfo, currentUserID string) *ApprovalStatus {
	// Calculate approvals
	receivedApprovals := 0
	var approvers []ApproverInfo

	for _, app := range approval.Approvals {
		approver := ApproverInfo{
			UserID:   app.UserID,
			Username: app.Username,
			State:    app.State,
		}

		if app.State == "approved" {
			receivedApprovals++
			approver.ApprovalDate = &app.Date
		}

		approvers = append(approvers, approver)
	}

	pendingApprovals := approval.ApprovalsRequired - receivedApprovals
	timeRemaining := time.Until(approval.Expires)
	isExpired := timeRemaining <= 0

	// Check if current user can approve (not already approved and not the creator)
	canUserApprove := false
	if currentUserID != "" && currentUserID != approval.Creator && !isExpired {
		userAlreadyApproved := false
		for _, app := range approval.Approvals {
			if app.UserID == currentUserID && app.State == "approved" {
				userAlreadyApproved = true
				break
			}
		}
		canUserApprove = !userAlreadyApproved && pendingApprovals > 0
	}

	status := &ApprovalStatus{
		ID:                approval.ID,
		Type:              approval.Type,
		State:             approval.State,
		WalletID:          approval.WalletID,
		RequiredApprovals: approval.ApprovalsRequired,
		ReceivedApprovals: receivedApprovals,
		PendingApprovals:  pendingApprovals,
		Approvers:         approvers,
		TimeRemaining:     timeRemaining,
		IsExpired:         isExpired,
		CanUserApprove:    canUserApprove,
		Created:           approval.Created,
		Expires:           approval.Expires,
	}

	// Add transaction info if available
	if approval.Info.TransactionRequest != nil {
		txInfo := approval.Info.TransactionRequest
		recipients := make([]string, len(txInfo.Recipients))
		for i, recipient := range txInfo.Recipients {
			recipients[i] = recipient.Address
		}

		status.TransactionInfo = &TransactionSummary{
			Coin:       txInfo.Coin,
			Amount:     txInfo.ValueString,
			Recipients: recipients,
			Fee:        txInfo.FeeString,
			Message:    txInfo.Message,
		}
	}

	return status
}

// GetTransferApprovalStatus gets approval status for a specific transfer
func (as *ApprovalService) GetTransferApprovalStatus(ctx context.Context, walletID, coin, transferID string, currentUserID string) (*ApprovalStatus, error) {
	// Get all pending approvals for the wallet
	approvals, err := as.GetWalletApprovals(ctx, walletID, coin)
	if err != nil {
		return nil, err
	}

	// Find approval matching the transfer
	for _, approval := range approvals {
		if approval.Info.TransactionRequest != nil &&
			approval.Info.TransactionRequest.TxRequestID == transferID {
			return as.MapApprovalToUIStatus(&approval, currentUserID), nil
		}
	}

	// No pending approval found for this transfer
	return nil, nil
}
