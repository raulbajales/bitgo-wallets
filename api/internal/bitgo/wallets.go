package bitgo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// WalletType represents different types of wallets
type WalletType string

const (
	WalletTypeHot  WalletType = "hot"
	WalletTypeCold WalletType = "cold"
	WalletTypeWarm WalletType = "custodial"
)

// Wallet represents a BitGo wallet
type Wallet struct {
	ID                              string            `json:"id"`
	Label                           string            `json:"label"`
	Coin                            string            `json:"coin"`
	Enterprise                      string            `json:"enterprise,omitempty"`
	BitGoID                         string            `json:"bitGoId,omitempty"`
	Balance                         string            `json:"balance"`
	ConfirmedBalance                string            `json:"confirmedBalance"`
	SpendableBalance                string            `json:"spendableBalance"`
	BalanceString                   string            `json:"balanceString"`
	ConfirmedBalanceString          string            `json:"confirmedBalanceString"`
	SpendableBalanceString          string            `json:"spendableBalanceString"`
	ReceiveAddress                  *Address          `json:"receiveAddress,omitempty"`
	PendingApprovals                []PendingApproval `json:"pendingApprovals,omitempty"`
	Multisig                        bool              `json:"multisig"`
	MultisigType                    string            `json:"multisigType,omitempty"`
	Threshold                       int               `json:"threshold,omitempty"`
	Tags                            []string          `json:"tags,omitempty"`
	Frozen                          bool              `json:"frozen"`
	ApprovalsRequired               int               `json:"approvalsRequired,omitempty"`
	DisableTransactionNotifications bool              `json:"disableTransactionNotifications"`
	Type                            WalletType        `json:"type,omitempty"`
	BuildDefaults                   *BuildDefaults    `json:"buildDefaults,omitempty"`
	CustomChangeKeySignatures       map[string]string `json:"customChangeKeySignatures,omitempty"`
	RecoveryXpub                    string            `json:"recoveryXpub,omitempty"`
	AllowBackupKeySigning           bool              `json:"allowBackupKeySigning"`
	CoinSpecific                    interface{}       `json:"coinSpecific,omitempty"`
	ClientFlags                     []string          `json:"clientFlags,omitempty"`
	WalletFlags                     []string          `json:"walletFlags,omitempty"`
}

// Address represents a wallet address
type Address struct {
	Address     string `json:"address"`
	Chain       int    `json:"chain"`
	Index       int    `json:"index"`
	Coin        string `json:"coin"`
	WalletID    string `json:"wallet,omitempty"`
	AddressType string `json:"addressType,omitempty"`
}

// PendingApproval represents a pending approval for a wallet operation
type PendingApproval struct {
	ID                string                 `json:"id"`
	Type              string                 `json:"type"`
	State             string                 `json:"state"`
	Creator           string                 `json:"creator"`
	Wallet            string                 `json:"wallet"`
	Enterprise        string                 `json:"enterprise"`
	Info              map[string]interface{} `json:"info"`
	ApprovalsRequired int                    `json:"approvalsRequired"`
	Scope             string                 `json:"scope"`
	CreateDate        time.Time              `json:"createDate"`
	ResolveDate       *time.Time             `json:"resolveDate,omitempty"`
}

// BuildDefaults represents default build parameters for transactions
type BuildDefaults struct {
	MinConfirms                 int   `json:"minConfirms,omitempty"`
	MinValue                    int64 `json:"minValue,omitempty"`
	MaxValue                    int64 `json:"maxValue,omitempty"`
	FeeRate                     int64 `json:"feeRate,omitempty"`
	MaxFeeRate                  int64 `json:"maxFeeRate,omitempty"`
	DisableKRSEmail             bool  `json:"disableKRSEmail,omitempty"`
	EnforceMinConfirmsForChange bool  `json:"enforceMinConfirmsForChange,omitempty"`
}

// WalletListOptions holds options for listing wallets
type WalletListOptions struct {
	Coin        string `json:"coin,omitempty"`
	Limit       int    `json:"limit,omitempty"`
	Skip        int    `json:"skip,omitempty"`
	Enterprise  string `json:"enterprise,omitempty"`
	IsCustodial *bool  `json:"isCustodial,omitempty"`
}

// WalletListResponse represents the response from listing wallets
type WalletListResponse struct {
	Wallets []Wallet `json:"wallets"`
	Coin    string   `json:"coin"`
	Count   int      `json:"count"`
	Total   int      `json:"total"`
}

// ListWallets retrieves a list of wallets for the enterprise/user
func (c *Client) ListWallets(ctx context.Context, opts WalletListOptions) (*WalletListResponse, error) {
	path := "/wallets"

	// Add enterprise filter if specified
	if opts.Enterprise != "" || c.enterprise != "" {
		enterprise := opts.Enterprise
		if enterprise == "" {
			enterprise = c.enterprise
		}
		path += "/" + enterprise
	}

	resp, err := c.makeRequest(ctx, RequestOptions{
		Method: http.MethodGet,
		Path:   path,
		Headers: map[string]string{
			"Accept": "application/json",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list wallets: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var result WalletListResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	c.logger.Info("Listed wallets successfully",
		"count", len(result.Wallets),
		"coin", opts.Coin,
	)

	return &result, nil
}

// CreateWalletRaw creates a wallet using raw request body
func (c *Client) CreateWalletRaw(ctx context.Context, coin string, body map[string]interface{}) (*Wallet, error) {
	// Direct API endpoint (not BitGo Express): POST /api/v2/{coin}/wallet
	path := fmt.Sprintf("/%s/wallet", coin)

	c.logger.Info("Creating wallet via direct API",
		"coin", coin,
		"path", path,
		"enterprise", c.enterprise,
	)

	resp, err := c.makeRequest(ctx, RequestOptions{
		Method: http.MethodPost,
		Path:   path,
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create wallet: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, c.parseAPIError(resp, "")
	}

	var wallet Wallet
	if err := json.NewDecoder(resp.Body).Decode(&wallet); err != nil {
		return nil, fmt.Errorf("failed to decode wallet response: %w", err)
	}

	c.logger.Info("Wallet created successfully",
		"wallet_id", wallet.ID,
		"label", wallet.Label,
		"coin", coin,
	)

	return &wallet, nil
}

// GetWallet retrieves a specific wallet by ID
func (c *Client) GetWallet(ctx context.Context, walletID, coin string) (*Wallet, error) {
	if walletID == "" {
		return nil, fmt.Errorf("wallet ID is required")
	}
	if coin == "" {
		return nil, fmt.Errorf("coin is required")
	}

	path := fmt.Sprintf("/%s/wallet/%s", coin, walletID)

	resp, err := c.makeRequest(ctx, RequestOptions{
		Method: http.MethodGet,
		Path:   path,
		Headers: map[string]string{
			"Accept": "application/json",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var wallet Wallet
	if err := json.Unmarshal(body, &wallet); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	c.logger.Info("Retrieved wallet successfully",
		"wallet_id", walletID,
		"coin", coin,
		"label", wallet.Label,
	)

	return &wallet, nil
}

// GetWalletBalance retrieves the current balance for a wallet
func (c *Client) GetWalletBalance(ctx context.Context, walletID, coin string) (*WalletBalance, error) {
	wallet, err := c.GetWallet(ctx, walletID, coin)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet for balance: %w", err)
	}

	balance := &WalletBalance{
		WalletID:               walletID,
		Coin:                   coin,
		Balance:                wallet.Balance,
		ConfirmedBalance:       wallet.ConfirmedBalance,
		SpendableBalance:       wallet.SpendableBalance,
		BalanceString:          wallet.BalanceString,
		ConfirmedBalanceString: wallet.ConfirmedBalanceString,
		SpendableBalanceString: wallet.SpendableBalanceString,
	}

	return balance, nil
}

// WalletBalance represents wallet balance information
type WalletBalance struct {
	WalletID               string `json:"walletId"`
	Coin                   string `json:"coin"`
	Balance                string `json:"balance"`
	ConfirmedBalance       string `json:"confirmedBalance"`
	SpendableBalance       string `json:"spendableBalance"`
	BalanceString          string `json:"balanceString"`
	ConfirmedBalanceString string `json:"confirmedBalanceString"`
	SpendableBalanceString string `json:"spendableBalanceString"`
}

// GenerateAddress creates a new receiving address for the wallet
func (c *Client) GenerateAddress(ctx context.Context, walletID, coin string, options *AddressOptions) (*Address, error) {
	if walletID == "" {
		return nil, fmt.Errorf("wallet ID is required")
	}
	if coin == "" {
		return nil, fmt.Errorf("coin is required")
	}

	path := fmt.Sprintf("/%s/wallet/%s/address", coin, walletID)

	body := map[string]interface{}{}
	if options != nil {
		if options.Chain != nil {
			body["chain"] = *options.Chain
		}
		if options.AddressType != "" {
			body["addressType"] = options.AddressType
		}
		if options.Label != "" {
			body["label"] = options.Label
		}
	}

	resp, err := c.makeRequest(ctx, RequestOptions{
		Method: http.MethodPost,
		Path:   path,
		Body:   body,
		Headers: map[string]string{
			"Accept": "application/json",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate address: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var address Address
	if err := json.Unmarshal(responseBody, &address); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	c.logger.Info("Generated new address",
		"wallet_id", walletID,
		"coin", coin,
		"address", address.Address,
	)

	return &address, nil
}

// AddressOptions holds options for generating addresses
type AddressOptions struct {
	Chain       *int   `json:"chain,omitempty"`
	AddressType string `json:"addressType,omitempty"`
	Label       string `json:"label,omitempty"`
}

// ListWalletAddresses retrieves addresses associated with a wallet
func (c *Client) ListWalletAddresses(ctx context.Context, walletID, coin string, options *AddressListOptions) (*AddressListResponse, error) {
	if walletID == "" {
		return nil, fmt.Errorf("wallet ID is required")
	}
	if coin == "" {
		return nil, fmt.Errorf("coin is required")
	}

	path := fmt.Sprintf("/%s/wallet/%s/addresses", coin, walletID)

	resp, err := c.makeRequest(ctx, RequestOptions{
		Method: http.MethodGet,
		Path:   path,
		Headers: map[string]string{
			"Accept": "application/json",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list addresses: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var result AddressListResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	c.logger.Info("Listed wallet addresses successfully",
		"wallet_id", walletID,
		"coin", coin,
		"count", len(result.Addresses),
	)

	return &result, nil
}

// AddressListOptions holds options for listing addresses
type AddressListOptions struct {
	Limit int `json:"limit,omitempty"`
	Skip  int `json:"skip,omitempty"`
	Chain int `json:"chain,omitempty"`
}

// AddressListResponse represents the response from listing addresses
type AddressListResponse struct {
	Addresses       []Address `json:"addresses"`
	Count           int       `json:"count"`
	Total           int       `json:"total"`
	NextBatchPrevId string    `json:"nextBatchPrevId,omitempty"`
}
