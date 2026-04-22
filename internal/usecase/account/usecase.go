package account

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gobank/internal/domain"
	"gobank/internal/repository"
)

var ErrCurrencyRequired = errors.New("currency is required")

type UseCase struct {
	accounts repository.AccountRepository
	audits   repository.AuditRepository
}

type CreateInput struct {
	UserID   string
	Type     domain.AccountType
	Currency string
}

func NewUseCase(accounts repository.AccountRepository, audits repository.AuditRepository) *UseCase {
	return &UseCase{
		accounts: accounts,
		audits:   audits,
	}
}

func (u *UseCase) Create(ctx context.Context, input CreateInput) (domain.Account, error) {
	if input.Currency == "" {
		return domain.Account{}, ErrCurrencyRequired
	}

	now := time.Now().UTC()
	account := domain.Account{
		ID:        fmt.Sprintf("acc_%d", now.UnixNano()),
		UserID:    input.UserID,
		Number:    fmt.Sprintf("10%d", now.UnixNano()),
		Type:      input.Type,
		Currency:  input.Currency,
		Balance:   0,
		Active:    true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	created, err := u.accounts.Create(ctx, account)
	if err != nil {
		return domain.Account{}, err
	}

	_ = u.audits.Create(ctx, domain.AuditLog{
		ID:         fmt.Sprintf("aud_%d", now.UnixNano()),
		ActorID:    input.UserID,
		Action:     "account.created",
		Resource:   "account",
		ResourceID: created.ID,
		CreatedAt:  now,
	})

	return created, nil
}

func (u *UseCase) GetByID(ctx context.Context, id string) (domain.Account, error) {
	return u.accounts.FindByID(ctx, id)
}

func (u *UseCase) ListByUser(ctx context.Context, userID string) ([]domain.Account, error) {
	return u.accounts.FindByUserID(ctx, userID)
}
