package httpdelivery

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"gobank/internal/delivery/http/handler"
	"gobank/internal/delivery/http/middleware"
)

type Server struct {
	httpServer *http.Server
	logger     *slog.Logger
}

func NewServer(
	addr string,
	logger *slog.Logger,
	authHandler *handler.AuthHandler,
	accountHandler *handler.AccountHandler,
	transactionHandler *handler.TransactionHandler,
	kycHandler *handler.KYCHandler,
) *Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/v1/auth/register", authHandler.Register)
	mux.HandleFunc("/v1/auth/login", authHandler.Login)
	mux.Handle("/v1/accounts", middleware.Auth(http.HandlerFunc(accountHandler.Handle)))
	mux.Handle("/v1/transactions/transfer", middleware.Auth(http.HandlerFunc(transactionHandler.Transfer)))
	mux.Handle("/v1/transactions", middleware.Auth(http.HandlerFunc(transactionHandler.History)))
	mux.Handle("/v1/kyc/submit", middleware.Auth(http.HandlerFunc(kycHandler.Submit)))

	baseHandler := chain(
		mux,
		middleware.RequestID,
		middleware.SecureHeaders,
		middleware.RateLimit(100, time.Minute),
	)

	return &Server{
		httpServer: &http.Server{
			Addr:              addr,
			Handler:           baseHandler,
			ReadHeaderTimeout: 5 * time.Second,
		},
		logger: logger,
	}
}

func (s *Server) Start() error {
	s.logger.Info("http server starting", "addr", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("http server shutting down")
	return s.httpServer.Shutdown(ctx)
}

func chain(base http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	wrapped := base
	for i := len(middlewares) - 1; i >= 0; i-- {
		wrapped = middlewares[i](wrapped)
	}
	return wrapped
}

func AddrFromPort(port int) string {
	return fmt.Sprintf(":%d", port)
}
