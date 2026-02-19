-- Add per-user scheduling fields to user_plugin_configs
-- cron_expression is nullable — NULL means scheduling is disabled for this config
ALTER TABLE user_plugin_configs
    ADD COLUMN cron_expression VARCHAR(100),
    ADD COLUMN timezone VARCHAR(100) NOT NULL DEFAULT 'UTC';

-- Partial index to accelerate the per-minute scheduler query:
-- only indexes rows that are enabled, not soft-deleted, and have a schedule set.
CREATE INDEX idx_user_plugin_configs_scheduled
    ON user_plugin_configs(enabled, cron_expression)
    WHERE deleted_at IS NULL
      AND enabled = true
      AND cron_expression IS NOT NULL;
