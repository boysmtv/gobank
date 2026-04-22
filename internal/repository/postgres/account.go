package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourorg/gobank/internal/domain"
	"github.com/yourorg/gobank/internal/repository"
)

type AccountRepo struct {
	db *pgxpool.Pool
}

func NewAccountRepo(db *pgxpool.Pool) *AccountRepo {
	return &AccountRepo{db: db}
}

func (r *AccountRepo) Create(ctx context.Context, account *domain.Account) error {
	const q = `
        INSERT INTO accounts (id, user_id, account_number, type, currency, balance, status, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := r.db.Exec(ctx, q,
		account.ID, account.UserID, account.AccountNumber, account.Type,
		account.Currency, account.Balance, account.Status,
		account.CreatedAt, account.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("accountRepo.Create: %w", err)
	}
	return nil
}

func (r *AccountRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Account, error) {
	const q = `
        SELECT id, user_id, account_number, type, currency, balance, status, created_at, updated_at
        FROM accounts WHERE id = $1`
	return r.scan(r.db.QueryRow(ctx, q, id))
}

func (r *AccountRepo) GetByAccountNumber(ctx context.Context, number string) (*domain.Account, error) {
	const q = `
        SELECT id, user_id, account_number, type, currency, balance, status, created_at, updated_at
        FROM accounts WHERE account_number = $1`
	return r.scan(r.db.QueryRow(ctx, q, number))
}

func (r *AccountRepo) ListByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Account, error) {
	const q = `
        SELECT id, user_id, account_number, type, currency, balance, status, created_at, updated_at
        FROM accounts WHERE user_id = $1 ORDER BY created_at ASC`
	rows, err := r.db.Query(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []domain.Account
	for rows.Next() {
		a, err := r.scan(rows)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, *a)
	}
	return accounts, rows.Err()
}

// LockForUpdate locks the account row for the duration of a database transaction.
// tx must be a *pgx.Tx obtained via the pool.
func (r *AccountRepo) LockForUpdate(ctx context.Context, tx repository.DBTX, id uuid.UUID) (*domain.Account, error) {
	const q = `
        SELECT id, user_id, account_number, type, currency, balance, status, created_at, updated_at
        FROM accounts WHERE id = $1 FOR UPDATE`
	pgxTx := tx.(pgx.Tx)
	return r.scan(pgxTx.QueryRow(ctx, q, id))
}

func (r *AccountRepo) UpdateBalance(ctx context.Context, tx repository.DBTX, accountID uuid.UUID, delta int64) error {
	const q = `
        UPDATE accounts
        SET balance = balance + $1, updated_at = NOW()
        WHERE id = $2`
	pgxTx := tx.(pgx.Tx)
	_, err := pgxTx.Exec(ctx, q, delta, accountID)
	return err
}

type scanner interface {
	Scan(dest ...any) error
}

func (r *AccountRepo) scan(row scanner) (*domain.Account, error) {
	a := &domain.Account{}
	err := row.Scan(
		&a.ID, &a.UserID, &a.AccountNumber, &a.Type,
		&a.Currency, &a.Balance, &a.Status, &a.CreatedAt, &a.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return a, nil
}
