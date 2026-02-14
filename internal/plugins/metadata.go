package plugins

import (
	"bytes"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// PluginMetadata represents the parsed plugin.yaml manifest file.
// All plugins must provide name and version; other fields are optional.
type PluginMetadata struct {
	Name               string                 `yaml:"name"`
	Description        string                 `yaml:"description"`
	Owner              string                 `yaml:"owner"`
	Version            string                 `yaml:"version"`
	SchemaVersion      string                 `yaml:"schema_version"`
	Capabilities       []string               `yaml:"capabilities"`
	DefaultConfig      map[string]interface{} `yaml:"default_config"`
	SettingsSchemaPath string                 `yaml:"settings_schema_path"`
}

// LoadPluginMetadata reads and parses a plugin.yaml file with strict validation.
// Unknown YAML fields are rejected (via KnownFields), and required fields are validated.
// SchemaVersion defaults to "v1" if not provided.
func LoadPluginMetadata(path string) (*PluginMetadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read plugin metadata: %w", err)
	}

	var meta PluginMetadata
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true) // CRITICAL: Reject unknown YAML keys to catch typos

	if err := decoder.Decode(&meta); err != nil {
		return nil, fmt.Errorf("failed to parse plugin metadata: %w", err)
	}

	// Set default schema version if not provided
	if meta.SchemaVersion == "" {
		meta.SchemaVersion = "v1"
	}

	// Validate required fields
	if meta.Name == "" {
		return nil, fmt.Errorf("plugin metadata missing required field: name")
	}
	if meta.Version == "" {
		return nil, fmt.Errorf("plugin metadata missing required field: version")
	}

	return &meta, nil
}
