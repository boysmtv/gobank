CREATE TABLE IF NOT EXISTS transactions (
    id TEXT PRIMARY KEY,
    reference_id TEXT NOT NULL UNIQUE,
    source_id TEXT NOT NULL REFERENCES accounts(id),
    destination_id TEXT NOT NULL REFERENCES accounts(id),
    type TEXT NOT NULL,
    status TEXT NOT NULL,
    amount BIGINT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL
);
