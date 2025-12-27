package models

import (
	"time"

	"github.com/google/uuid"
)

type WalletMembership struct {
	ID          uuid.UUID `json:"id" db:"id"`
	WalletID    uuid.UUID `json:"wallet_id" db:"wallet_id"`
	UserID      uuid.UUID `json:"user_id" db:"user_id"`
	Role        string    `json:"role" db:"role"`
	Permissions JSON      `json:"permissions" db:"permissions"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type WalletRole string

const (
	WalletRoleViewer  WalletRole = "viewer"
	WalletRoleSpender WalletRole = "spender"
	WalletRoleAdmin   WalletRole = "admin"
)
