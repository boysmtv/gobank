package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourorg/gobank/internal/domain"
)

type UserRepo struct {
	db *pgxpool.Pool
}

func NewUserRepo(db *pgxpool.Pool) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(ctx context.Context, user *domain.User) error {
	const q = `
        INSERT INTO users (id, email, password_hash, full_name, phone, role, status, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := r.db.Exec(ctx, q,
		user.ID, user.Email, user.PasswordHash, user.FullName,
		user.Phone, user.Role, user.Status, user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("userRepo.Create: %w", err)
	}
	return nil
}

func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	const q = `
        SELECT id, email, password_hash, full_name, phone, role, status, created_at, updated_at
        FROM users WHERE id = $1 AND status != 'suspended'`
	u := &domain.User{}
	err := r.db.QueryRow(ctx, q, id).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.FullName,
		&u.Phone, &u.Role, &u.Status, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("userRepo.GetByID: %w", err)
	}
	return u, nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	const q = `
        SELECT id, email, password_hash, full_name, phone, role, status, created_at, updated_at
        FROM users WHERE email = $1`
	u := &domain.User{}
	err := r.db.QueryRow(ctx, q, email).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.FullName,
		&u.Phone, &u.Role, &u.Status, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("userRepo.GetByEmail: %w", err)
	}
	return u, nil
}

func (r *UserRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.UserStatus) error {
	const q = `UPDATE users SET status = $1, updated_at = $2 WHERE id = $3`
	_, err := r.db.Exec(ctx, q, status, time.Now().UTC(), id)
	return err
}

func (r *UserRepo) SaveRefreshToken(ctx context.Context, token *domain.RefreshToken) error {
	const q = `
        INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, revoked, created_at)
        VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.db.Exec(ctx, q,
		token.ID, token.UserID, token.TokenHash,
		token.ExpiresAt, token.Revoked, token.CreatedAt,
	)
	return err
}

func (r *UserRepo) GetRefreshToken(ctx context.Context, id uuid.UUID) (*domain.RefreshToken, error) {
	const q = `
        SELECT id, user_id, token_hash, expires_at, revoked, created_at
        FROM refresh_tokens WHERE id = $1`
	t := &domain.RefreshToken{}
	err := r.db.QueryRow(ctx, q, id).Scan(
		&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.Revoked, &t.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return t, err
}

func (r *UserRepo) RevokeRefreshToken(ctx context.Context, id uuid.UUID) error {
	const q = `UPDATE refresh_tokens SET revoked = true WHERE id = $1`
	_, err := r.db.Exec(ctx, q, id)
	return err
}

func (r *UserRepo) RevokeAllUserRefreshTokens(ctx context.Context, userID uuid.UUID) error {
	const q = `UPDATE refresh_tokens SET revoked = true WHERE user_id = $1`
	_, err := r.db.Exec(ctx, q, userID)
	return err
}
