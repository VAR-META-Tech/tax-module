CREATE TABLE IF NOT EXISTS access_tokens (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider      VARCHAR(100) NOT NULL,
    access_token  TEXT NOT NULL,
    token_type    VARCHAR(50),
    expires_at    TIMESTAMPTZ NOT NULL,
    refresh_token TEXT,
    scope         TEXT,
    raw_response  JSONB,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Only one active token per provider
CREATE UNIQUE INDEX idx_access_tokens_provider ON access_tokens(provider);
