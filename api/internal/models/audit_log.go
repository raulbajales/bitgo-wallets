package models

import (
	"net"
	"time"

	"github.com/google/uuid"
)

type AuditLog struct {
	ID                uuid.UUID  `json:"id" db:"id"`
	UserID            *uuid.UUID `json:"user_id" db:"user_id"`
	OrganizationID    *uuid.UUID `json:"organization_id" db:"organization_id"`
	WalletID          *uuid.UUID `json:"wallet_id" db:"wallet_id"`
	TransferRequestID *uuid.UUID `json:"transfer_request_id" db:"transfer_request_id"`
	Action            string     `json:"action" db:"action"`
	ResourceType      string     `json:"resource_type" db:"resource_type"`
	ResourceID        *string    `json:"resource_id" db:"resource_id"`
	OldValues         JSON       `json:"old_values" db:"old_values"`
	NewValues         JSON       `json:"new_values" db:"new_values"`
	Metadata          JSON       `json:"metadata" db:"metadata"`
	IPAddress         *net.IP    `json:"ip_address" db:"ip_address"`
	UserAgent         *string    `json:"user_agent" db:"user_agent"`
	CorrelationID     *uuid.UUID `json:"correlation_id" db:"correlation_id"`
	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
}
