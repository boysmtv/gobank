package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourorg/gobank/internal/domain"
)

type AuditRepo struct {
	db *pgxpool.Pool
}

func NewAuditRepo(db *pgxpool.Pool) *AuditRepo {
	return &AuditRepo{db: db}
}

func (r *AuditRepo) Create(ctx context.Context, log *domain.AuditLog) error {
	meta, _ := json.Marshal(log.Metadata)
	const q = `
        INSERT INTO audit_logs (id, user_id, action, resource_id, ip_address, user_agent, metadata, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.db.Exec(ctx, q,
		log.ID, log.UserID, log.Action, log.ResourceID,
		log.IPAddress, log.UserAgent, meta, log.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("auditRepo.Create: %w", err)
	}
	return nil
}

func (r *AuditRepo) ListByUserID(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]domain.AuditLog, error) {
	offset := (page - 1) * pageSize
	const q = `
        SELECT id, user_id, action, resource_id, ip_address, user_agent, metadata, created_at
        FROM audit_logs WHERE user_id = $1
        ORDER BY created_at DESC LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, q, userID, pageSize, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []domain.AuditLog
	for rows.Next() {
		var l domain.AuditLog
		var meta []byte
		if err := rows.Scan(
			&l.ID, &l.UserID, &l.Action, &l.ResourceID,
			&l.IPAddress, &l.UserAgent, &meta, &l.CreatedAt,
		); err != nil {
			return nil, err
		}
		if len(meta) > 0 {
			_ = json.Unmarshal(meta, &l.Metadata)
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}
