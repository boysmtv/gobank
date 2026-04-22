package account

import (
	"context"
	"testing"

	"gobank/internal/domain"
)

type mockAccountRepo struct {
	items map[string]domain.Account
}

func (m *mockAccountRepo) Create(_ context.Context, account domain.Account) (domain.Account, error) {
	if m.items == nil {
		m.items = make(map[string]domain.Account)
	}
	m.items[account.ID] = account
	return account, nil
}

func (m *mockAccountRepo) FindByID(_ context.Context, id string) (domain.Account, error) {
	return m.items[id], nil
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

type mockAuditRepo struct{}

func (m *mockAuditRepo) Create(context.Context, domain.AuditLog) error {
	return nil
}

func (m *mockAuditRepo) List(context.Context, string) ([]domain.AuditLog, error) {
	return nil, nil
}

func TestCreate(t *testing.T) {
	uc := NewUseCase(&mockAccountRepo{}, &mockAuditRepo{})

	account, err := uc.Create(context.Background(), CreateInput{
		UserID:   "usr_1",
		Type:     domain.AccountTypeSavings,
		Currency: "IDR",
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if account.UserID != "usr_1" {
		t.Fatalf("expected user id usr_1, got %s", account.UserID)
	}
}
