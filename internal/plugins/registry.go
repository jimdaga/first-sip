package plugins

import (
	"fmt"
	"log"
	"sort"
)

// Registry holds discovered plugins in memory, indexed by plugin name.
// Provides methods for registration, lookup, and listing.
type Registry struct {
	plugins map[string]*PluginMetadata
}

// NewRegistry creates a new empty plugin registry.
func NewRegistry() *Registry {
	return &Registry{
		plugins: make(map[string]*PluginMetadata),
	}
}

// Register adds a plugin to the registry.
// Returns an error if a plugin with the same name is already registered.
func (r *Registry) Register(meta *PluginMetadata) error {
	if _, exists := r.plugins[meta.Name]; exists {
		return fmt.Errorf("plugin already registered: %s", meta.Name)
	}
	r.plugins[meta.Name] = meta
	return nil
}

// Get retrieves a plugin by name.
// Returns the plugin metadata and a boolean indicating if it was found.
func (r *Registry) Get(name string) (*PluginMetadata, bool) {
	meta, ok := r.plugins[name]
	return meta, ok
}

// List returns all registered plugins as a slice, sorted by name
// for deterministic ordering.
func (r *Registry) List() []*PluginMetadata {
	plugins := make([]*PluginMetadata, 0, len(r.plugins))
	for _, meta := range r.plugins {
		plugins = append(plugins, meta)
	}

	// Sort by name for deterministic ordering
	sort.Slice(plugins, func(i, j int) bool {
		return plugins[i].Name < plugins[j].Name
	})

	return plugins
}

// Count returns the number of registered plugins.
func (r *Registry) Count() int {
	return len(r.plugins)
}

// LoadRegistry is a convenience function that discovers plugins from
// the specified directory and registers them in a new Registry.
//
// Duplicate plugin names are logged and skipped. An empty registry
// is not an error (no plugins found is valid).
func LoadRegistry(pluginDir string) (*Registry, error) {
	// Discover all plugins in directory
	discovered, err := DiscoverPlugins(pluginDir)
	if err != nil {
		return nil, err
	}

	// Create registry and register discovered plugins
	registry := NewRegistry()
	for _, meta := range discovered {
		if err := registry.Register(meta); err != nil {
			log.Printf("Warning: duplicate plugin name, skipping %s: %v", meta.Name, err)
			continue
		}
	}

	return registry, nil
}
