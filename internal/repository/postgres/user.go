package postgres

import (
	"context"
	"errors"
	"sync"

	"gobank/internal/domain"
)

var errRecordNotFound = errors.New("record not found")

type UserRepository struct {
	mu      sync.RWMutex
	byID    map[string]domain.User
	byEmail map[string]string
}

func NewUserRepository() *UserRepository {
	return &UserRepository{
		byID:    make(map[string]domain.User),
		byEmail: make(map[string]string),
	}
}

func (r *UserRepository) Create(_ context.Context, user domain.User) (domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.byID[user.ID] = user
	r.byEmail[user.Email] = user.ID
	return user, nil
}

func (r *UserRepository) FindByID(_ context.Context, id string) (domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, ok := r.byID[id]
	if !ok {
		return domain.User{}, errRecordNotFound
	}

	return user, nil
}

func (r *UserRepository) FindByEmail(_ context.Context, email string) (domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	id, ok := r.byEmail[email]
	if !ok {
		return domain.User{}, errRecordNotFound
	}

	user, ok := r.byID[id]
	if !ok {
		return domain.User{}, errRecordNotFound
	}

	return user, nil
}

func (r *UserRepository) Update(_ context.Context, user domain.User) (domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.byID[user.ID]; !ok {
		return domain.User{}, errRecordNotFound
	}

	r.byID[user.ID] = user
	r.byEmail[user.Email] = user.ID
	return user, nil
}
