CREATE TABLE plugins (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    owner VARCHAR(255),
    version VARCHAR(50) NOT NULL,
    schema_version VARCHAR(10) NOT NULL DEFAULT 'v1',
    capabilities JSONB,
    default_config JSONB,
    settings_schema_path VARCHAR(255),
    enabled BOOLEAN NOT NULL DEFAULT true
);

CREATE INDEX idx_plugins_deleted_at ON plugins(deleted_at);

CREATE TABLE user_plugin_configs (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plugin_id BIGINT NOT NULL REFERENCES plugins(id) ON DELETE CASCADE,
    settings JSONB,
    enabled BOOLEAN NOT NULL DEFAULT false,
    UNIQUE(user_id, plugin_id)
);

CREATE INDEX idx_user_plugin_configs_deleted_at ON user_plugin_configs(deleted_at);
CREATE INDEX idx_user_plugin_configs_user_id ON user_plugin_configs(user_id);
CREATE INDEX idx_user_plugin_configs_plugin_id ON user_plugin_configs(plugin_id);

CREATE TABLE plugin_runs (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    plugin_run_id VARCHAR(36) NOT NULL UNIQUE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plugin_id BIGINT NOT NULL REFERENCES plugins(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    input JSONB,
    output JSONB,
    error_message TEXT,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ
);

CREATE INDEX idx_plugin_runs_deleted_at ON plugin_runs(deleted_at);
CREATE INDEX idx_plugin_runs_user_id ON plugin_runs(user_id);
CREATE INDEX idx_plugin_runs_plugin_id ON plugin_runs(plugin_id);
CREATE INDEX idx_plugin_runs_status ON plugin_runs(status);
