CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    email VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL DEFAULT '',
    timezone VARCHAR(50) NOT NULL DEFAULT 'UTC',
    preferred_briefing_time VARCHAR(5) NOT NULL DEFAULT '06:00',
    role VARCHAR(20) NOT NULL DEFAULT 'user',
    last_login_at TIMESTAMPTZ,
    last_briefing_at TIMESTAMPTZ
);

CREATE INDEX idx_users_deleted_at ON users(deleted_at);
CREATE UNIQUE INDEX idx_users_email ON users(email) WHERE deleted_at IS NULL;
