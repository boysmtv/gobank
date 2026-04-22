package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourorg/gobank/internal/domain"
)

type KYCRepo struct {
	db *pgxpool.Pool
}

func NewKYCRepo(db *pgxpool.Pool) *KYCRepo {
	return &KYCRepo{db: db}
}

func (r *KYCRepo) Create(ctx context.Context, record *domain.KYCRecord) error {
	const q = `
        INSERT INTO kyc_records (id, user_id, full_legal_name, date_of_birth, national_id, address, status, submitted_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.db.Exec(ctx, q,
		record.ID, record.UserID, record.FullLegalName, record.DateOfBirth,
		record.NationalID, record.Address, record.Status, record.SubmittedAt,
	)
	if err != nil {
		return fmt.Errorf("kycRepo.Create: %w", err)
	}
	return nil
}

func (r *KYCRepo) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.KYCRecord, error) {
	const q = `
        SELECT id, user_id, full_legal_name, date_of_birth, national_id, address, status, rejection_note, submitted_at, verified_at
        FROM kyc_records WHERE user_id = $1`
	k := &domain.KYCRecord{}
	err := r.db.QueryRow(ctx, q, userID).Scan(
		&k.ID, &k.UserID, &k.FullLegalName, &k.DateOfBirth,
		&k.NationalID, &k.Address, &k.Status, &k.RejectionNote,
		&k.SubmittedAt, &k.VerifiedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return k, err
}

func (r *KYCRepo) UpdateStatus(ctx context.Context, userID uuid.UUID, status domain.KYCStatus, note string) error {
	const q = `
        UPDATE kyc_records SET status = $1, rejection_note = $2,
        verified_at = CASE WHEN $1 = 'verified' THEN NOW() ELSE NULL END
        WHERE user_id = $3`
	_, err := r.db.Exec(ctx, q, status, note, userID)
	return err
}
