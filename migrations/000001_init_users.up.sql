CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TYPE user_role AS ENUM ('customer', 'admin', 'compliance');
CREATE TYPE user_status AS ENUM ('active', 'suspended', 'pending');

CREATE TABLE users (
                       id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
                       email         VARCHAR(255) NOT NULL UNIQUE,
                       password_hash TEXT NOT NULL,
                       full_name     VARCHAR(200) NOT NULL,
                       phone         VARCHAR(30)  NOT NULL,
                       role          user_role    NOT NULL DEFAULT 'customer',
                       status        user_status  NOT NULL DEFAULT 'pending',
                       created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
                       updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_status ON users(status);

CREATE TABLE refresh_tokens (
                                id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
                                user_id    UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                                token_hash TEXT         NOT NULL,
                                expires_at TIMESTAMPTZ  NOT NULL,
                                revoked    BOOLEAN      NOT NULL DEFAULT FALSE,
                                created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);