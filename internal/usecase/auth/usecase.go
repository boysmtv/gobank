package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/yourorg/gobank/internal/domain"
	"github.com/yourorg/gobank/internal/messaging"
	"github.com/yourorg/gobank/internal/repository"
	"github.com/yourorg/gobank/internal/repository/postgres"
	"github.com/yourorg/gobank/pkg/jwt"
	"github.com/yourorg/gobank/pkg/password"
	"go.uber.org/zap"
)

type UseCase struct {
	userRepo  repository.UserRepository
	jwtMgr    *jwt.Manager
	argon     *password.Params
	publisher messaging.Publisher
	log       *zap.Logger
}

func New(
	userRepo repository.UserRepository,
	jwtMgr *jwt.Manager,
	argon *password.Params,
	publisher messaging.Publisher,
	log *zap.Logger,
) *UseCase {
	return &UseCase{
		userRepo:  userRepo,
		jwtMgr:    jwtMgr,
		argon:     argon,
		publisher: publisher,
		log:       log,
	}
}

func (uc *UseCase) Register(ctx context.Context, req domain.RegisterRequest, ipAddr string) (*domain.User, error) {
	existing, err := uc.userRepo.GetByEmail(ctx, req.Email)
	if err != nil && !errors.Is(err, postgres.ErrNotFound) {
		return nil, fmt.Errorf("checking existing user: %w", err)
	}
	if existing != nil {
		return nil, ErrEmailTaken
	}

	hash, err := password.Hash(req.Password, uc.argon)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	now := time.Now().UTC()
	user := &domain.User{
		ID:           uuid.New(),
		Email:        req.Email,
		PasswordHash: hash,
		FullName:     req.FullName,
		Phone:        req.Phone,
		Role:         domain.RoleCustomer,
		Status:       domain.UserStatusPending,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := uc.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("creating user: %w", err)
	}

	// Async audit event — fire and forget (NATS JetStream guarantees delivery)
	_ = uc.publisher.Publish(ctx, messaging.SubjectAudit, domain.AuditEvent{
		AuditLog: domain.AuditLog{
			ID:         uuid.New(),
			UserID:     &user.ID,
			Action:     domain.AuditActionUserRegistered,
			ResourceID: user.ID.String(),
			IPAddress:  ipAddr,
			Metadata:   map[string]string{"email": user.Email},
			CreatedAt:  now,
		},
	})

	uc.log.Info("user registered", zap.String("user_id", user.ID.String()))
	return user, nil
}

func (uc *UseCase) Login(ctx context.Context, req domain.LoginRequest, ipAddr, userAgent string) (*domain.TokenPair, error) {
	user, err := uc.userRepo.GetByEmail(ctx, req.Email)
	if errors.Is(err, postgres.ErrNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}

	if user.Status == domain.UserStatusSuspended {
		return nil, ErrAccountSuspended
	}

	match, err := password.Verify(req.Password, user.PasswordHash)
	if err != nil || !match {
		return nil, ErrInvalidCredentials
	}

	accessToken, expiry, err := uc.jwtMgr.GenerateAccessToken(user.ID, string(user.Role))
	if err != nil {
		return nil, fmt.Errorf("generating access token: %w", err)
	}

	refreshToken, tokenID, refreshExpiry, err := uc.jwtMgr.GenerateRefreshToken(user.ID, string(user.Role))
	if err != nil {
		return nil, fmt.Errorf("generating refresh token: %w", err)
	}

	// Hash refresh token before storage — never store raw tokens
	refreshHash, err := password.Hash(refreshToken, uc.argon)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	if err := uc.userRepo.SaveRefreshToken(ctx, &domain.RefreshToken{
		ID:        tokenID,
		UserID:    user.ID,
		TokenHash: refreshHash,
		ExpiresAt: refreshExpiry,
		Revoked:   false,
		CreatedAt: now,
	}); err != nil {
		return nil, fmt.Errorf("saving refresh token: %w", err)
	}

	_ = uc.publisher.Publish(ctx, messaging.SubjectAudit, domain.AuditEvent{
		AuditLog: domain.AuditLog{
			ID:         uuid.New(),
			UserID:     &user.ID,
			Action:     domain.AuditActionUserLogin,
			ResourceID: user.ID.String(),
			IPAddress:  ipAddr,
			UserAgent:  userAgent,
			CreatedAt:  now,
		},
	})

	return &domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiry,
	}, nil
}

func (uc *UseCase) RefreshTokens(ctx context.Context, refreshToken string) (*domain.TokenPair, error) {
	claims, err := uc.jwtMgr.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, ErrInvalidToken
	}

	tokenID, err := uuid.Parse(claims.TokenID)
	if err != nil {
		return nil, ErrInvalidToken
	}

	stored, err := uc.userRepo.GetRefreshToken(ctx, tokenID)
	if errors.Is(err, postgres.ErrNotFound) || stored.Revoked {
		return nil, ErrInvalidToken
	}
	if err != nil {
		return nil, err
	}

	if time.Now().After(stored.ExpiresAt) {
		return nil, ErrTokenExpired
	}

	// Token rotation: revoke current, issue new pair
	if err := uc.userRepo.RevokeRefreshToken(ctx, tokenID); err != nil {
		return nil, err
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, ErrInvalidToken
	}

	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	accessToken, expiry, err := uc.jwtMgr.GenerateAccessToken(user.ID, string(user.Role))
	if err != nil {
		return nil, err
	}

	newRefreshToken, newTokenID, newRefreshExpiry, err := uc.jwtMgr.GenerateRefreshToken(user.ID, string(user.Role))
	if err != nil {
		return nil, err
	}

	newRefreshHash, err := password.Hash(newRefreshToken, uc.argon)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	if err := uc.userRepo.SaveRefreshToken(ctx, &domain.RefreshToken{
		ID:        newTokenID,
		UserID:    user.ID,
		TokenHash: newRefreshHash,
		ExpiresAt: newRefreshExpiry,
		Revoked:   false,
		CreatedAt: now,
	}); err != nil {
		return nil, err
	}

	return &domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresAt:    expiry,
	}, nil
}

func (uc *UseCase) Logout(ctx context.Context, userID uuid.UUID, ipAddr string) error {
	if err := uc.userRepo.RevokeAllUserRefreshTokens(ctx, userID); err != nil {
		return err
	}

	now := time.Now().UTC()
	_ = uc.publisher.Publish(ctx, messaging.SubjectAudit, domain.AuditEvent{
		AuditLog: domain.AuditLog{
			ID:         uuid.New(),
			UserID:     &userID,
			Action:     domain.AuditActionUserLogout,
			ResourceID: userID.String(),
			IPAddress:  ipAddr,
			CreatedAt:  now,
		},
	})

	return nil
}

var (
	ErrEmailTaken         = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAccountSuspended   = errors.New("account suspended")
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenExpired       = errors.New("token expired")
)
