package postgres

import (
	"context"
	"sync"

	"gobank/internal/domain"
)

type KYCRepository struct {
	mu       sync.RWMutex
	byID     map[string]domain.KYCSubmission
	byUserID map[string]string
}

func NewKYCRepository() *KYCRepository {
	return &KYCRepository{
		byID:     make(map[string]domain.KYCSubmission),
		byUserID: make(map[string]string),
	}
}

func (r *KYCRepository) Create(_ context.Context, submission domain.KYCSubmission) (domain.KYCSubmission, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.byID[submission.ID] = submission
	r.byUserID[submission.UserID] = submission.ID
	return submission, nil
}

func (r *KYCRepository) FindByUserID(_ context.Context, userID string) (domain.KYCSubmission, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	id, ok := r.byUserID[userID]
	if !ok {
		return domain.KYCSubmission{}, errRecordNotFound
	}

	return r.byID[id], nil
}

func (r *KYCRepository) Update(_ context.Context, submission domain.KYCSubmission) (domain.KYCSubmission, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.byID[submission.ID]; !ok {
		return domain.KYCSubmission{}, errRecordNotFound
	}

	r.byID[submission.ID] = submission
	r.byUserID[submission.UserID] = submission.ID
	return submission, nil
}
