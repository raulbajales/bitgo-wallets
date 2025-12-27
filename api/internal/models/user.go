package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID `json:"id" db:"id"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"` // Don't serialize password
	FirstName    *string   `json:"first_name" db:"first_name"`
	LastName     *string   `json:"last_name" db:"last_name"`
	Role         string    `json:"role" db:"role"`
	IsActive     bool      `json:"is_active" db:"is_active"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

type UserRole string

const (
	RoleEndUser  UserRole = "end_user"
	RoleOperator UserRole = "operator"
	RoleApprover UserRole = "approver"
	RoleAdmin    UserRole = "admin"
)
