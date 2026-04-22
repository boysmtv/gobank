package domain

import (
	"time"

	"github.com/google/uuid"
)

type KYCStatus string

const (
	KYCStatusPending  KYCStatus = "pending"
	KYCStatusVerified KYCStatus = "verified"
	KYCStatusRejected KYCStatus = "rejected"
)

type KYCRecord struct {
	ID            uuid.UUID  `json:"id"`
	UserID        uuid.UUID  `json:"user_id"`
	FullLegalName string     `json:"full_legal_name"`
	DateOfBirth   time.Time  `json:"date_of_birth"`
	NationalID    string     `json:"national_id"`
	Address       string     `json:"address"`
	Status        KYCStatus  `json:"status"`
	RejectionNote string     `json:"rejection_note,omitempty"`
	SubmittedAt   time.Time  `json:"submitted_at"`
	VerifiedAt    *time.Time `json:"verified_at,omitempty"`
}

type KYCSubmitRequest struct {
	FullLegalName string `json:"full_legal_name" validate:"required,min=2,max=200"`
	DateOfBirth   string `json:"date_of_birth"   validate:"required"` // YYYY-MM-DD
	NationalID    string `json:"national_id"     validate:"required,min=5,max=50"`
	Address       string `json:"address"         validate:"required,min=10,max=500"`
}
