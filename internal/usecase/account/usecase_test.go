package account_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/yourorg/gobank/internal/domain"
	"github.com/yourorg/gobank/internal/repository"
	"github.com/yourorg/gobank/internal/repository/postgres"
	accountUC "github.com/yourorg/gobank/internal/usecase/account"
	"go.uber.org/zap"
)

// --- Mock Account Repository ---

type mockAccountRepo struct {
	accounts map[uuid.UUID]*domain.Account
}

func newMockAccountRepo() *mockAccountRepo {
	return &mockAccountRepo{accounts: make(map[uuid.UUID]*domain.Account)}
}

func (m *mockAccountRepo) Create(ctx context.Context, a *domain.Account) error {
	m.accounts[a.ID] = a
	return nil
}

func (m *mockAccountRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Account, error) {
	a, ok := m.accounts[id]
	if !ok {
		return nil, postgres.ErrNotFound
	}
	return a, nil
}

func (m *mockAccountRepo) GetByAccountNumber(ctx context.Context, number string) (*domain.Account, error) {
	for _, a := range m.accounts {
		if a.AccountNumber == number {
			return a, nil
		}
	}
	return nil, postgres.ErrNotFound
}

func (m *mockAccountRepo) ListByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Account, error) {
	var result []domain.Account
	for _, a := range m.accounts {
		if a.UserID == userID {
			result = append(result, *a)
		}
	}
	return result, nil
}

func (m *mockAccountRepo) LockForUpdate(ctx context.Context, tx repository.DBTX, id uuid.UUID) (*domain.Account, error) {
	return m.GetByID(ctx, id)
}

func (m *mockAccountRepo) UpdateBalance(ctx context.Context, tx repository.DBTX, id uuid.UUID, delta int64) error {
	if a, ok := m.accounts[id]; ok {
		a.Balance += delta
	}
	return nil
}

// --- Mock Publisher ---

type mockPublisher struct{}

func (m *mockPublisher) Publish(_ context.Context, _ string, _ interface{}) error {
	return nil
}

// --- Helpers ---

func newTestUseCase(repo *mockAccountRepo) *accountUC.UseCase {
	log, _ := zap.NewDevelopment()
	return accountUC.New(repo, &mockPublisher{}, log)
}

// --- Tests ---

func TestCreateAccount_Success(t *testing.T) {
	repo := newMockAccountRepo()
	uc := newTestUseCase(repo)
	userID := uuid.New()

	acct, err := uc.CreateAccount(context.Background(), userID, domain.CreateAccountRequest{
		Type:     domain.AccountTypeSavings,
		Currency: "USD",
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if acct.UserID != userID {
		t.Errorf("expected userID %s, got %s", userID, acct.UserID)
	}
	if acct.Type != domain.AccountTypeSavings {
		t.Errorf("expected savings, got %s", acct.Type)
	}
	if acct.Balance != 0 {
		t.Errorf("expected initial balance 0, got %d", acct.Balance)
	}
	if acct.Status != domain.AccountStatusActive {
		t.Errorf("expected status active, got %s", acct.Status)
	}
	if acct.AccountNumber == "" {
		t.Error("expected non-empty account number")
	}
	if acct.Currency != "USD" {
		t.Errorf("expected currency USD, got %s", acct.Currency)
	}
}

func TestCreateAccount_MultipleTypes(t *testing.T) {
	repo := newMockAccountRepo()
	uc := newTestUseCase(repo)
	userID := uuid.New()

	types := []domain.AccountType{
		domain.AccountTypeSavings,
		domain.AccountTypeCurrent,
	}

	for _, acctType := range types {
		acct, err := uc.CreateAccount(context.Background(), userID, domain.CreateAccountRequest{
			Type:     acctType,
			Currency: "USD",
		})
		if err != nil {
			t.Fatalf("expected no error for type %s, got: %v", acctType, err)
		}
		if acct.Type != acctType {
			t.Errorf("expected type %s, got %s", acctType, acct.Type)
		}
	}

	accounts, _ := repo.ListByUserID(context.Background(), userID)
	if len(accounts) != 2 {
		t.Errorf("expected 2 accounts, got %d", len(accounts))
	}
}

func TestCreateAccount_MaxAccountsReached(t *testing.T) {
	repo := newMockAccountRepo()
	uc := newTestUseCase(repo)
	userID := uuid.New()

	// Create 5 accounts (max allowed)
	for i := 0; i < 5; i++ {
		_, err := uc.CreateAccount(context.Background(), userID, domain.CreateAccountRequest{
			Type:     domain.AccountTypeSavings,
			Currency: "USD",
		})
		if err != nil {
			t.Fatalf("expected no error on creation %d, got: %v", i+1, err)
		}
	}

	// 6th account should fail
	_, err := uc.CreateAccount(context.Background(), userID, domain.CreateAccountRequest{
		Type:     domain.AccountTypeSavings,
		Currency: "USD",
	})
	if err != accountUC.ErrMaxAccountsReached {
		t.Errorf("expected ErrMaxAccountsReached, got: %v", err)
	}
}

func TestCreateAccount_AccountNumberIsUnique(t *testing.T) {
	repo := newMockAccountRepo()
	uc := newTestUseCase(repo)
	userID := uuid.New()

	seen := make(map[string]bool)
	for i := 0; i < 5; i++ {
		acct, err := uc.CreateAccount(context.Background(), userID, domain.CreateAccountRequest{
			Type:     domain.AccountTypeSavings,
			Currency: "USD",
		})
		if err != nil {
			t.Fatalf("creation %d failed: %v", i+1, err)
		}
		if seen[acct.AccountNumber] {
			t.Errorf("duplicate account number generated: %s", acct.AccountNumber)
		}
		seen[acct.AccountNumber] = true
	}
}

func TestGetAccount_Success(t *testing.T) {
	repo := newMockAccountRepo()
	uc := newTestUseCase(repo)
	userID := uuid.New()

	created, err := uc.CreateAccount(context.Background(), userID, domain.CreateAccountRequest{
		Type:     domain.AccountTypeCurrent,
		Currency: "EUR",
	})
	if err != nil {
		t.Fatal(err)
	}

	fetched, err := uc.GetAccount(context.Background(), userID, created.ID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if fetched.ID != created.ID {
		t.Errorf("expected ID %s, got %s", created.ID, fetched.ID)
	}
}

func TestGetAccount_ForbiddenForOtherUser(t *testing.T) {
	repo := newMockAccountRepo()
	uc := newTestUseCase(repo)

	ownerID := uuid.New()
	intruderID := uuid.New()

	created, err := uc.CreateAccount(context.Background(), ownerID, domain.CreateAccountRequest{
		Type:     domain.AccountTypeSavings,
		Currency: "USD",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = uc.GetAccount(context.Background(), intruderID, created.ID)
	if err != accountUC.ErrForbidden {
		t.Errorf("expected ErrForbidden, got: %v", err)
	}
}

func TestGetAccount_NotFound(t *testing.T) {
	repo := newMockAccountRepo()
	uc := newTestUseCase(repo)
	userID := uuid.New()

	_, err := uc.GetAccount(context.Background(), userID, uuid.New())
	if err == nil {
		t.Error("expected error for non-existent account, got nil")
	}
}

func TestListAccounts_Empty(t *testing.T) {
	repo := newMockAccountRepo()
	uc := newTestUseCase(repo)
	userID := uuid.New()

	accounts, err := uc.ListAccounts(context.Background(), userID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(accounts) != 0 {
		t.Errorf("expected 0 accounts, got %d", len(accounts))
	}
}

func TestListAccounts_IsolatedPerUser(t *testing.T) {
	repo := newMockAccountRepo()
	uc := newTestUseCase(repo)

	user1 := uuid.New()
	user2 := uuid.New()

	// user1 gets 3 accounts
	for i := 0; i < 3; i++ {
		_, err := uc.CreateAccount(context.Background(), user1, domain.CreateAccountRequest{
			Type:     domain.AccountTypeSavings,
			Currency: "USD",
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	// user2 gets 1 account
	_, err := uc.CreateAccount(context.Background(), user2, domain.CreateAccountRequest{
		Type:     domain.AccountTypeCurrent,
		Currency: "USD",
	})
	if err != nil {
		t.Fatal(err)
	}

	user1Accounts, _ := uc.ListAccounts(context.Background(), user1)
	user2Accounts, _ := uc.ListAccounts(context.Background(), user2)

	if len(user1Accounts) != 3 {
		t.Errorf("expected 3 accounts for user1, got %d", len(user1Accounts))
	}
	if len(user2Accounts) != 1 {
		t.Errorf("expected 1 account for user2, got %d", len(user2Accounts))
	}
}
