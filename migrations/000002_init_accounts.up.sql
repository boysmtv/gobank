CREATE TYPE account_type AS ENUM ('savings', 'current');
CREATE TYPE account_status AS ENUM ('active', 'frozen', 'closed');

CREATE TABLE accounts (
                          id             UUID           PRIMARY KEY DEFAULT uuid_generate_v4(),
                          user_id        UUID           NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                          account_number VARCHAR(20)    NOT NULL UNIQUE,
                          type           account_type   NOT NULL,
                          currency       CHAR(3)        NOT NULL,
                          balance        BIGINT         NOT NULL DEFAULT 0 CHECK (balance >= 0),
                          status         account_status NOT NULL DEFAULT 'active',
                          created_at     TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
                          updated_at     TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_accounts_user_id ON accounts(user_id);
CREATE INDEX idx_accounts_account_number ON accounts(account_number);