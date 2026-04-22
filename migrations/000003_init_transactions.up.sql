CREATE TYPE transaction_type AS ENUM ('transfer', 'deposit', 'withdrawal');
CREATE TYPE transaction_status AS ENUM ('pending', 'success', 'failed');

CREATE TABLE transactions (
                              id               UUID               PRIMARY KEY DEFAULT uuid_generate_v4(),
                              idempotency_key  VARCHAR(64)        NOT NULL UNIQUE,
                              from_account_id  UUID               REFERENCES accounts(id),
                              to_account_id    UUID               REFERENCES accounts(id),
                              amount           BIGINT             NOT NULL CHECK (amount > 0),
                              currency         CHAR(3)            NOT NULL,
                              type             transaction_type   NOT NULL,
                              status           transaction_status NOT NULL DEFAULT 'pending',
                              description      VARCHAR(255),
                              metadata         JSONB,
                              created_at       TIMESTAMPTZ        NOT NULL DEFAULT NOW(),
                              updated_at       TIMESTAMPTZ        NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_transactions_from_account ON transactions(from_account_id);
CREATE INDEX idx_transactions_to_account ON transactions(to_account_id);
CREATE INDEX idx_transactions_idempotency ON transactions(idempotency_key);
CREATE INDEX idx_transactions_created_at ON transactions(created_at DESC);
CREATE INDEX idx_transactions_status ON transactions(status);