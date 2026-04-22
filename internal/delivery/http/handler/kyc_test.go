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
	"github.com/yourorg/gobank/internal/repository/postgres"
	kycUC "github.com/yourorg/gobank/internal/usecase/kyc"
	"github.com/yourorg/gobank/pkg/jwt"
	"go.uber.org/zap"
)

type mockKYCRepo struct{ mock.Mock }

func (m *mockKYCRepo) Create(ctx context.Context, record *domain.KYCRecord) error {
	return m.Called(ctx, record).Error(0)
}
func (m *mockKYCRepo) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.KYCRecord, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.KYCRecord), args.Error(1)
}
func (m *mockKYCRepo) UpdateStatus(ctx context.Context, userID uuid.UUID, status domain.KYCStatus, note string) error {
	return m.Called(ctx, userID, status, note).Error(0)
}

func TestKYC_Submit(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	jwtMgr := jwt.NewManager("access-secret-32-chars-long-enough", "refresh-secret-32-chars-long-enough", time.Hour, time.Hour*24)

	kycRepo := new(mockKYCRepo)
	userRepo := new(mockUserRepo)
	publisher := new(mockPublisher)

	kycUseCase := kycUC.New(kycRepo, userRepo, publisher, logger)
	kycHandler := handler.NewKYCHandler(kycUseCase)

	server := httpDelivery.NewServer(8080, jwtMgr, &handler.AuthHandler{}, &handler.AccountHandler{}, &handler.TransactionHandler{}, kycHandler, logger)

	userID := uuid.New()
	token, _, _ := jwtMgr.GenerateAccessToken(userID, "user")

	reqBody := domain.KYCSubmitRequest{
		FullLegalName: "John Doe",
		DateOfBirth:   "1990-01-01",
		NationalID:    "1234567890",
		Address:       "123 Main St, Anytown, USA",
	}

	kycRepo.On("GetByUserID", mock.Anything, userID).Return(nil, postgres.ErrNotFound)
	kycRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	userRepo.On("UpdateStatus", mock.Anything, userID, domain.UserStatusActive).Return(nil)
	publisher.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/kyc", bytes.NewBuffer(body))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	w := httptest.NewRecorder()

	server.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}
