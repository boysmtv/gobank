package redis

import (
	"context"
	"sync"
)

type IdempotencyRepository struct {
	mu   sync.Mutex
	keys map[string]struct{}
}

func NewIdempotencyRepository() *IdempotencyRepository {
	return &IdempotencyRepository{
		keys: make(map[string]struct{}),
	}
}

func (r *IdempotencyRepository) Reserve(_ context.Context, key string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.keys[key]; exists {
		return false, nil
	}

	r.keys[key] = struct{}{}
	return true, nil
}

func (r *IdempotencyRepository) Release(_ context.Context, key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.keys, key)
	return nil
}
