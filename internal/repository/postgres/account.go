package postgres

import (
	"context"
	"sync"

	"gobank/internal/domain"
)

type AccountRepository struct {
	mu        sync.RWMutex
	byID      map[string]domain.Account
	byUserID  map[string][]string
	byAccount map[string]string
}

func NewAccountRepository() *AccountRepository {
	return &AccountRepository{
		byID:      make(map[string]domain.Account),
		byUserID:  make(map[string][]string),
		byAccount: make(map[string]string),
	}
}

func (r *AccountRepository) Create(_ context.Context, account domain.Account) (domain.Account, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.byID[account.ID] = account
	r.byUserID[account.UserID] = append(r.byUserID[account.UserID], account.ID)
	r.byAccount[account.Number] = account.ID
	return account, nil
}

func (r *AccountRepository) FindByID(_ context.Context, id string) (domain.Account, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	account, ok := r.byID[id]
	if !ok {
		return domain.Account{}, errRecordNotFound
	}

	return account, nil
}

func (r *AccountRepository) FindByUserID(_ context.Context, userID string) ([]domain.Account, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := r.byUserID[userID]
	accounts := make([]domain.Account, 0, len(ids))
	for _, id := range ids {
		accounts = append(accounts, r.byID[id])
	}

	return accounts, nil
}

func (r *AccountRepository) Update(_ context.Context, account domain.Account) (domain.Account, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.byID[account.ID]; !ok {
		return domain.Account{}, errRecordNotFound
	}

	r.byID[account.ID] = account
	r.byAccount[account.Number] = account.ID
	return account, nil
}
