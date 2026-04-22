package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"

	"gobank/pkg/response"
)

type visitor struct {
	count     int
	resetTime time.Time
}

func RateLimit(limit int, window time.Duration) func(http.Handler) http.Handler {
	var (
		mu       sync.Mutex
		visitors = make(map[string]visitor)
	)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			host, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				host = r.RemoteAddr
			}

			now := time.Now()

			mu.Lock()
			current := visitors[host]
			if now.After(current.resetTime) {
				current = visitor{resetTime: now.Add(window)}
			}
			current.count++
			visitors[host] = current
			mu.Unlock()

			if current.count > limit {
				response.Error(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
