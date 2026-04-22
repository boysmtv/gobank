-- name: CreateUser :one
INSERT INTO users (
    id, email, full_name, password_hash, status, kyc_status, failed_attempts, created_at, updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1 LIMIT 1;
