package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/yourorg/gobank/internal/domain"
)

// UserRepository defines all persistence operations for users.
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.UserStatus) error
	SaveRefreshToken(ctx context.Context, token *domain.RefreshToken) error
	GetRefreshToken(ctx context.Context, id uuid.UUID) (*domain.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, id uuid.UUID) error
	RevokeAllUserRefreshTokens(ctx context.Context, userID uuid.UUID) error
}

// AccountRepository defines all persistence operations for accounts.
type AccountRepository interface {
	Create(ctx context.Context, account *domain.Account) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Account, error)
	GetByAccountNumber(ctx context.Context, number string) (*domain.Account, error)
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Account, error)
	// UpdateBalanceTx executes a balance update within a provided transaction context.
	UpdateBalance(ctx context.Context, tx DBTX, accountID uuid.UUID, delta int64) error
	LockForUpdate(ctx context.Context, tx DBTX, accountID uuid.UUID) (*domain.Account, error)
}

// TransactionRepository defines persistence for financial transactions.
type TransactionRepository interface {
	Create(ctx context.Context, tx DBTX, txn *domain.Transaction) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Transaction, error)
	GetByIdempotencyKey(ctx context.Context, key string) (*domain.Transaction, error)
	ListByAccountID(ctx context.Context, accountID uuid.UUID, page, pageSize int) ([]domain.Transaction, int, error)
	UpdateStatus(ctx context.Context, tx DBTX, id uuid.UUID, status domain.TransactionStatus) error
}

// AuditRepository is append-only. No updates, no deletes.
type AuditRepository interface {
	Create(ctx context.Context, log *domain.AuditLog) error
	ListByUserID(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]domain.AuditLog, error)
}

// KYCRepository manages KYC records.
type KYCRepository interface {
	Create(ctx context.Context, record *domain.KYCRecord) error
	GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.KYCRecord, error)
	UpdateStatus(ctx context.Context, userID uuid.UUID, status domain.KYCStatus, note string) error
}

// IdempotencyStore handles idempotency key locking and retrieval.
type IdempotencyStore interface {
	// Set stores the idempotency key with expiry. Returns false if key already exists.
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error)
	Get(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
}

// DBTX is the minimal interface for a DB transaction handle, fulfilled by pgx.Tx.
type DBTX interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// TxManager wraps transaction lifecycle management.
type TxManager interface {
	WithTransaction(ctx context.Context, fn func(ctx context.Context, tx DBTX) error) error
}
