package postgres

import (
	"context"
	"sync"

	"gobank/internal/domain"
)

type TransactionRepository struct {
	mu          sync.RWMutex
	byID        map[string]domain.Transaction
	byReference map[string]string
	byAccountID map[string][]string
}

func NewTransactionRepository() *TransactionRepository {
	return &TransactionRepository{
		byID:        make(map[string]domain.Transaction),
		byReference: make(map[string]string),
		byAccountID: make(map[string][]string),
	}
}

func (r *TransactionRepository) Create(_ context.Context, tx domain.Transaction) (domain.Transaction, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.byID[tx.ID] = tx
	r.byReference[tx.ReferenceID] = tx.ID
	if tx.SourceID != "" {
		r.byAccountID[tx.SourceID] = append(r.byAccountID[tx.SourceID], tx.ID)
	}
	if tx.DestinationID != "" && tx.DestinationID != tx.SourceID {
		r.byAccountID[tx.DestinationID] = append(r.byAccountID[tx.DestinationID], tx.ID)
	}

	return tx, nil
}

func (r *TransactionRepository) FindByID(_ context.Context, id string) (domain.Transaction, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tx, ok := r.byID[id]
	if !ok {
		return domain.Transaction{}, errRecordNotFound
	}

	return tx, nil
}

func (r *TransactionRepository) FindByReferenceID(_ context.Context, referenceID string) (domain.Transaction, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	id, ok := r.byReference[referenceID]
	if !ok {
		return domain.Transaction{}, errRecordNotFound
	}

	return r.byID[id], nil
}

func (r *TransactionRepository) ListByAccountID(_ context.Context, accountID string) ([]domain.Transaction, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := r.byAccountID[accountID]
	result := make([]domain.Transaction, 0, len(ids))
	for _, id := range ids {
		result = append(result, r.byID[id])
	}

	return result, nil
}
