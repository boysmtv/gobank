package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
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
	authUC "github.com/yourorg/gobank/internal/usecase/auth"
	"github.com/yourorg/gobank/pkg/jwt"
	"github.com/yourorg/gobank/pkg/password"
	"go.uber.org/zap"
)

// Mock objects
type mockUserRepo struct{ mock.Mock }

func (m *mockUserRepo) Create(ctx context.Context, user *domain.User) error {
	return m.Called(ctx, user).Error(0)
}
func (m *mockUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}
func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}
func (m *mockUserRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.UserStatus) error {
	return m.Called(ctx, id, status).Error(0)
}
func (m *mockUserRepo) SaveRefreshToken(ctx context.Context, token *domain.RefreshToken) error {
	return m.Called(ctx, token).Error(0)
}
func (m *mockUserRepo) GetRefreshToken(ctx context.Context, id uuid.UUID) (*domain.RefreshToken, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.RefreshToken), args.Error(1)
}
func (m *mockUserRepo) RevokeRefreshToken(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *mockUserRepo) RevokeAllUserRefreshTokens(ctx context.Context, userID uuid.UUID) error {
	return m.Called(ctx, userID).Error(0)
}

type mockPublisher struct{ mock.Mock }

func (m *mockPublisher) Publish(ctx context.Context, subject string, payload interface{}) error {
	return m.Called(ctx, subject, payload).Error(0)
}

func TestAuth_Register(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	jwtMgr := jwt.NewManager("access-secret-32-chars-long-enough", "refresh-secret-32-chars-long-enough", time.Hour, time.Hour*24)

	userRepo := new(mockUserRepo)
	publisher := new(mockPublisher)

	authUseCase := authUC.New(userRepo, jwtMgr, password.DefaultParams, publisher, logger)
	authHandler := handler.NewAuthHandler(authUseCase)

	reqBody := domain.RegisterRequest{
		Email:    "test@example.com",
		Password: "password123",
		FullName: "Test User",
		Phone:    "+1234567890",
	}

	userRepo.On("GetByEmail", mock.Anything, reqBody.Email).Return(nil, postgres.ErrNotFound)
	userRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	publisher.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	server := httpDelivery.NewServer(8080, jwtMgr, authHandler, &handler.AccountHandler{}, &handler.TransactionHandler{}, &handler.KYCHandler{}, logger)
	server.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp struct {
		Success bool        `json:"success"`
		Data    domain.User `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, reqBody.Email, resp.Data.Email)
}

func TestAuth_Login(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	jwtMgr := jwt.NewManager("access-secret-32-chars-long-enough", "refresh-secret-32-chars-long-enough", time.Hour, time.Hour*24)

	userRepo := new(mockUserRepo)
	publisher := new(mockPublisher)

	authUseCase := authUC.New(userRepo, jwtMgr, password.DefaultParams, publisher, logger)
	authHandler := handler.NewAuthHandler(authUseCase)

	server := httpDelivery.NewServer(8080, jwtMgr, authHandler, &handler.AccountHandler{}, &handler.TransactionHandler{}, &handler.KYCHandler{}, logger)

	userID := uuid.New()
	hashedPassword, _ := password.Hash("password123", password.DefaultParams)
	user := &domain.User{
		ID:           userID,
		Email:        "test@example.com",
		PasswordHash: hashedPassword,
		Status:       domain.UserStatusActive,
	}

	loginReq := domain.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}

	userRepo.On("GetByEmail", mock.Anything, loginReq.Email).Return(user, nil)
	userRepo.On("SaveRefreshToken", mock.Anything, mock.Anything).Return(nil)
	publisher.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	body, _ := json.Marshal(loginReq)
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	server.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Success bool             `json:"success"`
		Data    domain.TokenPair `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.True(t, resp.Success)
	assert.NotEmpty(t, resp.Data.AccessToken)
	assert.NotEmpty(t, resp.Data.RefreshToken)
}
