package domain

import "time"

type KYCStatus string

const (
	KYCStatusNotSubmitted KYCStatus = "not_submitted"
	KYCStatusReview       KYCStatus = "review"
	KYCStatusApproved     KYCStatus = "approved"
	KYCStatusRejected     KYCStatus = "rejected"
)

type KYCSubmission struct {
	ID             string
	UserID         string
	DocumentType   string
	DocumentNumber string
	Status         KYCStatus
	Notes          string
	SubmittedAt    time.Time
	ReviewedAt     *time.Time
}
