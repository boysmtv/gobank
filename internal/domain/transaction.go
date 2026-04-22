package domain

import "time"

type TransactionType string

const (
	TransactionTypeCredit   TransactionType = "credit"
	TransactionTypeDebit    TransactionType = "debit"
	TransactionTypeTransfer TransactionType = "transfer"
)

type TransactionStatus string

const (
	TransactionStatusPending   TransactionStatus = "pending"
	TransactionStatusCompleted TransactionStatus = "completed"
	TransactionStatusFailed    TransactionStatus = "failed"
)

type Transaction struct {
	ID            string
	ReferenceID   string
	SourceID      string
	DestinationID string
	Type          TransactionType
	Status        TransactionStatus
	Amount        int64
	Description   string
	CreatedAt     time.Time
}
