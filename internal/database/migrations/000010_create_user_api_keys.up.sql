CREATE TABLE user_api_keys (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key_type VARCHAR(50) NOT NULL,
    provider VARCHAR(50) NOT NULL DEFAULT '',
    encrypted_value TEXT NOT NULL
);

CREATE INDEX idx_user_api_keys_deleted_at ON user_api_keys(deleted_at);
CREATE INDEX idx_user_api_keys_user_id ON user_api_keys(user_id);
CREATE UNIQUE INDEX idx_user_api_keys_unique_active ON user_api_keys(user_id, key_type, provider) WHERE deleted_at IS NULL;
