CREATE TABLE IF NOT EXISTS kyc_submissions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL UNIQUE REFERENCES users(id),
    document_type TEXT NOT NULL,
    document_number TEXT NOT NULL,
    status TEXT NOT NULL,
    notes TEXT NOT NULL DEFAULT '',
    submitted_at TIMESTAMPTZ NOT NULL,
    reviewed_at TIMESTAMPTZ
);
