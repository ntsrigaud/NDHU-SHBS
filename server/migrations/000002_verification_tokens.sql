-- 000002_verification_tokens.sql
-- Email / password-reset verification tokens.

CREATE TABLE IF NOT EXISTS verification_tokens (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT        NOT NULL UNIQUE,   -- SHA-256 of the raw token
    type       TEXT        NOT NULL CHECK (type IN ('email_verification','password_reset')),
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_verification_tokens_user_id    ON verification_tokens (user_id);
CREATE INDEX IF NOT EXISTS idx_verification_tokens_expires_at ON verification_tokens (expires_at);
