CREATE TABLE audit_logs (
                            id          UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
                            user_id     UUID        REFERENCES users(id),
                            action      VARCHAR(80) NOT NULL,
                            resource_id VARCHAR(80) NOT NULL,
                            ip_address  VARCHAR(45),
                            user_agent  TEXT,
                            metadata    JSONB,
                            created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Audit logs are append-only. Revoke UPDATE and DELETE from application role.
-- ALTER TABLE audit_logs DISABLE ROW LEVEL SECURITY;  -- done at DB level in prod

CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at DESC);