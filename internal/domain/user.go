package domain

import "time"

type UserStatus string

const (
	UserStatusPending UserStatus = "pending"
	UserStatusActive  UserStatus = "active"
	UserStatusBlocked UserStatus = "blocked"
)

type User struct {
	ID             string
	Email          string
	FullName       string
	PasswordHash   string
	Status         UserStatus
	KYCStatus      KYCStatus
	CreatedAt      time.Time
	UpdatedAt      time.Time
	LastLoginAt    *time.Time
	FailedAttempts int
}
