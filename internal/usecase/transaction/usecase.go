package transaction

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/yourorg/gobank/internal/domain"
	"github.com/yourorg/gobank/internal/messaging"
	"github.com/yourorg/gobank/internal/repository"
	"github.com/yourorg/gobank/pkg/metrics"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

var tracer = otel.Tracer("transaction")

type UseCase struct {
	txnRepo     repository.TransactionRepository
	accountRepo repository.AccountRepository
	idemStore   repository.IdempotencyStore
	txManager   repository.TxManager
	publisher   messaging.Publisher
	log         *zap.Logger
}

func New(
	txnRepo repository.TransactionRepository,
	accountRepo repository.AccountRepository,
	idemStore repository.IdempotencyStore,
	txManager repository.TxManager,
	publisher messaging.Publisher,
	log *zap.Logger,
) *UseCase {
	return &UseCase{
		txnRepo:     txnRepo,
		accountRepo: accountRepo,
		idemStore:   idemStore,
		txManager:   txManager,
		publisher:   publisher,
		log:         log,
	}
}

// Transfer executes a funds transfer with full ACID compliance and idempotency.
func (uc *UseCase) Transfer(ctx context.Context, userID uuid.UUID, req domain.TransferRequest, ipAddr string) (*domain.Transaction, error) {
	ctx, span := tracer.Start(ctx, "usecase.Transfer")
	defer span.End()
	span.SetAttributes(
		attribute.String("idempotency_key", req.IdempotencyKey),
		attribute.String("from_account", req.FromAccountID),
		attribute.String("to_account", req.ToAccountID),
		attribute.Int64("amount", req.Amount),
	)

	// --- Idempotency check (Redis SETNX) ---
	idemKey := "transfer:" + req.IdempotencyKey
	existing, err := uc.idemStore.Get(ctx, idemKey)
	if err != nil {
		return nil, fmt.Errorf("idempotency check: %w", err)
	}
	if existing != nil {
		// Return cached response — idempotent replay
		var cached domain.Transaction
		if err := json.Unmarshal(existing, &cached); err != nil {
			return nil, fmt.Errorf("deserializing cached transaction: %w", err)
		}
		return &cached, nil
	}

	// Check DB for any existing transaction with this key (covers restart scenarios)
	if dbTxn, err := uc.txnRepo.GetByIdempotencyKey(ctx, req.IdempotencyKey); err == nil {
		return dbTxn, nil
	}

	fromID, err := uuid.Parse(req.FromAccountID)
	if err != nil {
		return nil, ErrInvalidAccount
	}
	toID, err := uuid.Parse(req.ToAccountID)
	if err != nil {
		return nil, ErrInvalidAccount
	}
	if fromID == toID {
		return nil, ErrSameAccount
	}

	now := time.Now().UTC()
	txnRecord := &domain.Transaction{
		ID:             uuid.New(),
		IdempotencyKey: req.IdempotencyKey,
		FromAccountID:  &fromID,
		ToAccountID:    &toID,
		Amount:         req.Amount,
		Currency:       req.Currency,
		Type:           domain.TransactionTypeTransfer,
		Status:         domain.TransactionStatusPending,
		Description:    req.Description,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	// Audit: initiation (before DB tx)
	_ = uc.publisher.Publish(ctx, messaging.SubjectAudit, domain.AuditEvent{
		AuditLog: domain.AuditLog{
			ID:         uuid.New(),
			UserID:     &userID,
			Action:     domain.AuditActionTransactionInit,
			ResourceID: txnRecord.ID.String(),
			IPAddress:  ipAddr,
			Metadata: map[string]string{
				"from": req.FromAccountID,
				"to":   req.ToAccountID,
				"amt":  fmt.Sprintf("%d", req.Amount),
				"ccy":  req.Currency,
			},
			CreatedAt: now,
		},
	})

	// --- ACID database transaction with serializable isolation ---
	var finalStatus domain.TransactionStatus
	txErr := uc.txManager.WithTransaction(ctx, func(ctx context.Context, tx repository.DBTX) error {
		// Lock both accounts in consistent order (lower UUID first) to prevent deadlock
		first, second := fromID, toID
		if fromID.String() > toID.String() {
			first, second = toID, fromID
		}

		fromAcct, err := uc.accountRepo.LockForUpdate(ctx, tx, first)
		if err != nil {
			return fmt.Errorf("locking account %s: %w", first, err)
		}
		toAcct, err := uc.accountRepo.LockForUpdate(ctx, tx, second)
		if err != nil {
			return fmt.Errorf("locking account %s: %w", second, err)
		}

		// Reassign if order was swapped
		if first == toID {
			fromAcct, toAcct = toAcct, fromAcct
		}

		// Validation inside transaction to prevent TOCTOU
		if fromAcct.UserID != userID {
			return ErrUnauthorized
		}
		if fromAcct.Status != domain.AccountStatusActive || toAcct.Status != domain.AccountStatusActive {
			return ErrAccountNotActive
		}
		if fromAcct.Currency != req.Currency || toAcct.Currency != req.Currency {
			return ErrCurrencyMismatch
		}
		if fromAcct.Balance < req.Amount {
			return ErrInsufficientFunds
		}

		// Insert transaction record
		if err := uc.txnRepo.Create(ctx, tx, txnRecord); err != nil {
			return fmt.Errorf("creating transaction: %w", err)
		}

		// Debit sender
		if err := uc.accountRepo.UpdateBalance(ctx, tx, fromID, -req.Amount); err != nil {
			return fmt.Errorf("debiting from: %w", err)
		}

		// Credit receiver
		if err := uc.accountRepo.UpdateBalance(ctx, tx, toID, req.Amount); err != nil {
			return fmt.Errorf("crediting to: %w", err)
		}

		// Mark success inside same transaction
		if err := uc.txnRepo.UpdateStatus(ctx, tx, txnRecord.ID, domain.TransactionStatusSuccess); err != nil {
			return fmt.Errorf("updating transaction status: %w", err)
		}

		finalStatus = domain.TransactionStatusSuccess
		return nil
	})

	if txErr != nil {
		finalStatus = domain.TransactionStatusFailed
		txnRecord.Status = domain.TransactionStatusFailed
		metrics.TransferTotal.WithLabelValues("failed", req.Currency).Inc()

		_ = uc.publisher.Publish(ctx, messaging.SubjectAudit, domain.AuditEvent{
			AuditLog: domain.AuditLog{
				ID:         uuid.New(),
				UserID:     &userID,
				Action:     domain.AuditActionTransactionFailed,
				ResourceID: txnRecord.ID.String(),
				IPAddress:  ipAddr,
				Metadata:   map[string]string{"reason": txErr.Error()},
				CreatedAt:  time.Now().UTC(),
			},
		})

		uc.log.Error("transfer failed",
			zap.String("txn_id", txnRecord.ID.String()),
			zap.Error(txErr),
		)
		return nil, txErr
	}

	txnRecord.Status = finalStatus
	metrics.TransferTotal.WithLabelValues("success", req.Currency).Inc()
	metrics.TransferAmount.WithLabelValues(req.Currency).Observe(float64(req.Amount))

	// Cache result for idempotency replay (TTL: 24h)
	if encoded, err := json.Marshal(txnRecord); err == nil {
		_, _ = uc.idemStore.Set(ctx, idemKey, encoded, 24*time.Hour)
	}

	_ = uc.publisher.Publish(ctx, messaging.SubjectAudit, domain.AuditEvent{
		AuditLog: domain.AuditLog{
			ID:         uuid.New(),
			UserID:     &userID,
			Action:     domain.AuditActionTransactionOK,
			ResourceID: txnRecord.ID.String(),
			IPAddress:  ipAddr,
			CreatedAt:  time.Now().UTC(),
		},
	})

	uc.log.Info("transfer complete",
		zap.String("txn_id", txnRecord.ID.String()),
		zap.Int64("amount", req.Amount),
		zap.String("currency", req.Currency),
	)

	return txnRecord, nil
}

func (uc *UseCase) GetTransaction(ctx context.Context, userID uuid.UUID, txnID uuid.UUID) (*domain.Transaction, error) {
	txn, err := uc.txnRepo.GetByID(ctx, txnID)
	if err != nil {
		return nil, err
	}
	// Verify ownership via account
	if txn.FromAccountID != nil {
		acct, err := uc.accountRepo.GetByID(ctx, *txn.FromAccountID)
		if err == nil && acct.UserID == userID {
			return txn, nil
		}
	}
	if txn.ToAccountID != nil {
		acct, err := uc.accountRepo.GetByID(ctx, *txn.ToAccountID)
		if err == nil && acct.UserID == userID {
			return txn, nil
		}
	}
	return nil, ErrForbidden
}

func (uc *UseCase) ListTransactions(ctx context.Context, userID uuid.UUID, req domain.TransactionListRequest) ([]domain.Transaction, int, error) {
	accountID, err := uuid.Parse(req.AccountID)
	if err != nil {
		return nil, 0, ErrInvalidAccount
	}
	// Ownership check
	acct, err := uc.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, 0, err
	}
	if acct.UserID != userID {
		return nil, 0, ErrForbidden
	}
	return uc.txnRepo.ListByAccountID(ctx, accountID, req.Page, req.PageSize)
}

var (
	ErrInvalidAccount    = errors.New("invalid account ID")
	ErrSameAccount       = errors.New("cannot transfer to same account")
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrAccountNotActive  = errors.New("account not active")
	ErrCurrencyMismatch  = errors.New("currency mismatch")
	ErrUnauthorized      = errors.New("unauthorized")
	ErrForbidden         = errors.New("forbidden")
)
