CREATE TABLE account_tiers (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    name VARCHAR(50) NOT NULL UNIQUE,
    max_enabled_plugins INTEGER NOT NULL,
    min_frequency_hours INTEGER NOT NULL
);

CREATE INDEX idx_account_tiers_deleted_at ON account_tiers(deleted_at);

ALTER TABLE users ADD COLUMN IF NOT EXISTS account_tier_id BIGINT REFERENCES account_tiers(id);

CREATE INDEX IF NOT EXISTS idx_users_account_tier_id ON users(account_tier_id);
