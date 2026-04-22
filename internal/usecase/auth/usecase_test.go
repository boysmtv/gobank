package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/yourorg/gobank/internal/domain"
	"github.com/yourorg/gobank/internal/repository/postgres"
	authUC "github.com/yourorg/gobank/internal/usecase/auth"
	"github.com/yourorg/gobank/pkg/jwt"
	"github.com/yourorg/gobank/pkg/password"
	"go.uber.org/zap"
)

// --- Mock implementations ---

type mockUserRepo struct {
	users  map[string]*domain.User
	tokens map[uuid.UUID]*domain.RefreshToken
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		users:  make(map[string]*domain.User),
		tokens: make(map[uuid.UUID]*domain.RefreshToken),
	}
}

func (m *mockUserRepo) Create(ctx context.Context, user *domain.User) error {
	m.users[user.Email] = user
	return nil
}

func (m *mockUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, postgres.ErrNotFound
}

func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	u, ok := m.users[email]
	if !ok {
		return nil, postgres.ErrNotFound
	}
	return u, nil
}

func (m *mockUserRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.UserStatus) error {
	return nil
}

func (m *mockUserRepo) SaveRefreshToken(ctx context.Context, token *domain.RefreshToken) error {
	m.tokens[token.ID] = token
	return nil
}

func (m *mockUserRepo) GetRefreshToken(ctx context.Context, id uuid.UUID) (*domain.RefreshToken, error) {
	t, ok := m.tokens[id]
	if !ok {
		return nil, postgres.ErrNotFound
	}
	return t, nil
}

func (m *mockUserRepo) RevokeRefreshToken(ctx context.Context, id uuid.UUID) error {
	if t, ok := m.tokens[id]; ok {
		t.Revoked = true
	}
	return nil
}

func (m *mockUserRepo) RevokeAllUserRefreshTokens(ctx context.Context, userID uuid.UUID) error {
	for _, t := range m.tokens {
		if t.UserID == userID {
			t.Revoked = true
		}
	}
	return nil
}

type mockPublisher struct{}

func (m *mockPublisher) Publish(ctx context.Context, subject string, payload interface{}) error {
	return nil
}

// --- Tests ---

func newTestUseCase(repo *mockUserRepo) *authUC.UseCase {
	jwtMgr := jwt.NewManager("test-access-secret-32-characters!!", "test-refresh-secret-32-characters!", 15*time.Minute, 7*24*time.Hour)
	argon := password.DefaultParams
	log, _ := zap.NewDevelopment()
	return authUC.New(repo, jwtMgr, argon, &mockPublisher{}, log)
}

func TestRegister_Success(t *testing.T) {
	repo := newMockUserRepo()
	uc := newTestUseCase(repo)

	req := domain.RegisterRequest{
		Email:    "alice@example.com",
		Password: "securepassword123",
		FullName: "Alice Smith",
		Phone:    "+12025550100",
	}

	user, err := uc.Register(context.Background(), req, "127.0.0.1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if user.Email != req.Email {
		t.Errorf("expected email %s, got %s", req.Email, user.Email)
	}
	if user.PasswordHash == req.Password {
		t.Error("password should be hashed")
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	repo := newMockUserRepo()
	uc := newTestUseCase(repo)

	req := domain.RegisterRequest{
		Email:    "bob@example.com",
		Password: "password123",
		FullName: "Bob",
		Phone:    "+12025550101",
	}

	_, err := uc.Register(context.Background(), req, "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}

	_, err = uc.Register(context.Background(), req, "127.0.0.1")
	if err != authUC.ErrEmailTaken {
		t.Errorf("expected ErrEmailTaken, got: %v", err)
	}
}

func TestLogin_InvalidCredentials(t *testing.T) {
	repo := newMockUserRepo()
	uc := newTestUseCase(repo)

	req := domain.LoginRequest{
		Email:    "nobody@example.com",
		Password: "wrong",
	}

	_, err := uc.Login(context.Background(), req, "127.0.0.1", "test-agent")
	if err != authUC.ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got: %v", err)
	}
}

func TestLogin_Success_And_Refresh(t *testing.T) {
	repo := newMockUserRepo()
	uc := newTestUseCase(repo)

	regReq := domain.RegisterRequest{
		Email:    "carol@example.com",
		Password: "mypassword123",
		FullName: "Carol",
		Phone:    "+12025550102",
	}
	_, err := uc.Register(context.Background(), regReq, "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}

	loginReq := domain.LoginRequest{Email: regReq.Email, Password: regReq.Password}
	tokens, err := uc.Login(context.Background(), loginReq, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Fatal("expected non-empty tokens")
	}

	newTokens, err := uc.RefreshTokens(context.Background(), tokens.RefreshToken)
	if err != nil {
		t.Fatalf("refresh failed: %v", err)
	}
	if newTokens.AccessToken == tokens.AccessToken {
		t.Error("new access token should differ from old")
	}
}
