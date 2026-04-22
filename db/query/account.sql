-- name: CreateAccount :one
INSERT INTO accounts (
    id, user_id, number, type, currency, balance, active, created_at, updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: ListAccountsByUserID :many
SELECT * FROM accounts WHERE user_id = $1 ORDER BY created_at DESC;
