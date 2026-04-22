package repository

import (
	"context"

	"gobank/internal/domain"
)

type UserRepository interface {
	Create(context.Context, domain.User) (domain.User, error)
	FindByID(context.Context, string) (domain.User, error)
	FindByEmail(context.Context, string) (domain.User, error)
	Update(context.Context, domain.User) (domain.User, error)
}

type AccountRepository interface {
	Create(context.Context, domain.Account) (domain.Account, error)
	FindByID(context.Context, string) (domain.Account, error)
	FindByUserID(context.Context, string) ([]domain.Account, error)
	Update(context.Context, domain.Account) (domain.Account, error)
}

type TransactionRepository interface {
	Create(context.Context, domain.Transaction) (domain.Transaction, error)
	FindByID(context.Context, string) (domain.Transaction, error)
	FindByReferenceID(context.Context, string) (domain.Transaction, error)
	ListByAccountID(context.Context, string) ([]domain.Transaction, error)
}

type AuditRepository interface {
	Create(context.Context, domain.AuditLog) error
	List(context.Context, string) ([]domain.AuditLog, error)
}

type KYCRepository interface {
	Create(context.Context, domain.KYCSubmission) (domain.KYCSubmission, error)
	FindByUserID(context.Context, string) (domain.KYCSubmission, error)
	Update(context.Context, domain.KYCSubmission) (domain.KYCSubmission, error)
}

type IdempotencyRepository interface {
	Reserve(context.Context, string) (bool, error)
	Release(context.Context, string) error
}
