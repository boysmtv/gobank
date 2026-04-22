package postgres

import (
	"context"
	"sync"

	"gobank/internal/domain"
)

type AuditRepository struct {
	mu    sync.RWMutex
	items []domain.AuditLog
}

func NewAuditRepository() *AuditRepository {
	return &AuditRepository{
		items: make([]domain.AuditLog, 0),
	}
}

func (r *AuditRepository) Create(_ context.Context, log domain.AuditLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.items = append(r.items, log)
	return nil
}

func (r *AuditRepository) List(_ context.Context, actorID string) ([]domain.AuditLog, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]domain.AuditLog, 0, len(r.items))
	for _, item := range r.items {
		if actorID == "" || item.ActorID == actorID {
			result = append(result, item)
		}
	}

	return result, nil
}
