package domain

import "time"

type AccountType string

const (
	AccountTypeSavings AccountType = "savings"
	AccountTypeWallet  AccountType = "wallet"
)

type Account struct {
	ID        string
	UserID    string
	Number    string
	Type      AccountType
	Currency  string
	Balance   int64
	Active    bool
	CreatedAt time.Time
	UpdatedAt time.Time
}
