package middleware

import (
	"net/http"
	"time"

	"github.com/go-chi/httprate"
	"github.com/yourorg/gobank/pkg/response"
)

// RateLimitByIP limits requests per IP.
func RateLimitByIP(limit int, window time.Duration) func(http.Handler) http.Handler {
	return httprate.Limit(
		limit,
		window,
		httprate.WithKeyFuncs(httprate.KeyByIP),
		httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
			response.JSONError(w, http.StatusTooManyRequests, "RATE_LIMIT", "too many requests")
		}),
	)
}

// RateLimitStrict is for auth endpoints.
func RateLimitStrict() func(http.Handler) http.Handler {
	return RateLimitByIP(10, time.Minute)
}

// RateLimitStandard is for general API endpoints.
func RateLimitStandard() func(http.Handler) http.Handler {
	return RateLimitByIP(100, time.Minute)
}
