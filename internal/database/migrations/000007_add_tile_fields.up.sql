-- Add tile display fields to plugins table
ALTER TABLE plugins
    ADD COLUMN IF NOT EXISTS icon VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS tile_size VARCHAR(10) NOT NULL DEFAULT '1x1';

-- Add display order for user tile arrangement
ALTER TABLE user_plugin_configs
    ADD COLUMN IF NOT EXISTS display_order INTEGER;

-- Index for dashboard query: enabled plugins ordered by display_order
CREATE INDEX IF NOT EXISTS idx_user_plugin_configs_order
    ON user_plugin_configs(user_id, display_order)
    WHERE deleted_at IS NULL AND enabled = true;
