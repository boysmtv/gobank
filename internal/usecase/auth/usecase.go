package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gobank/internal/domain"
	"gobank/internal/repository"
)

var (
	ErrEmailAlreadyRegistered = errors.New("email already registered")
	ErrInvalidCredentials     = errors.New("invalid credentials")
)

type PasswordHasher interface {
	Hash(string) (string, error)
	Compare(string, string) error
}

type TokenManager interface {
	Generate(string) (string, error)
}

type UseCase struct {
	users  repository.UserRepository
	audits repository.AuditRepository
	hasher PasswordHasher
	tokens TokenManager
}

type RegisterInput struct {
	Email    string
	FullName string
	Password string
}

type LoginInput struct {
	Email    string
	Password string
}

func NewUseCase(
	users repository.UserRepository,
	audits repository.AuditRepository,
	hasher PasswordHasher,
	tokens TokenManager,
) *UseCase {
	return &UseCase{
		users:  users,
		audits: audits,
		hasher: hasher,
		tokens: tokens,
	}
}

func (u *UseCase) Register(ctx context.Context, input RegisterInput) (domain.User, error) {
	if existing, err := u.users.FindByEmail(ctx, input.Email); err == nil && existing.ID != "" {
		return domain.User{}, ErrEmailAlreadyRegistered
	}

	hash, err := u.hasher.Hash(input.Password)
	if err != nil {
		return domain.User{}, err
	}

	now := time.Now().UTC()
	user := domain.User{
		ID:           fmt.Sprintf("usr_%d", now.UnixNano()),
		Email:        input.Email,
		FullName:     input.FullName,
		PasswordHash: hash,
		Status:       domain.UserStatusActive,
		KYCStatus:    domain.KYCStatusNotSubmitted,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	created, err := u.users.Create(ctx, user)
	if err != nil {
		return domain.User{}, err
	}

	_ = u.audits.Create(ctx, domain.AuditLog{
		ID:         fmt.Sprintf("aud_%d", now.UnixNano()),
		ActorID:    created.ID,
		Action:     "user.registered",
		Resource:   "user",
		ResourceID: created.ID,
		CreatedAt:  now,
	})

	return created, nil
}

func (u *UseCase) Login(ctx context.Context, input LoginInput) (string, error) {
	user, err := u.users.FindByEmail(ctx, input.Email)
	if err != nil {
		return "", ErrInvalidCredentials
	}

	if err := u.hasher.Compare(input.Password, user.PasswordHash); err != nil {
		return "", ErrInvalidCredentials
	}

	now := time.Now().UTC()
	user.LastLoginAt = &now
	user.UpdatedAt = now
	_, _ = u.users.Update(ctx, user)

	_ = u.audits.Create(ctx, domain.AuditLog{
		ID:         fmt.Sprintf("aud_%d", now.UnixNano()),
		ActorID:    user.ID,
		Action:     "user.logged_in",
		Resource:   "user",
		ResourceID: user.ID,
		CreatedAt:  now,
	})

	return u.tokens.Generate(user.ID)
}
