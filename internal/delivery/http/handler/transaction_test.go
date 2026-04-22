package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	httpDelivery "github.com/yourorg/gobank/internal/delivery/http"
	"github.com/yourorg/gobank/internal/delivery/http/handler"
	"github.com/yourorg/gobank/internal/domain"
	"github.com/yourorg/gobank/internal/repository"
	"github.com/yourorg/gobank/internal/repository/postgres"
	txnUC "github.com/yourorg/gobank/internal/usecase/transaction"
	"github.com/yourorg/gobank/pkg/jwt"
	"go.uber.org/zap"
)

type mockTransactionRepo struct{ mock.Mock }

func (m *mockTransactionRepo) Create(ctx context.Context, tx repository.DBTX, txn *domain.Transaction) error {
	return m.Called(ctx, tx, txn).Error(0)
}
func (m *mockTransactionRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Transaction, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Transaction), args.Error(1)
}
func (m *mockTransactionRepo) GetByIdempotencyKey(ctx context.Context, key string) (*domain.Transaction, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Transaction), args.Error(1)
}
func (m *mockTransactionRepo) ListByAccountID(ctx context.Context, accountID uuid.UUID, page, pageSize int) ([]domain.Transaction, int, error) {
	args := m.Called(ctx, accountID, page, pageSize)
	return args.Get(0).([]domain.Transaction), args.Int(1), args.Error(2)
}
func (m *mockTransactionRepo) UpdateStatus(ctx context.Context, tx repository.DBTX, id uuid.UUID, status domain.TransactionStatus) error {
	return m.Called(ctx, tx, id, status).Error(0)
}

type mockIdempotencyStore struct{ mock.Mock }

func (m *mockIdempotencyStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error) {
	args := m.Called(ctx, key, value, ttl)
	return args.Bool(0), args.Error(1)
}
func (m *mockIdempotencyStore) Get(ctx context.Context, key string) ([]byte, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}
func (m *mockIdempotencyStore) Delete(ctx context.Context, key string) error {
	return m.Called(ctx, key).Error(0)
}

type mockTxManager struct{ mock.Mock }

func (m *mockTxManager) WithTransaction(ctx context.Context, fn func(ctx context.Context, tx repository.DBTX) error) error {
	args := m.Called(ctx, fn)
	return args.Error(0)
}

func TestTransaction_Transfer(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	jwtMgr := jwt.NewManager("access-secret-32-chars-long-enough", "refresh-secret-32-chars-long-enough", time.Hour, time.Hour*24)

	txnRepo := new(mockTransactionRepo)
	accountRepo := new(mockAccountRepo)
	idemStore := new(mockIdempotencyStore)
	txManager := new(mockTxManager)
	publisher := new(mockPublisher)

	// Mock txManager.WithTransaction to just call the function
	txManager.On("WithTransaction", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		fn := args.Get(1).(func(context.Context, repository.DBTX) error)
		_ = fn(context.Background(), nil)
	}).Return(nil)

	txnUseCase := txnUC.New(txnRepo, accountRepo, idemStore, txManager, publisher, logger)
	txHandler := handler.NewTransactionHandler(txnUseCase)

	server := httpDelivery.NewServer(8080, jwtMgr, &handler.AuthHandler{}, &handler.AccountHandler{}, txHandler, &handler.KYCHandler{}, logger)

	userID := uuid.New()
	token, _, _ := jwtMgr.GenerateAccessToken(userID, "user")

	fromAcctID := uuid.New()
	toAcctID := uuid.New()

	reqBody := domain.TransferRequest{
		FromAccountID:  fromAcctID.String(),
		ToAccountID:    toAcctID.String(),
		Amount:         500,
		Currency:       "USD",
		Description:    "Test transfer",
		IdempotencyKey: uuid.New().String(),
	}

	fromAcct := &domain.Account{ID: fromAcctID, UserID: userID, Balance: 1000, Currency: "USD", Status: domain.AccountStatusActive}
	toAcct := &domain.Account{ID: toAcctID, Balance: 0, Currency: "USD", Status: domain.AccountStatusActive}

	idemStore.On("Get", mock.Anything, mock.MatchedBy(func(k string) bool {
		return k == "transfer:"+reqBody.IdempotencyKey
	})).Return(nil, nil)
	idemStore.On("Set", mock.Anything, "transfer:"+reqBody.IdempotencyKey, mock.Anything, mock.Anything).Return(true, nil)
	txnRepo.On("GetByIdempotencyKey", mock.Anything, reqBody.IdempotencyKey).Return(nil, postgres.ErrNotFound)
	accountRepo.On("GetByID", mock.Anything, fromAcctID).Return(fromAcct, nil)
	accountRepo.On("GetByID", mock.Anything, toAcctID).Return(toAcct, nil)

	accountRepo.On("LockForUpdate", mock.Anything, mock.Anything, fromAcctID).Return(fromAcct, nil)
	accountRepo.On("LockForUpdate", mock.Anything, mock.Anything, toAcctID).Return(toAcct, nil)

	accountRepo.On("UpdateBalance", mock.Anything, mock.Anything, fromAcctID, int64(-500)).Return(nil)
	accountRepo.On("UpdateBalance", mock.Anything, mock.Anything, toAcctID, int64(500)).Return(nil)

	txnRepo.On("Create", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	txnRepo.On("UpdateStatus", mock.Anything, mock.Anything, mock.Anything, domain.TransactionStatusSuccess).Return(nil)
	publisher.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/transactions/transfer", bytes.NewBuffer(body))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	w := httptest.NewRecorder()

	server.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}
