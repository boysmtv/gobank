package kyc

import (
	"context"
	"fmt"
	"time"

	"gobank/internal/domain"
	"gobank/internal/repository"
)

type UseCase struct {
	kyc    repository.KYCRepository
	users  repository.UserRepository
	audits repository.AuditRepository
}

type SubmitInput struct {
	UserID         string
	DocumentType   string
	DocumentNumber string
}

func NewUseCase(
	kyc repository.KYCRepository,
	users repository.UserRepository,
	audits repository.AuditRepository,
) *UseCase {
	return &UseCase{
		kyc:    kyc,
		users:  users,
		audits: audits,
	}
}

func (u *UseCase) Submit(ctx context.Context, input SubmitInput) (domain.KYCSubmission, error) {
	now := time.Now().UTC()
	submission := domain.KYCSubmission{
		ID:             fmt.Sprintf("kyc_%d", now.UnixNano()),
		UserID:         input.UserID,
		DocumentType:   input.DocumentType,
		DocumentNumber: input.DocumentNumber,
		Status:         domain.KYCStatusReview,
		SubmittedAt:    now,
	}

	created, err := u.kyc.Create(ctx, submission)
	if err != nil {
		return domain.KYCSubmission{}, err
	}

	user, err := u.users.FindByID(ctx, input.UserID)
	if err == nil {
		user.KYCStatus = domain.KYCStatusReview
		user.UpdatedAt = now
		_, _ = u.users.Update(ctx, user)
	}

	_ = u.audits.Create(ctx, domain.AuditLog{
		ID:         fmt.Sprintf("aud_%d", now.UnixNano()),
		ActorID:    input.UserID,
		Action:     "kyc.submitted",
		Resource:   "kyc",
		ResourceID: created.ID,
		CreatedAt:  now,
	})

	return created, nil
}

func (u *UseCase) Approve(ctx context.Context, userID, notes string) (domain.KYCSubmission, error) {
	submission, err := u.kyc.FindByUserID(ctx, userID)
	if err != nil {
		return domain.KYCSubmission{}, err
	}

	now := time.Now().UTC()
	submission.Status = domain.KYCStatusApproved
	submission.Notes = notes
	submission.ReviewedAt = &now

	updated, err := u.kyc.Update(ctx, submission)
	if err != nil {
		return domain.KYCSubmission{}, err
	}

	user, err := u.users.FindByID(ctx, userID)
	if err == nil {
		user.KYCStatus = domain.KYCStatusApproved
		user.UpdatedAt = now
		_, _ = u.users.Update(ctx, user)
	}

	_ = u.audits.Create(ctx, domain.AuditLog{
		ID:         fmt.Sprintf("aud_%d", now.UnixNano()),
		ActorID:    userID,
		Action:     "kyc.approved",
		Resource:   "kyc",
		ResourceID: updated.ID,
		CreatedAt:  now,
	})

	return updated, nil
}
