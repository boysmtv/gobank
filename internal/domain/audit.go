package domain

import (
	"time"

	"github.com/google/uuid"
)

type AuditAction string

const (
	AuditActionUserRegistered    AuditAction = "user.registered"
	AuditActionUserLogin         AuditAction = "user.login"
	AuditActionUserLogout        AuditAction = "user.logout"
	AuditActionAccountCreated    AuditAction = "account.created"
	AuditActionTransactionInit   AuditAction = "transaction.initiated"
	AuditActionTransactionOK     AuditAction = "transaction.success"
	AuditActionTransactionFailed AuditAction = "transaction.failed"
	AuditActionKYCSubmitted      AuditAction = "kyc.submitted"
	AuditActionKYCVerified       AuditAction = "kyc.verified"
)

// AuditLog is append-only. Never update, never delete.
type AuditLog struct {
	ID         uuid.UUID         `json:"id"`
	UserID     *uuid.UUID        `json:"user_id,omitempty"`
	Action     AuditAction       `json:"action"`
	ResourceID string            `json:"resource_id"`
	IPAddress  string            `json:"ip_address"`
	UserAgent  string            `json:"user_agent"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
}

// AuditEvent is the Kafka/NATS message payload.
type AuditEvent struct {
	AuditLog
}
