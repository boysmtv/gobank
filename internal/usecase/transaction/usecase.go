package transaction

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gobank/internal/domain"
	"gobank/internal/repository"
)

var (
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrInvalidAmount     = errors.New("amount must be greater than zero")
	ErrDuplicateRequest  = errors.New("duplicate transfer request")
)

type UseCase struct {
	accounts     repository.AccountRepository
	transactions repository.TransactionRepository
	audits       repository.AuditRepository
	idempotency  repository.IdempotencyRepository
}

type TransferInput struct {
	ReferenceID   string
	SourceID      string
	DestinationID string
	Amount        int64
	Description   string
	ActorID       string
}

func NewUseCase(
	accounts repository.AccountRepository,
	transactions repository.TransactionRepository,
	audits repository.AuditRepository,
	idempotency repository.IdempotencyRepository,
) *UseCase {
	return &UseCase{
		accounts:     accounts,
		transactions: transactions,
		audits:       audits,
		idempotency:  idempotency,
	}
}

func (u *UseCase) Transfer(ctx context.Context, input TransferInput) (domain.Transaction, error) {
	if input.Amount <= 0 {
		return domain.Transaction{}, ErrInvalidAmount
	}

	reserved, err := u.idempotency.Reserve(ctx, input.ReferenceID)
	if err != nil {
		return domain.Transaction{}, err
	}
	if !reserved {
		tx, existingErr := u.transactions.FindByReferenceID(ctx, input.ReferenceID)
		if existingErr == nil {
			return tx, nil
		}
		return domain.Transaction{}, ErrDuplicateRequest
	}

	source, err := u.accounts.FindByID(ctx, input.SourceID)
	if err != nil {
		_ = u.idempotency.Release(ctx, input.ReferenceID)
		return domain.Transaction{}, err
	}

	destination, err := u.accounts.FindByID(ctx, input.DestinationID)
	if err != nil {
		_ = u.idempotency.Release(ctx, input.ReferenceID)
		return domain.Transaction{}, err
	}

	if source.Balance < input.Amount {
		_ = u.idempotency.Release(ctx, input.ReferenceID)
		return domain.Transaction{}, ErrInsufficientFunds
	}

	now := time.Now().UTC()
	source.Balance -= input.Amount
	source.UpdatedAt = now
	destination.Balance += input.Amount
	destination.UpdatedAt = now

	if _, err := u.accounts.Update(ctx, source); err != nil {
		_ = u.idempotency.Release(ctx, input.ReferenceID)
		return domain.Transaction{}, err
	}
	if _, err := u.accounts.Update(ctx, destination); err != nil {
		_ = u.idempotency.Release(ctx, input.ReferenceID)
		return domain.Transaction{}, err
	}

	tx := domain.Transaction{
		ID:            fmt.Sprintf("trx_%d", now.UnixNano()),
		ReferenceID:   input.ReferenceID,
		SourceID:      input.SourceID,
		DestinationID: input.DestinationID,
		Type:          domain.TransactionTypeTransfer,
		Status:        domain.TransactionStatusCompleted,
		Amount:        input.Amount,
		Description:   input.Description,
		CreatedAt:     now,
	}

	created, err := u.transactions.Create(ctx, tx)
	if err != nil {
		_ = u.idempotency.Release(ctx, input.ReferenceID)
		return domain.Transaction{}, err
	}

	_ = u.audits.Create(ctx, domain.AuditLog{
		ID:         fmt.Sprintf("aud_%d", now.UnixNano()),
		ActorID:    input.ActorID,
		Action:     "transaction.transfer",
		Resource:   "transaction",
		ResourceID: created.ID,
		CreatedAt:  now,
	})

	return created, nil
}

func (u *UseCase) History(ctx context.Context, accountID string) ([]domain.Transaction, error) {
	return u.transactions.ListByAccountID(ctx, accountID)
}
