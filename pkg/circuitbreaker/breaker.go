package circuitbreaker

import (
	"errors"
	"sync"
	"time"
)

var ErrCircuitOpen = errors.New("circuit is open")

type Breaker struct {
	mu        sync.Mutex
	failures  int
	threshold int
	openUntil time.Time
	cooldown  time.Duration
}

func New(threshold int, cooldown time.Duration) *Breaker {
	return &Breaker{
		threshold: threshold,
		cooldown:  cooldown,
	}
}

func (b *Breaker) Do(fn func() error) error {
	b.mu.Lock()
	if time.Now().Before(b.openUntil) {
		b.mu.Unlock()
		return ErrCircuitOpen
	}
	b.mu.Unlock()

	err := fn()

	b.mu.Lock()
	defer b.mu.Unlock()

	if err != nil {
		b.failures++
		if b.failures >= b.threshold {
			b.openUntil = time.Now().Add(b.cooldown)
		}
		return err
	}

	b.failures = 0
	b.openUntil = time.Time{}
	return nil
}
