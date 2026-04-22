package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	redislib "github.com/redis/go-redis/v9"
	"github.com/yourorg/gobank/configs"
	httpdelivery "github.com/yourorg/gobank/internal/delivery/http"
	"github.com/yourorg/gobank/internal/delivery/http/handler"
	"github.com/yourorg/gobank/internal/messaging"
	pgRepo "github.com/yourorg/gobank/internal/repository/postgres"
	redisRepo "github.com/yourorg/gobank/internal/repository/redis"
	accountUC "github.com/yourorg/gobank/internal/usecase/account"
	authUC "github.com/yourorg/gobank/internal/usecase/auth"
	kycUC "github.com/yourorg/gobank/internal/usecase/kyc"
	txUC "github.com/yourorg/gobank/internal/usecase/transaction"
	"github.com/yourorg/gobank/pkg/jwt"
	"github.com/yourorg/gobank/pkg/logger"
	"github.com/yourorg/gobank/pkg/password"
	"github.com/yourorg/gobank/pkg/tracer"
	"go.uber.org/zap"
)

var Version = "dev"

func main() {
	// --- Configuration ---
	cfg, err := configs.Load()
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	// --- Logger ---
	log, err := logger.New(cfg.App.Env)
	if err != nil {
		panic("failed to init logger: " + err.Error())
	}
	defer log.Sync()

	// --- Tracer ---
	shutdownTracer, err := tracer.Init(cfg.Tracer.ServiceName, cfg.Tracer.JaegerEndpoint)
	if err != nil {
		log.Fatal("failed to init tracer", zap.Error(err))
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = shutdownTracer(ctx)
	}()

	// --- PostgreSQL ---
	poolCfg, err := pgxpool.ParseConfig(cfg.Database.DSN)
	if err != nil {
		log.Fatal("invalid database DSN", zap.Error(err))
	}
	poolCfg.MaxConns = int32(cfg.Database.MaxOpenConns)
	poolCfg.MinConns = int32(cfg.Database.MaxIdleConns)
	poolCfg.MaxConnLifetime = cfg.Database.ConnMaxLifetime

	dbPool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		log.Fatal("failed to connect to postgres", zap.Error(err))
	}
	defer dbPool.Close()

	if err := dbPool.Ping(context.Background()); err != nil {
		log.Fatal("postgres ping failed", zap.Error(err))
	}
	log.Info("postgres connected")

	// --- Redis ---
	rdb := redislib.NewClient(&redislib.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatal("redis ping failed", zap.Error(err))
	}
	log.Info("redis connected")

	// --- NATS ---
	nc, err := nats.Connect(cfg.NATS.URL,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(10),
		nats.ReconnectWait(2*time.Second),
	)
	if err != nil {
		log.Fatal("failed to connect to nats", zap.Error(err))
	}
	defer nc.Close()
	log.Info("nats connected")

	// --- Repositories ---
	userRepo := pgRepo.NewUserRepo(dbPool)
	accountRepo := pgRepo.NewAccountRepo(dbPool)
	txnRepo := pgRepo.NewTransactionRepo(dbPool)
	auditRepo := pgRepo.NewAuditRepo(dbPool)
	kycRepo := pgRepo.NewKYCRepo(dbPool)
	txManager := pgRepo.NewTxManager(dbPool)
	idemStore := redisRepo.NewIdempotencyStore(rdb)

	// --- Messaging ---
	publisher, err := messaging.NewNATSPublisher(nc, log)
	if err != nil {
		log.Fatal("failed to init NATS publisher", zap.Error(err))
	}

	auditConsumer, err := messaging.NewAuditConsumer(nc, auditRepo, log)
	if err != nil {
		log.Fatal("failed to init audit consumer", zap.Error(err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := auditConsumer.Start(ctx); err != nil {
		log.Fatal("failed to start audit consumer", zap.Error(err))
	}

	// --- JWT ---
	jwtMgr := jwt.NewManager(
		cfg.JWT.AccessSecret,
		cfg.JWT.RefreshSecret,
		cfg.JWT.AccessExpiry,
		cfg.JWT.RefreshExpiry,
	)

	// --- Argon2 params ---
	argon := &password.Params{
		Memory:      cfg.Argon2.Memory,
		Iterations:  cfg.Argon2.Iterations,
		Parallelism: cfg.Argon2.Parallelism,
		SaltLength:  cfg.Argon2.SaltLength,
		KeyLength:   cfg.Argon2.KeyLength,
	}

	// --- Use Cases ---
	authUseCase := authUC.New(userRepo, jwtMgr, argon, publisher, log)
	accountUseCase := accountUC.New(accountRepo, publisher, log)
	txUseCase := txUC.New(txnRepo, accountRepo, idemStore, txManager, publisher, log)
	kycUseCase := kycUC.New(kycRepo, userRepo, publisher, log)

	// --- Handlers ---
	authHandler := handler.NewAuthHandler(authUseCase)
	accountHandler := handler.NewAccountHandler(accountUseCase)
	txHandler := handler.NewTransactionHandler(txUseCase)
	kycHandler := handler.NewKYCHandler(kycUseCase)

	// --- HTTP Server ---
	httpServer := httpdelivery.NewServer(
		cfg.App.Port,
		jwtMgr,
		authHandler,
		accountHandler,
		txHandler,
		kycHandler,
		log,
	)

	// --- Graceful shutdown ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := httpServer.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("HTTP server error", zap.Error(err))
		}
	}()

	log.Info("gobank started", zap.Int("port", cfg.App.Port))

	<-quit
	log.Info("shutdown signal received")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	cancel() // Stop NATS consumers

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Error("server shutdown error", zap.Error(err))
	}

	log.Info("gobank stopped cleanly")
}
