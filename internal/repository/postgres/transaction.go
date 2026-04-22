package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourorg/gobank/internal/domain"
	"github.com/yourorg/gobank/internal/repository"
)

type TransactionRepo struct {
	db *pgxpool.Pool
}

func NewTransactionRepo(db *pgxpool.Pool) *TransactionRepo {
	return &TransactionRepo{db: db}
}

func (r *TransactionRepo) Create(ctx context.Context, tx repository.DBTX, txn *domain.Transaction) error {
	meta, _ := json.Marshal(txn.Metadata)
	const q = `
        INSERT INTO transactions (id, idempotency_key, from_account_id, to_account_id, amount, currency, type, status, description, metadata, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	pgxTx := tx.(pgx.Tx)
	_, err := pgxTx.Exec(ctx, q,
		txn.ID, txn.IdempotencyKey, txn.FromAccountID, txn.ToAccountID,
		txn.Amount, txn.Currency, txn.Type, txn.Status,
		txn.Description, meta, txn.CreatedAt, txn.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("transactionRepo.Create: %w", err)
	}
	return nil
}

func (r *TransactionRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Transaction, error) {
	const q = `
        SELECT id, idempotency_key, from_account_id, to_account_id, amount, currency, type, status, description, metadata, created_at, updated_at
        FROM transactions WHERE id = $1`
	return r.scan(r.db.QueryRow(ctx, q, id))
}

func (r *TransactionRepo) GetByIdempotencyKey(ctx context.Context, key string) (*domain.Transaction, error) {
	const q = `
        SELECT id, idempotency_key, from_account_id, to_account_id, amount, currency, type, status, description, metadata, created_at, updated_at
        FROM transactions WHERE idempotency_key = $1`
	return r.scan(r.db.QueryRow(ctx, q, key))
}

func (r *TransactionRepo) ListByAccountID(ctx context.Context, accountID uuid.UUID, page, pageSize int) ([]domain.Transaction, int, error) {
	offset := (page - 1) * pageSize
	const countQ = `SELECT COUNT(*) FROM transactions WHERE from_account_id = $1 OR to_account_id = $1`
	var total int
	if err := r.db.QueryRow(ctx, countQ, accountID).Scan(&total); err != nil {
		return nil, 0, err
	}

	const q = `
        SELECT id, idempotency_key, from_account_id, to_account_id, amount, currency, type, status, description, metadata, created_at, updated_at
        FROM transactions
        WHERE from_account_id = $1 OR to_account_id = $1
        ORDER BY created_at DESC
        LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, q, accountID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var txns []domain.Transaction
	for rows.Next() {
		t, err := r.scan(rows)
		if err != nil {
			return nil, 0, err
		}
		txns = append(txns, *t)
	}
	return txns, total, rows.Err()
}

func (r *TransactionRepo) UpdateStatus(ctx context.Context, tx repository.DBTX, id uuid.UUID, status domain.TransactionStatus) error {
	const q = `UPDATE transactions SET status = $1, updated_at = NOW() WHERE id = $2`
	pgxTx := tx.(pgx.Tx)
	_, err := pgxTx.Exec(ctx, q, status, id)
	return err
}

func (r *TransactionRepo) scan(row scanner) (*domain.Transaction, error) {
	t := &domain.Transaction{}
	var meta []byte
	err := row.Scan(
		&t.ID, &t.IdempotencyKey, &t.FromAccountID, &t.ToAccountID,
		&t.Amount, &t.Currency, &t.Type, &t.Status,
		&t.Description, &meta, &t.CreatedAt, &t.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if len(meta) > 0 {
		_ = json.Unmarshal(meta, &t.Metadata)
	}
	return t, nil
}
