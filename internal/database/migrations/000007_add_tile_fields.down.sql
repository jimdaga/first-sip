DROP INDEX IF EXISTS idx_user_plugin_configs_order;
ALTER TABLE plugins DROP COLUMN IF EXISTS icon;
ALTER TABLE plugins DROP COLUMN IF EXISTS tile_size;
ALTER TABLE user_plugin_configs DROP COLUMN IF EXISTS display_order;
