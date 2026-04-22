package account

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/yourorg/gobank/internal/domain"
	"github.com/yourorg/gobank/internal/messaging"
	"github.com/yourorg/gobank/internal/repository"
	"go.uber.org/zap"
)

type UseCase struct {
	accountRepo repository.AccountRepository
	publisher   messaging.Publisher
	log         *zap.Logger
}

func New(accountRepo repository.AccountRepository, publisher messaging.Publisher, log *zap.Logger) *UseCase {
	return &UseCase{accountRepo: accountRepo, publisher: publisher, log: log}
}

func (uc *UseCase) CreateAccount(ctx context.Context, userID uuid.UUID, req domain.CreateAccountRequest) (*domain.Account, error) {
	existing, err := uc.accountRepo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("listing accounts: %w", err)
	}
	if len(existing) >= 5 {
		return nil, ErrMaxAccountsReached
	}

	now := time.Now().UTC()
	acct := &domain.Account{
		ID:            uuid.New(),
		UserID:        userID,
		AccountNumber: generateAccountNumber(),
		Type:          req.Type,
		Currency:      req.Currency,
		Balance:       0,
		Status:        domain.AccountStatusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := uc.accountRepo.Create(ctx, acct); err != nil {
		return nil, fmt.Errorf("creating account: %w", err)
	}

	_ = uc.publisher.Publish(ctx, messaging.SubjectAudit, domain.AuditEvent{
		AuditLog: domain.AuditLog{
			ID:         uuid.New(),
			UserID:     &userID,
			Action:     domain.AuditActionAccountCreated,
			ResourceID: acct.ID.String(),
			Metadata:   map[string]string{"account_number": acct.AccountNumber, "type": string(acct.Type)},
			CreatedAt:  now,
		},
	})

	return acct, nil
}

func (uc *UseCase) GetAccount(ctx context.Context, userID, accountID uuid.UUID) (*domain.Account, error) {
	acct, err := uc.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	// Ensure ownership
	if acct.UserID != userID {
		return nil, ErrForbidden
	}
	return acct, nil
}

func (uc *UseCase) ListAccounts(ctx context.Context, userID uuid.UUID) ([]domain.Account, error) {
	return uc.accountRepo.ListByUserID(ctx, userID)
}

// generateAccountNumber creates a unique 12-digit account number with a checksum.
func generateAccountNumber() string {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	prefix := "9900"
	suffix := fmt.Sprintf("%08d", rng.Intn(100000000))
	num := prefix + suffix
	return addLuhnCheckDigit(num)
}

func addLuhnCheckDigit(number string) string {
	sum := 0
	nDigits := len(number)
	parity := nDigits % 2
	for i := 0; i < nDigits; i++ {
		digit, _ := strconv.Atoi(string(number[i]))
		if i%2 == parity {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
	}
	checkDigit := (10 - (sum % 10)) % 10
	return number + strconv.Itoa(checkDigit)
}

var (
	ErrMaxAccountsReached = errors.New("maximum accounts per user reached")
	ErrForbidden          = errors.New("access forbidden")
)
