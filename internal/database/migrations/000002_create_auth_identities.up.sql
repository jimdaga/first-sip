CREATE TABLE auth_identities (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL,
    provider_user_id VARCHAR(255) NOT NULL,
    access_token TEXT,
    refresh_token TEXT,
    token_expiry TIMESTAMPTZ
);

CREATE INDEX idx_auth_identities_deleted_at ON auth_identities(deleted_at);
CREATE INDEX idx_auth_identities_user_id ON auth_identities(user_id);
CREATE UNIQUE INDEX idx_auth_identities_provider_user ON auth_identities(provider, provider_user_id) WHERE deleted_at IS NULL;
