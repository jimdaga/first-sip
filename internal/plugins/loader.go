package plugins

import (
	"encoding/json"
	"log"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// InitPlugins discovers plugins from the specified directory,
// syncs their metadata to the database, and returns a populated registry.
//
// This function is called at application startup to:
// 1. Discover all plugins from the plugin directory
// 2. Sync discovered plugin metadata to the database (upsert pattern)
// 3. Return the in-memory registry for use by the application
//
// Non-fatal: logs warnings but does not fail if individual plugins have issues.
func InitPlugins(db *gorm.DB, pluginDir string) (*Registry, error) {
	// Discover plugins from disk
	registry, err := LoadRegistry(pluginDir)
	if err != nil {
		return nil, err
	}

	log.Printf("Discovered %d plugin(s) from %s", registry.Count(), pluginDir)

	// Sync each discovered plugin to database
	for _, meta := range registry.List() {
		if err := syncPluginToDB(db, meta); err != nil {
			log.Printf("Warning: failed to sync plugin %s to database: %v", meta.Name, err)
			continue
		}
		log.Printf("Synced plugin to database: %s (version %s)", meta.Name, meta.Version)
	}

	return registry, nil
}

// syncPluginToDB persists or updates a plugin's metadata in the database.
// Uses an upsert pattern: creates if new, updates if already exists.
func syncPluginToDB(db *gorm.DB, meta *PluginMetadata) error {
	// Marshal capabilities and default config to JSON
	capabilitiesJSON, err := json.Marshal(meta.Capabilities)
	if err != nil {
		return err
	}

	defaultConfigJSON, err := json.Marshal(meta.DefaultConfig)
	if err != nil {
		return err
	}

	// Check if plugin already exists
	var dbPlugin Plugin
	result := db.Where("name = ?", meta.Name).First(&dbPlugin)

	if result.Error == gorm.ErrRecordNotFound {
		// Plugin doesn't exist - create new record
		dbPlugin = Plugin{
			Name:               meta.Name,
			Description:        meta.Description,
			Owner:              meta.Owner,
			Version:            meta.Version,
			SchemaVersion:      meta.SchemaVersion,
			Capabilities:       datatypes.JSON(capabilitiesJSON),
			DefaultConfig:      datatypes.JSON(defaultConfigJSON),
			SettingsSchemaPath: meta.SettingsSchemaPath,
			Enabled:            true,
		}
		return db.Create(&dbPlugin).Error
	} else if result.Error != nil {
		return result.Error
	}

	// Plugin exists - update its metadata
	updates := map[string]interface{}{
		"description":          meta.Description,
		"owner":                meta.Owner,
		"version":              meta.Version,
		"schema_version":       meta.SchemaVersion,
		"capabilities":         datatypes.JSON(capabilitiesJSON),
		"default_config":       datatypes.JSON(defaultConfigJSON),
		"settings_schema_path": meta.SettingsSchemaPath,
	}

	return db.Model(&dbPlugin).Updates(updates).Error
}
