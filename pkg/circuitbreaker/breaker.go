package circuitbreaker

import (
	"fmt"
	"time"

	"github.com/sony/gobreaker"
)

// New creates a circuit breaker suitable for wrapping external calls (DB, Redis, NATS).
func New(name string) *gobreaker.CircuitBreaker {
	return gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        name,
		MaxRequests: 3,
		Interval:    10 * time.Second,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 5 && failureRatio >= 0.6
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			fmt.Printf("circuit breaker %q: %s → %s\n", name, from, to)
		},
	})
}
