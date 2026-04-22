package transaction

import (
	"context"
	"errors"
	"testing"
	"time"

	"gobank/internal/domain"
)

type mockAccountRepo struct {
	items map[string]domain.Account
}

func (m *mockAccountRepo) Create(_ context.Context, account domain.Account) (domain.Account, error) {
	m.items[account.ID] = account
	return account, nil
}

func (m *mockAccountRepo) FindByID(_ context.Context, id string) (domain.Account, error) {
	account, ok := m.items[id]
	if !ok {
		return domain.Account{}, errors.New("not found")
	}
	return account, nil
}

func (m *mockAccountRepo) FindByUserID(_ context.Context, userID string) ([]domain.Account, error) {
	result := make([]domain.Account, 0)
	for _, account := range m.items {
		if account.UserID == userID {
			result = append(result, account)
		}
	}
	return result, nil
}

func (m *mockAccountRepo) Update(_ context.Context, account domain.Account) (domain.Account, error) {
	m.items[account.ID] = account
	return account, nil
}

type mockTransactionRepo struct {
	items map[string]domain.Transaction
	refs  map[string]string
}

func (m *mockTransactionRepo) Create(_ context.Context, tx domain.Transaction) (domain.Transaction, error) {
	if m.items == nil {
		m.items = make(map[string]domain.Transaction)
		m.refs = make(map[string]string)
	}
	m.items[tx.ID] = tx
	m.refs[tx.ReferenceID] = tx.ID
	return tx, nil
}

func (m *mockTransactionRepo) FindByID(_ context.Context, id string) (domain.Transaction, error) {
	return m.items[id], nil
}

func (m *mockTransactionRepo) FindByReferenceID(_ context.Context, referenceID string) (domain.Transaction, error) {
	id, ok := m.refs[referenceID]
	if !ok {
		return domain.Transaction{}, errors.New("not found")
	}
	return m.items[id], nil
}

func (m *mockTransactionRepo) ListByAccountID(_ context.Context, accountID string) ([]domain.Transaction, error) {
	result := make([]domain.Transaction, 0)
	for _, tx := range m.items {
		if tx.SourceID == accountID || tx.DestinationID == accountID {
			result = append(result, tx)
		}
	}
	return result, nil
}

type mockAuditRepo struct{}

func (m *mockAuditRepo) Create(context.Context, domain.AuditLog) error {
	return nil
}

func (m *mockAuditRepo) List(context.Context, string) ([]domain.AuditLog, error) {
	return nil, nil
}

type mockIdempotencyRepo struct {
	reserved map[string]struct{}
}

func (m *mockIdempotencyRepo) Reserve(_ context.Context, key string) (bool, error) {
	if m.reserved == nil {
		m.reserved = make(map[string]struct{})
	}
	if _, ok := m.reserved[key]; ok {
		return false, nil
	}
	m.reserved[key] = struct{}{}
	return true, nil
}

func (m *mockIdempotencyRepo) Release(_ context.Context, key string) error {
	delete(m.reserved, key)
	return nil
}

func TestTransfer(t *testing.T) {
	accountRepo := &mockAccountRepo{
		items: map[string]domain.Account{
			"acc_1": {ID: "acc_1", UserID: "usr_1", Balance: 100_000, UpdatedAt: time.Now()},
			"acc_2": {ID: "acc_2", UserID: "usr_2", Balance: 10_000, UpdatedAt: time.Now()},
		},
	}
	uc := NewUseCase(accountRepo, &mockTransactionRepo{}, &mockAuditRepo{}, &mockIdempotencyRepo{})

	tx, err := uc.Transfer(context.Background(), TransferInput{
		ReferenceID:   "ref-1",
		SourceID:      "acc_1",
		DestinationID: "acc_2",
		Amount:        25_000,
		ActorID:       "usr_1",
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if tx.Amount != 25_000 {
		t.Fatalf("expected amount 25000, got %d", tx.Amount)
	}
	if accountRepo.items["acc_1"].Balance != 75_000 {
		t.Fatalf("unexpected source balance %d", accountRepo.items["acc_1"].Balance)
	}
	if accountRepo.items["acc_2"].Balance != 35_000 {
		t.Fatalf("unexpected destination balance %d", accountRepo.items["acc_2"].Balance)
	}
}
