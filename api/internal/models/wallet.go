package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type Wallet struct {
	ID                     uuid.UUID      `json:"id" db:"id"`
	OrganizationID         uuid.UUID      `json:"organization_id" db:"organization_id"`
	BitgoWalletID          string         `json:"bitgo_wallet_id" db:"bitgo_wallet_id"`
	Label                  string         `json:"label" db:"label"`
	Coin                   string         `json:"coin" db:"coin"`
	WalletType             WalletType     `json:"wallet_type" db:"wallet_type"`
	BalanceString          string         `json:"balance_string" db:"balance_string"`
	ConfirmedBalanceString string         `json:"confirmed_balance_string" db:"confirmed_balance_string"`
	SpendableBalanceString string         `json:"spendable_balance_string" db:"spendable_balance_string"`
	IsActive               bool           `json:"is_active" db:"is_active"`
	Frozen                 bool           `json:"frozen" db:"frozen"`
	MultisigType           *string        `json:"multisig_type" db:"multisig_type"`
	Threshold              int            `json:"threshold" db:"threshold"`
	Tags                   pq.StringArray `json:"tags" db:"tags"`
	Metadata               JSON           `json:"metadata" db:"metadata"`
	CreatedAt              time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time      `json:"updated_at" db:"updated_at"`
}

type WalletType string

const (
	WalletTypeCustodial WalletType = "custodial"
	WalletTypeHot       WalletType = "hot"
	WalletTypeWarm      WalletType = "warm"
	WalletTypeCold      WalletType = "cold"
)

// JSON type for handling JSONB in PostgreSQL
type JSON map[string]interface{}

func (j JSON) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, j)
}
