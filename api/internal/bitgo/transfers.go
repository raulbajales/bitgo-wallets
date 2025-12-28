package bitgo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// TransferStatus represents the status of a transfer
type TransferStatus string

const (
	TransferStatusPending   TransferStatus = "pending"
	TransferStatusConfirmed TransferStatus = "confirmed"
	TransferStatusRejected  TransferStatus = "rejected"
	TransferStatusCanceled  TransferStatus = "canceled"
	TransferStatusFailed    TransferStatus = "failed"
	TransferStatusSigning   TransferStatus = "signing"
	TransferStatusSubmitted TransferStatus = "submitted"
)

// TransferType represents the type of transfer
type TransferType string

const (
	TransferTypeSend     TransferType = "send"
	TransferTypeReceive  TransferType = "receive"
	TransferTypeInternal TransferType = "internal"
)

// Transfer represents a BitGo transfer/transaction
type Transfer struct {
	ID              string            `json:"id"`
	Coin            string            `json:"coin"`
	Wallet          string            `json:"wallet"`
	Enterprise      string            `json:"enterprise,omitempty"`
	TxID            string            `json:"txid,omitempty"`
	Height          int64             `json:"height,omitempty"`
	Date            time.Time         `json:"date"`
	Type            TransferType      `json:"type"`
	Value           int64             `json:"value"`
	ValueString     string            `json:"valueString"`
	BaseValue       int64             `json:"baseValue,omitempty"`
	BaseValueString string            `json:"baseValueString,omitempty"`
	FeeString       string            `json:"feeString,omitempty"`
	PayGoFeeString  string            `json:"payGoFeeString,omitempty"`
	USDValue        float64           `json:"usdValue,omitempty"`
	State           TransferStatus    `json:"state"`
	Tags            []string          `json:"tags,omitempty"`
	History         []TransferHistory `json:"history,omitempty"`
	Comment         string            `json:"comment,omitempty"`
	VOut            int               `json:"vout,omitempty"`
	Entries         []TransferEntry   `json:"entries,omitempty"`
	Confirmations   int               `json:"confirmations,omitempty"`
	ConfirmedTime   *time.Time        `json:"confirmedTime,omitempty"`
	UnconfirmedTime *time.Time        `json:"unconfirmedTime,omitempty"`
	CreatedTime     time.Time         `json:"createdTime"`
	ModifiedTime    time.Time         `json:"modifiedTime"`
}

// TransferHistory represents the history of state changes for a transfer
type TransferHistory struct {
	Date    time.Time `json:"date"`
	User    string    `json:"user,omitempty"`
	Action  string    `json:"action"`
	Comment string    `json:"comment,omitempty"`
}

// TransferEntry represents an input/output entry in a transfer
type TransferEntry struct {
	Address     string `json:"address"`
	Value       int64  `json:"value"`
	ValueString string `json:"valueString"`
	IsChange    bool   `json:"isChange,omitempty"`
	IsPayGo     bool   `json:"isPayGo,omitempty"`
}

// BuildTransferRequest represents a request to build a transfer
type BuildTransferRequest struct {
	Type                        string               `json:"type,omitempty"`
	Recipients                  []TransferRecipient  `json:"recipients"`
	FeeRate                     int64                `json:"feeRate,omitempty"`
	FeeMultiplier               float64              `json:"feeMultiplier,omitempty"`
	MaxFeeRate                  int64                `json:"maxFeeRate,omitempty"`
	MinConfirms                 int                  `json:"minConfirms,omitempty"`
	EnforceMinConfirmsForChange bool                 `json:"enforceMinConfirmsForChange,omitempty"`
	SequenceId                  string               `json:"sequenceId,omitempty"`
	Comment                     string               `json:"comment,omitempty"`
	Otp                         string               `json:"otp,omitempty"`
	Memo                        string               `json:"memo,omitempty"`
	CpfpTxIds                   []string             `json:"cpfpTxIds,omitempty"`
	CpfpFeeRate                 int64                `json:"cpfpFeeRate,omitempty"`
	MaxValue                    int64                `json:"maxValue,omitempty"`
	Prebuild                    *PrebuildTransaction `json:"prebuild,omitempty"`
	Preview                     bool                 `json:"preview,omitempty"`
}

// TransferRecipient represents a recipient in a transfer
type TransferRecipient struct {
	Address      string `json:"address"`
	Amount       int64  `json:"amount,omitempty"`
	AmountString string `json:"amountString,omitempty"`
	Data         string `json:"data,omitempty"`
}

// PrebuildTransaction represents a prebuilt transaction
type PrebuildTransaction struct {
	TxHex       string                 `json:"txHex"`
	TxInfo      map[string]interface{} `json:"txInfo"`
	FeeInfo     FeeInfo                `json:"feeInfo"`
	WalletId    string                 `json:"walletId"`
	BuildParams map[string]interface{} `json:"buildParams"`
}

// FeeInfo represents fee information for a transaction
type FeeInfo struct {
	Fee       int64  `json:"fee"`
	FeeString string `json:"feeString"`
	FeeRate   int64  `json:"feeRate,omitempty"`
	Size      int    `json:"size,omitempty"`
}

// BuildTransferResponse represents the response from building a transfer
type BuildTransferResponse struct {
	Transfer     *Transfer              `json:"transfer,omitempty"`
	PrebuildTx   *PrebuildTransaction   `json:"prebuildTx,omitempty"`
	BuildParams  map[string]interface{} `json:"buildParams,omitempty"`
	FeeInfo      *FeeInfo               `json:"feeInfo,omitempty"`
	CoinSpecific interface{}            `json:"coinSpecific,omitempty"`
}

// SubmitTransferRequest represents a request to submit a transfer
type SubmitTransferRequest struct {
	TxHex      string                 `json:"txHex"`
	HalfSigned map[string]interface{} `json:"halfSigned,omitempty"`
	Comment    string                 `json:"comment,omitempty"`
	Otp        string                 `json:"otp,omitempty"`
}

// SubmitTransferResponse represents the response from submitting a transfer
type SubmitTransferResponse struct {
	Transfer *Transfer `json:"transfer,omitempty"`
	TxID     string    `json:"txid,omitempty"`
	Status   string    `json:"status,omitempty"`
}

// BuildTransfer creates a new transfer (transaction) for the specified wallet
func (c *Client) BuildTransfer(ctx context.Context, walletID, coin string, req BuildTransferRequest) (*BuildTransferResponse, error) {
	if walletID == "" {
		return nil, fmt.Errorf("wallet ID is required")
	}
	if coin == "" {
		return nil, fmt.Errorf("coin is required")
	}
	if len(req.Recipients) == 0 {
		return nil, fmt.Errorf("at least one recipient is required")
	}

	// Generate sequence ID if not provided for idempotency
	if req.SequenceId == "" {
		req.SequenceId = uuid.New().String()
	}

	path := fmt.Sprintf("/%s/wallet/%s/tx/build", coin, walletID)

	c.logger.Info("Building transfer",
		"wallet_id", walletID,
		"coin", coin,
		"sequence_id", req.SequenceId,
		"recipients_count", len(req.Recipients),
	)

	resp, err := c.makeRequest(ctx, RequestOptions{
		Method: http.MethodPost,
		Path:   path,
		Body:   req,
		Headers: map[string]string{
			"Accept": "application/json",
		},
		IdempotencyKey: req.SequenceId,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to build transfer: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var result BuildTransferResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	c.logger.Info("Transfer built successfully",
		"wallet_id", walletID,
		"coin", coin,
		"sequence_id", req.SequenceId,
	)

	return &result, nil
}

// SubmitTransfer submits a signed transfer to the network
func (c *Client) SubmitTransfer(ctx context.Context, walletID, coin string, req SubmitTransferRequest) (*SubmitTransferResponse, error) {
	if walletID == "" {
		return nil, fmt.Errorf("wallet ID is required")
	}
	if coin == "" {
		return nil, fmt.Errorf("coin is required")
	}
	if req.TxHex == "" && req.HalfSigned == nil {
		return nil, fmt.Errorf("either txHex or halfSigned is required")
	}

	path := fmt.Sprintf("/%s/wallet/%s/tx/send", coin, walletID)

	c.logger.Info("Submitting transfer",
		"wallet_id", walletID,
		"coin", coin,
	)

	resp, err := c.makeRequest(ctx, RequestOptions{
		Method: http.MethodPost,
		Path:   path,
		Body:   req,
		Headers: map[string]string{
			"Accept": "application/json",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to submit transfer: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var result SubmitTransferResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	c.logger.Info("Transfer submitted successfully",
		"wallet_id", walletID,
		"coin", coin,
		"txid", result.TxID,
	)

	return &result, nil
}

// GetTransfer retrieves a specific transfer by ID
func (c *Client) GetTransfer(ctx context.Context, walletID, coin, transferID string) (*Transfer, error) {
	if walletID == "" {
		return nil, fmt.Errorf("wallet ID is required")
	}
	if coin == "" {
		return nil, fmt.Errorf("coin is required")
	}
	if transferID == "" {
		return nil, fmt.Errorf("transfer ID is required")
	}

	path := fmt.Sprintf("/%s/wallet/%s/transfer/%s", coin, walletID, transferID)

	resp, err := c.makeRequest(ctx, RequestOptions{
		Method: http.MethodGet,
		Path:   path,
		Headers: map[string]string{
			"Accept": "application/json",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get transfer: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var transfer Transfer
	if err := json.Unmarshal(body, &transfer); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	c.logger.Info("Retrieved transfer successfully",
		"wallet_id", walletID,
		"coin", coin,
		"transfer_id", transferID,
		"status", transfer.State,
	)

	return &transfer, nil
}

// ListTransfers retrieves transfers for a wallet
func (c *Client) ListTransfers(ctx context.Context, walletID, coin string, options *TransferListOptions) (*TransferListResponse, error) {
	if walletID == "" {
		return nil, fmt.Errorf("wallet ID is required")
	}
	if coin == "" {
		return nil, fmt.Errorf("coin is required")
	}

	path := fmt.Sprintf("/%s/wallet/%s/transfer", coin, walletID)

	resp, err := c.makeRequest(ctx, RequestOptions{
		Method: http.MethodGet,
		Path:   path,
		Headers: map[string]string{
			"Accept": "application/json",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list transfers: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var result TransferListResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	c.logger.Info("Listed transfers successfully",
		"wallet_id", walletID,
		"coin", coin,
		"count", len(result.Transfers),
	)

	return &result, nil
}

// TransferListOptions holds options for listing transfers
type TransferListOptions struct {
	Limit       int            `json:"limit,omitempty"`
	Skip        int            `json:"skip,omitempty"`
	State       TransferStatus `json:"state,omitempty"`
	Type        TransferType   `json:"type,omitempty"`
	SearchLabel string         `json:"searchLabel,omitempty"`
	StartDate   *time.Time     `json:"startDate,omitempty"`
	EndDate     *time.Time     `json:"endDate,omitempty"`
}

// TransferListResponse represents the response from listing transfers
type TransferListResponse struct {
	Transfers       []Transfer `json:"transfers"`
	Count           int        `json:"count"`
	Total           int        `json:"total"`
	NextBatchPrevId string     `json:"nextBatchPrevId,omitempty"`
}

// BuildAndSubmitTransfer is a convenience method that builds and submits a transfer in one operation
// This is primarily for custodial (warm) wallets where no additional signing is required
func (c *Client) BuildAndSubmitTransfer(ctx context.Context, walletID, coin string, buildReq BuildTransferRequest) (*SubmitTransferResponse, error) {
	// First, build the transaction
	buildResp, err := c.BuildTransfer(ctx, walletID, coin, buildReq)
	if err != nil {
		return nil, fmt.Errorf("failed to build transfer: %w", err)
	}

	if buildResp.PrebuildTx == nil {
		return nil, fmt.Errorf("no prebuild transaction returned")
	}

	// For custodial wallets, the transaction should be ready to submit
	submitReq := SubmitTransferRequest{
		TxHex:   buildResp.PrebuildTx.TxHex,
		Comment: buildReq.Comment,
		Otp:     buildReq.Otp,
	}

	// Submit the transaction
	submitResp, err := c.SubmitTransfer(ctx, walletID, coin, submitReq)
	if err != nil {
		return nil, fmt.Errorf("failed to submit transfer: %w", err)
	}

	c.logger.Info("Built and submitted transfer successfully",
		"wallet_id", walletID,
		"coin", coin,
		"sequence_id", buildReq.SequenceId,
		"txid", submitResp.TxID,
	)

	return submitResp, nil
}
