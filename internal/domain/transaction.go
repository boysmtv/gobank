package domain

import (
	"time"

	"github.com/google/uuid"
)

type TransactionType string

const (
	TransactionTypeTransfer TransactionType = "transfer"
	TransactionTypeDeposit  TransactionType = "deposit"
	TransactionTypeWithdraw TransactionType = "withdrawal"
)

type TransactionStatus string

const (
	TransactionStatusPending TransactionStatus = "pending"
	TransactionStatusSuccess TransactionStatus = "success"
	TransactionStatusFailed  TransactionStatus = "failed"
)

// Transaction records a financial movement.
type Transaction struct {
	ID             uuid.UUID         `json:"id"`
	IdempotencyKey string            `json:"idempotency_key"`
	FromAccountID  *uuid.UUID        `json:"from_account_id,omitempty"`
	ToAccountID    *uuid.UUID        `json:"to_account_id,omitempty"`
	Amount         int64             `json:"amount"` // minor units
	Currency       string            `json:"currency"`
	Type           TransactionType   `json:"type"`
	Status         TransactionStatus `json:"status"`
	Description    string            `json:"description"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

type TransferRequest struct {
	IdempotencyKey string `json:"idempotency_key" validate:"required,min=8,max=64"`
	FromAccountID  string `json:"from_account_id"  validate:"required,uuid"`
	ToAccountID    string `json:"to_account_id"    validate:"required,uuid"`
	Amount         int64  `json:"amount"           validate:"required,min=1"`
	Currency       string `json:"currency"         validate:"required,len=3"`
	Description    string `json:"description"      validate:"max=255"`
}

type TransactionListRequest struct {
	AccountID string `validate:"required,uuid"`
	Page      int    `validate:"min=1"`
	PageSize  int    `validate:"min=1,max=100"`
}
