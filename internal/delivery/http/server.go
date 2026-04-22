package http

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/yourorg/gobank/internal/delivery/http/handler"
	"github.com/yourorg/gobank/internal/delivery/http/middleware"
	"github.com/yourorg/gobank/pkg/jwt"
	"go.uber.org/zap"
)

type Server struct {
	http *http.Server
	log  *zap.Logger
}

func NewServer(
	port int,
	jwtMgr *jwt.Manager,
	authH *handler.AuthHandler,
	accountH *handler.AccountHandler,
	txnH *handler.TransactionHandler,
	kycH *handler.KYCHandler,
	log *zap.Logger,
) *Server {
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.RequestID)
	r.Use(middleware.RequestID)
	r.Use(middleware.SecureHeaders)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(30 * time.Second))
	r.Use(middleware.RateLimitStandard())

	// Observability
	r.Handle("/metrics", promhttp.Handler())
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","service":"gobank"}`))
	})

	// API v1
	r.Route("/api/v1", func(r chi.Router) {
		// Public auth routes (stricter rate limit)
		r.Group(func(r chi.Router) {
			r.Use(middleware.RateLimitStrict())
			r.Post("/auth/register", authH.Register)
			r.Post("/auth/login", authH.Login)
			r.Post("/auth/refresh", authH.RefreshToken)
		})

		// Authenticated routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.Authenticate(jwtMgr))

			r.Post("/auth/logout", authH.Logout)

			// Account routes
			r.Route("/accounts", func(r chi.Router) {
				r.Post("/", accountH.Create)
				r.Get("/", accountH.List)
				r.Get("/{accountID}", accountH.Get)
			})

			// Transaction routes
			r.Route("/transactions", func(r chi.Router) {
				r.Post("/transfer", txnH.Transfer)
				r.Get("/", txnH.List)
				r.Get("/{txnID}", txnH.Get)
			})

			// KYC routes
			r.Route("/kyc", func(r chi.Router) {
				r.Post("/", kycH.Submit)
				r.Get("/", kycH.GetStatus)
			})

			// Admin routes
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireRole("admin"))
				r.Post("/admin/kyc/verify", kycH.AdminVerify)
			})
		})
	})

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return &Server{http: srv, log: log}
}

func (s *Server) Start() error {
	s.log.Info("HTTP server starting", zap.String("addr", s.http.Addr))
	return s.http.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.log.Info("HTTP server shutting down gracefully")
	return s.http.Shutdown(ctx)
}

func (s *Server) Handler() http.Handler {
	return s.http.Handler
}
