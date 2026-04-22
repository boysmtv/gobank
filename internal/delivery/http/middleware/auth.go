package middleware

import (
	"context"
	"net/http"
	"strings"

	"gobank/pkg/response"
)

type contextKey string

const SubjectContextKey contextKey = "subject"

func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			response.Error(w, http.StatusUnauthorized, "missing bearer token")
			return
		}

		subject := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		ctx := context.WithValue(r.Context(), SubjectContextKey, subject)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
