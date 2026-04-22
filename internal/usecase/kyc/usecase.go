package kyc

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
	"go.uber.org/zap"
)

type UseCase struct {
	kycRepo   repository.KYCRepository
	userRepo  repository.UserRepository
	publisher messaging.Publisher
	log       *zap.Logger
}

func New(kycRepo repository.KYCRepository, userRepo repository.UserRepository, publisher messaging.Publisher, log *zap.Logger) *UseCase {
	return &UseCase{kycRepo: kycRepo, userRepo: userRepo, publisher: publisher, log: log}
}

func (uc *UseCase) Submit(ctx context.Context, userID uuid.UUID, req domain.KYCSubmitRequest) (*domain.KYCRecord, error) {
	existing, err := uc.kycRepo.GetByUserID(ctx, userID)
	if err != nil && !errors.Is(err, postgres.ErrNotFound) {
		return nil, err
	}
	if existing != nil && existing.Status == domain.KYCStatusVerified {
		return nil, ErrAlreadyVerified
	}

	dob, err := time.Parse("2006-01-02", req.DateOfBirth)
	if err != nil {
		return nil, fmt.Errorf("invalid date of birth: %w", err)
	}

	now := time.Now().UTC()
	record := &domain.KYCRecord{
		ID:            uuid.New(),
		UserID:        userID,
		FullLegalName: req.FullLegalName,
		DateOfBirth:   dob,
		NationalID:    req.NationalID,
		Address:       req.Address,
		Status:        domain.KYCStatusPending,
		SubmittedAt:   now,
	}

	if existing != nil {
		// Re-submission: update status back to pending
		if err := uc.kycRepo.UpdateStatus(ctx, userID, domain.KYCStatusPending, ""); err != nil {
			return nil, err
		}
	} else {
		if err := uc.kycRepo.Create(ctx, record); err != nil {
			return nil, err
		}
	}

	// Activate user on KYC submission
	_ = uc.userRepo.UpdateStatus(ctx, userID, domain.UserStatusActive)

	_ = uc.publisher.Publish(ctx, messaging.SubjectAudit, domain.AuditEvent{
		AuditLog: domain.AuditLog{
			ID:         uuid.New(),
			UserID:     &userID,
			Action:     domain.AuditActionKYCSubmitted,
			ResourceID: record.ID.String(),
			CreatedAt:  now,
		},
	})

	return record, nil
}

func (uc *UseCase) GetStatus(ctx context.Context, userID uuid.UUID) (*domain.KYCRecord, error) {
	return uc.kycRepo.GetByUserID(ctx, userID)
}

// Verify is an admin-only action to mark KYC as verified.
func (uc *UseCase) Verify(ctx context.Context, adminID, targetUserID uuid.UUID) error {
	if err := uc.kycRepo.UpdateStatus(ctx, targetUserID, domain.KYCStatusVerified, ""); err != nil {
		return err
	}
	now := time.Now().UTC()
	_ = uc.publisher.Publish(ctx, messaging.SubjectAudit, domain.AuditEvent{
		AuditLog: domain.AuditLog{
			ID:         uuid.New(),
			UserID:     &adminID,
			Action:     domain.AuditActionKYCVerified,
			ResourceID: targetUserID.String(),
			CreatedAt:  now,
		},
	})
	return nil
}

var ErrAlreadyVerified = errors.New("KYC already verified")
