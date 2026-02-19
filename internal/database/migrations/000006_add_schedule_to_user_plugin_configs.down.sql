-- Rollback: remove scheduling fields from user_plugin_configs
DROP INDEX IF EXISTS idx_user_plugin_configs_scheduled;

ALTER TABLE user_plugin_configs
    DROP COLUMN IF EXISTS cron_expression,
    DROP COLUMN IF EXISTS timezone;
