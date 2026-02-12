CREATE TABLE briefings (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content JSONB,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    error_message TEXT,
    read_at TIMESTAMPTZ
);

CREATE INDEX idx_briefings_deleted_at ON briefings(deleted_at);
CREATE INDEX idx_briefings_user_id ON briefings(user_id);
CREATE INDEX idx_briefings_status ON briefings(status);
CREATE INDEX idx_briefings_user_status ON briefings(user_id, status);
