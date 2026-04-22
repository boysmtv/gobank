-- name: CreateTransaction :one
INSERT INTO transactions (
    id, reference_id, source_id, destination_id, type, status, amount, description, created_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: ListTransactionsByAccountID :many
SELECT * FROM transactions
WHERE source_id = $1 OR destination_id = $1
ORDER BY created_at DESC;
