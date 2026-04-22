package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourorg/gobank/internal/repository"
)

type TxManager struct {
	db *pgxpool.Pool
}

func NewTxManager(db *pgxpool.Pool) *TxManager {
	return &TxManager{db: db}
}

func (m *TxManager) WithTransaction(ctx context.Context, fn func(ctx context.Context, tx repository.DBTX) error) error {
	pgxTx, err := m.db.BeginTx(ctx, pgx.TxOptions{
		IsoLevel:   pgx.Serializable,
		AccessMode: pgx.ReadWrite,
	})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = pgxTx.Rollback(ctx)
			panic(p)
		}
	}()

	if err := fn(ctx, pgxTx); err != nil {
		if rbErr := pgxTx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("rollback failed: %v, original: %w", rbErr, err)
		}
		return err
	}

	return pgxTx.Commit(ctx)
}
