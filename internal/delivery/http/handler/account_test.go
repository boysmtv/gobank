package handler_test

import (
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
	accountUC "github.com/yourorg/gobank/internal/usecase/account"
	"github.com/yourorg/gobank/pkg/jwt"
	"go.uber.org/zap"
)

type mockAccountRepo struct{ mock.Mock }

func (m *mockAccountRepo) Create(ctx context.Context, acct *domain.Account) error {
	return m.Called(ctx, acct).Error(0)
}
func (m *mockAccountRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Account, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Account), args.Error(1)
}
func (m *mockAccountRepo) GetByAccountNumber(ctx context.Context, num string) (*domain.Account, error) {
	args := m.Called(ctx, num)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Account), args.Error(1)
}
func (m *mockAccountRepo) ListByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Account, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Account), args.Error(1)
}
func (m *mockAccountRepo) UpdateBalance(ctx context.Context, tx repository.DBTX, id uuid.UUID, delta int64) error {
	return m.Called(ctx, tx, id, delta).Error(0)
}
func (m *mockAccountRepo) LockForUpdate(ctx context.Context, tx repository.DBTX, id uuid.UUID) (*domain.Account, error) {
	args := m.Called(ctx, tx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Account), args.Error(1)
}

func TestAccount_List(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	accessSecret := "access-secret-32-chars-long-enough"
	jwtMgr := jwt.NewManager(accessSecret, "refresh-secret-32-chars-long-enough", time.Hour, time.Hour*24)

	accountRepo := new(mockAccountRepo)
	publisher := new(mockPublisher)

	accountUseCase := accountUC.New(accountRepo, publisher, logger)
	accountHandler := handler.NewAccountHandler(accountUseCase)

	server := httpDelivery.NewServer(8080, jwtMgr, &handler.AuthHandler{}, accountHandler, &handler.TransactionHandler{}, &handler.KYCHandler{}, logger)

	userID := uuid.New()
	token, _, _ := jwtMgr.GenerateAccessToken(userID, "user")

	accounts := []domain.Account{
		{ID: uuid.New(), UserID: userID, Type: domain.AccountTypeSavings, Balance: 1000, Currency: "USD"},
	}

	accountRepo.On("ListByUserID", mock.Anything, userID).Return(accounts, nil)

	req := httptest.NewRequest("GET", "/api/v1/accounts", nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	w := httptest.NewRecorder()

	server.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Success bool             `json:"success"`
		Data    []domain.Account `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Len(t, resp.Data, 1)
	assert.Equal(t, accounts[0].ID, resp.Data[0].ID)
}
