CREATE TYPE kyc_status AS ENUM ('pending', 'verified', 'rejected');

CREATE TABLE kyc_records (
                             id              UUID       PRIMARY KEY DEFAULT uuid_generate_v4(),
                             user_id         UUID       NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
                             full_legal_name VARCHAR(200) NOT NULL,
                             date_of_birth   DATE       NOT NULL,
                             national_id     VARCHAR(50) NOT NULL,
                             address         TEXT       NOT NULL,
                             status          kyc_status NOT NULL DEFAULT 'pending',
                             rejection_note  TEXT,
                             submitted_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                             verified_at     TIMESTAMPTZ
);

CREATE INDEX idx_kyc_records_user_id ON kyc_records(user_id);
CREATE INDEX idx_kyc_records_status ON kyc_records(status);