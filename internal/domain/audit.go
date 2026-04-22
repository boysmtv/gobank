package domain

import "time"

type AuditLog struct {
	ID         string
	ActorID    string
	Action     string
	Resource   string
	ResourceID string
	Metadata   map[string]string
	CreatedAt  time.Time
}
