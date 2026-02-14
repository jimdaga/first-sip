package plugins

import (
	"log"
	"os"
	"path/filepath"
)

// DiscoverPlugins scans the specified directory for plugin subdirectories
// containing plugin.yaml manifest files. Invalid plugins are logged and
// skipped (not fatal) to allow partial discovery.
//
// Returns all successfully loaded plugin metadata.
func DiscoverPlugins(pluginDir string) ([]*PluginMetadata, error) {
	var plugins []*PluginMetadata

	// List all entries in the plugin directory
	entries, err := os.ReadDir(pluginDir)
	if err != nil {
		return nil, err
	}

	// Scan each subdirectory for plugin.yaml
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Construct path to plugin.yaml
		manifestPath := filepath.Join(pluginDir, entry.Name(), "plugin.yaml")

		// Check if plugin.yaml exists
		if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
			continue // Skip directories without plugin.yaml
		}

		// Attempt to load plugin metadata
		meta, err := LoadPluginMetadata(manifestPath)
		if err != nil {
			log.Printf("Warning: failed to load plugin from %s: %v", entry.Name(), err)
			continue // Log and skip invalid plugins
		}

		plugins = append(plugins, meta)
	}

	return plugins, nil
}
