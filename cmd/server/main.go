package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gobank/configs"
	httpdelivery "gobank/internal/delivery/http"
	"gobank/internal/delivery/http/handler"
	"gobank/internal/repository/postgres"
	redisrepo "gobank/internal/repository/redis"
	accountuc "gobank/internal/usecase/account"
	authuc "gobank/internal/usecase/auth"
	kycuc "gobank/internal/usecase/kyc"
	transactionuc "gobank/internal/usecase/transaction"
	"gobank/pkg/jwt"
	"gobank/pkg/logger"
	"gobank/pkg/password"
)

func main() {
	cfg := configs.Load()
	log := logger.New()

	userRepo := postgres.NewUserRepository()
	accountRepo := postgres.NewAccountRepository()
	transactionRepo := postgres.NewTransactionRepository()
	auditRepo := postgres.NewAuditRepository()
	kycRepo := postgres.NewKYCRepository()
	idempotencyRepo := redisrepo.NewIdempotencyRepository()

	authUseCase := authuc.NewUseCase(
		userRepo,
		auditRepo,
		password.NewHasher(cfg.PasswordPepper),
		jwt.NewManager(cfg.JWTSecret, 24*time.Hour),
	)
	accountUseCase := accountuc.NewUseCase(accountRepo, auditRepo)
	transactionUseCase := transactionuc.NewUseCase(accountRepo, transactionRepo, auditRepo, idempotencyRepo)
	kycUseCase := kycuc.NewUseCase(kycRepo, userRepo, auditRepo)

	server := httpdelivery.NewServer(
		httpdelivery.AddrFromPort(cfg.HTTPPort),
		log,
		handler.NewAuthHandler(authUseCase),
		handler.NewAccountHandler(accountUseCase),
		handler.NewTransactionHandler(transactionUseCase),
		handler.NewKYCHandler(kycUseCase),
	)

	go func() {
		if err := server.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("failed to start server", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}
}
