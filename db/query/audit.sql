-- name: CreateAuditLog :exec
INSERT INTO audit_logs (
    id, actor_id, action, resource, resource_id, metadata, created_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: ListAuditLogsByActor :many
SELECT * FROM audit_logs WHERE actor_id = $1 ORDER BY created_at DESC;
