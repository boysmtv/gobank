package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type IdempotencyStore struct {
	client *redis.Client
	prefix string
}

func NewIdempotencyStore(client *redis.Client) *IdempotencyStore {
	return &IdempotencyStore{client: client, prefix: "idem:"}
}

// Set stores key-value pair with TTL using SET NX (atomic, no-overwrite).
func (s *IdempotencyStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error) {
	result, err := s.client.SetNX(ctx, s.prefix+key, value, ttl).Result()
	if err != nil {
		return false, err
	}
	return result, nil
}

func (s *IdempotencyStore) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := s.client.Get(ctx, s.prefix+key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	return val, err
}

func (s *IdempotencyStore) Delete(ctx context.Context, key string) error {
	return s.client.Del(ctx, s.prefix+key).Err()
}
