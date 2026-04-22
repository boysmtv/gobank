package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"gobank/internal/domain"
)

type mockUserRepo struct {
	users map[string]domain.User
}

func (m *mockUserRepo) Create(_ context.Context, user domain.User) (domain.User, error) {
	if m.users == nil {
		m.users = make(map[string]domain.User)
	}
	m.users[user.Email] = user
	return user, nil
}

func (m *mockUserRepo) FindByID(_ context.Context, id string) (domain.User, error) {
	for _, user := range m.users {
		if user.ID == id {
			return user, nil
		}
	}
	return domain.User{}, errors.New("not found")
}

func (m *mockUserRepo) FindByEmail(_ context.Context, email string) (domain.User, error) {
	user, ok := m.users[email]
	if !ok {
		return domain.User{}, errors.New("not found")
	}
	return user, nil
}

func (m *mockUserRepo) Update(_ context.Context, user domain.User) (domain.User, error) {
	m.users[user.Email] = user
	return user, nil
}

type mockAuditRepo struct{}

func (m *mockAuditRepo) Create(context.Context, domain.AuditLog) error {
	return nil
}

func (m *mockAuditRepo) List(context.Context, string) ([]domain.AuditLog, error) {
	return nil, nil
}

type mockHasher struct{}

func (m *mockHasher) Hash(raw string) (string, error) {
	return "hash:" + raw, nil
}

func (m *mockHasher) Compare(raw, hashed string) error {
	if hashed != "hash:"+raw {
		return errors.New("mismatch")
	}
	return nil
}

type mockTokenManager struct{}

func (m *mockTokenManager) Generate(subject string) (string, error) {
	return "token:" + subject, nil
}

func TestRegister(t *testing.T) {
	uc := NewUseCase(&mockUserRepo{}, &mockAuditRepo{}, &mockHasher{}, &mockTokenManager{})

	user, err := uc.Register(context.Background(), RegisterInput{
		Email:    "john@example.com",
		FullName: "John Doe",
		Password: "secret",
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if user.Email != "john@example.com" {
		t.Fatalf("expected email to be stored")
	}
}

func TestLogin(t *testing.T) {
	repo := &mockUserRepo{
		users: map[string]domain.User{
			"john@example.com": {
				ID:           "usr_1",
				Email:        "john@example.com",
				PasswordHash: "hash:secret",
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
		},
	}
	uc := NewUseCase(repo, &mockAuditRepo{}, &mockHasher{}, &mockTokenManager{})

	token, err := uc.Login(context.Background(), LoginInput{
		Email:    "john@example.com",
		Password: "secret",
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if token != "token:usr_1" {
		t.Fatalf("unexpected token %q", token)
	}
}
