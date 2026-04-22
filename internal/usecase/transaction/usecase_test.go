package transaction_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/yourorg/gobank/internal/domain"
	"github.com/yourorg/gobank/internal/repository"
	"github.com/yourorg/gobank/internal/repository/postgres"
	txUC "github.com/yourorg/gobank/internal/usecase/transaction"
	"go.uber.org/zap"
)

// --- Mock TxManager ---

type mockTxManager struct{}

func (m *mockTxManager) WithTransaction(ctx context.Context, fn func(context.Context, repository.DBTX) error) error {
	return fn(ctx, &mockDBTX{})
}

type mockDBTX struct{}

func (m *mockDBTX) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (m *mockDBTX) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return &mockRow{}
}

type mockRow struct{}

func (m *mockRow) Scan(dest ...any) error {
	return nil
}

// --- Mock Account Repo ---

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
func (m *mockAccountRepo) GetByAccountNumber(ctx context.Context, n string) (*domain.Account, error) {
	return nil, postgres.ErrNotFound
}
func (m *mockAccountRepo) ListByUserID(ctx context.Context, uid uuid.UUID) ([]domain.Account, error) {
	return nil, nil
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

// --- Mock Transaction Repo ---

type mockTxnRepo struct {
	txns map[uuid.UUID]*domain.Transaction
}

func newMockTxnRepo() *mockTxnRepo {
	return &mockTxnRepo{txns: make(map[uuid.UUID]*domain.Transaction)}
}

func (m *mockTxnRepo) Create(ctx context.Context, tx repository.DBTX, t *domain.Transaction) error {
	m.txns[t.ID] = t
	return nil
}
func (m *mockTxnRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Transaction, error) {
	t, ok := m.txns[id]
	if !ok {
		return nil, postgres.ErrNotFound
	}
	return t, nil
}
func (m *mockTxnRepo) GetByIdempotencyKey(ctx context.Context, key string) (*domain.Transaction, error) {
	for _, t := range m.txns {
		if t.IdempotencyKey == key {
			return t, nil
		}
	}
	return nil, postgres.ErrNotFound
}
func (m *mockTxnRepo) ListByAccountID(ctx context.Context, accountID uuid.UUID, page, pageSize int) ([]domain.Transaction, int, error) {
	return nil, 0, nil
}
func (m *mockTxnRepo) UpdateStatus(ctx context.Context, tx repository.DBTX, id uuid.UUID, status domain.TransactionStatus) error {
	if t, ok := m.txns[id]; ok {
		t.Status = status
	}
	return nil
}

// --- Mock Idempotency Store ---

type mockIdemStore struct {
	data map[string][]byte
}

func newMockIdemStore() *mockIdemStore {
	return &mockIdemStore{data: make(map[string][]byte)}
}
func (m *mockIdemStore) Set(ctx context.Context, key string, val []byte, _ time.Duration) (bool, error) {
	if _, exists := m.data[key]; exists {
		return false, nil
	}
	m.data[key] = val
	return true, nil
}
func (m *mockIdemStore) Get(ctx context.Context, key string) ([]byte, error) {
	return m.data[key], nil
}
func (m *mockIdemStore) Delete(ctx context.Context, key string) error {
	delete(m.data, key)
	return nil
}

type mockPublisher struct{}

func (m *mockPublisher) Publish(_ context.Context, _ string, _ interface{}) error { return nil }

func newTestTxUseCase(acctRepo *mockAccountRepo, txnRepo *mockTxnRepo, idem *mockIdemStore) *txUC.UseCase {
	log, _ := zap.NewDevelopment()
	return txUC.New(txnRepo, acctRepo, idem, &mockTxManager{}, &mockPublisher{}, log)
}

func TestTransfer_Success(t *testing.T) {
	acctRepo := newMockAccountRepo()
	txnRepo := newMockTxnRepo()
	idem := newMockIdemStore()

	userID := uuid.New()
	fromID := uuid.New()
	toID := uuid.New()

	acctRepo.accounts[fromID] = &domain.Account{
		ID: fromID, UserID: userID, Currency: "USD",
		Balance: 10000, Status: domain.AccountStatusActive,
	}
	acctRepo.accounts[toID] = &domain.Account{
		ID: toID, UserID: uuid.New(), Currency: "USD",
		Balance: 0, Status: domain.AccountStatusActive,
	}

	uc := newTestTxUseCase(acctRepo, txnRepo, idem)

	txn, err := uc.Transfer(context.Background(), userID, domain.TransferRequest{
		IdempotencyKey: "test-idem-key-001",
		FromAccountID:  fromID.String(),
		ToAccountID:    toID.String(),
		Amount:         500,
		Currency:       "USD",
	}, "127.0.0.1")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if txn.Status != domain.TransactionStatusSuccess {
		t.Errorf("expected success, got: %s", txn.Status)
	}
	if acctRepo.accounts[fromID].Balance != 9500 {
		t.Errorf("expected from balance 9500, got %d", acctRepo.accounts[fromID].Balance)
	}
	if acctRepo.accounts[toID].Balance != 500 {
		t.Errorf("expected to balance 500, got %d", acctRepo.accounts[toID].Balance)
	}
}

func TestTransfer_InsufficientFunds(t *testing.T) {
	acctRepo := newMockAccountRepo()
	txnRepo := newMockTxnRepo()
	idem := newMockIdemStore()

	userID := uuid.New()
	fromID := uuid.New()
	toID := uuid.New()

	acctRepo.accounts[fromID] = &domain.Account{
		ID: fromID, UserID: userID, Currency: "USD",
		Balance: 100, Status: domain.AccountStatusActive,
	}
	acctRepo.accounts[toID] = &domain.Account{
		ID: toID, UserID: uuid.New(), Currency: "USD",
		Balance: 0, Status: domain.AccountStatusActive,
	}

	uc := newTestTxUseCase(acctRepo, txnRepo, idem)

	_, err := uc.Transfer(context.Background(), userID, domain.TransferRequest{
		IdempotencyKey: "test-idem-key-002",
		FromAccountID:  fromID.String(),
		ToAccountID:    toID.String(),
		Amount:         9999,
		Currency:       "USD",
	}, "127.0.0.1")

	if err != txUC.ErrInsufficientFunds {
		t.Errorf("expected ErrInsufficientFunds, got: %v", err)
	}
}

func TestTransfer_Idempotency(t *testing.T) {
	acctRepo := newMockAccountRepo()
	txnRepo := newMockTxnRepo()
	idem := newMockIdemStore()

	userID := uuid.New()
	fromID := uuid.New()
	toID := uuid.New()

	acctRepo.accounts[fromID] = &domain.Account{
		ID: fromID, UserID: userID, Currency: "USD",
		Balance: 10000, Status: domain.AccountStatusActive,
	}
	acctRepo.accounts[toID] = &domain.Account{
		ID: toID, UserID: uuid.New(), Currency: "USD",
		Balance: 0, Status: domain.AccountStatusActive,
	}

	uc := newTestTxUseCase(acctRepo, txnRepo, idem)

	req := domain.TransferRequest{
		IdempotencyKey: "test-idem-key-003",
		FromAccountID:  fromID.String(),
		ToAccountID:    toID.String(),
		Amount:         200,
		Currency:       "USD",
	}

	first, err := uc.Transfer(context.Background(), userID, req, "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}

	// Replay same request
	second, err := uc.Transfer(context.Background(), userID, req, "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}

	if first.ID != second.ID {
		t.Error("idempotent replay should return the same transaction ID")
	}

	// Balance should only have been deducted once
	if acctRepo.accounts[fromID].Balance != 9800 {
		t.Errorf("expected balance 9800, got %d", acctRepo.accounts[fromID].Balance)
	}
}
