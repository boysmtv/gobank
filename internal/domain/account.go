package domain

import (
	"time"

	"github.com/google/uuid"
)

type AccountType string

const (
	AccountTypeSavings AccountType = "savings"
	AccountTypeCurrent AccountType = "current"
)

type AccountStatus string

const (
	AccountStatusActive AccountStatus = "active"
	AccountStatusFrozen AccountStatus = "frozen"
	AccountStatusClosed AccountStatus = "closed"
)

// Account represents a bank account with exact integer balance in minor currency units (e.g. cents).
type Account struct {
	ID            uuid.UUID     `json:"id"`
	UserID        uuid.UUID     `json:"user_id"`
	AccountNumber string        `json:"account_number"`
	Type          AccountType   `json:"type"`
	Currency      string        `json:"currency"`
	Balance       int64         `json:"balance"` // in minor units (cents)
	Status        AccountStatus `json:"status"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
}

type CreateAccountRequest struct {
	Type     AccountType `json:"type"     validate:"required,oneof=savings current"`
	Currency string      `json:"currency" validate:"required,len=3"`
}
