---
phase: 08-plugin-framework-foundation
plan: 03
subsystem: plugin-framework
tags: [plugins, startup, example-plugin, integration]

dependency_graph:
  requires:
    - 08-01-plugin-registry
    - 08-02-plugin-database
  provides:
    - plugin-startup-integration
    - example-plugin-daily-news-digest
    - plugin-discovery-sync
  affects:
    - application-startup
    - database-seed
    - configuration

tech_stack:
  added:
    - plugins/daily-news-digest (example plugin)
  patterns:
    - Plugin discovery at startup
    - Database upsert pattern for plugin metadata
    - Non-fatal plugin initialization
    - Environment-based plugin directory configuration

key_files:
  created:
    - plugins/daily-news-digest/plugin.yaml
    - plugins/daily-news-digest/settings.schema.json
    - plugins/daily-news-digest/crew/.gitkeep
    - internal/plugins/loader.go
  modified:
    - cmd/server/main.go
    - internal/database/seed.go
    - internal/config/config.go

decisions:
  - title: "Non-fatal plugin initialization"
    rationale: "Application can still serve v1.0 functionality if plugin loading fails"
    alternatives: ["Fatal error on plugin failure"]
    outcome: "Graceful degradation with warning logs"

  - title: "Plugin directory via config field"
    rationale: "Allows environment-based override via PLUGIN_DIR while maintaining ./plugins default"
    alternatives: ["Hardcoded path", "CLI flag"]
    outcome: "Config-based with env var support"

  - title: "Seed data creates plugin config conditionally"
    rationale: "Plugins may not be loaded yet when seed runs for the first time"
    alternatives: ["Require plugins to exist", "Separate seed step"]
    outcome: "Non-blocking warning if plugin not found"

metrics:
  duration_minutes: 2
  tasks_completed: 2
  files_created: 4
  files_modified: 3
  commits: 2
  test_coverage: manual
  completed_at: "2026-02-14"
---

# Phase 08 Plan 03: Plugin Startup Integration & Example Plugin

**One-liner:** Example daily-news-digest plugin with YAML metadata, JSON Schema settings, and automatic database sync at application startup via InitPlugins loader.

## What Was Built

### 1. Example Plugin: daily-news-digest

Created a complete example plugin demonstrating the plugin framework:

**Metadata (plugin.yaml):**
- Name: daily-news-digest
- Version: 1.0.0
- Schema version: v1
- Capabilities: briefing, scheduled
- Default config: daily frequency at 06:00, topics [technology, business]
- Settings schema reference

**Settings Schema (settings.schema.json):**
- JSON Schema Draft 2020-12 compliant
- Configurable fields:
  - `frequency`: enum [daily, weekly]
  - `preferred_time`: HH:MM pattern validation
  - `topics`: array with 1-5 items from predefined categories
  - `summary_length`: enum [brief, standard, detailed]
- Required fields: frequency, preferred_time, topics

**Directory Structure:**
```
plugins/daily-news-digest/
├── plugin.yaml
├── settings.schema.json
└── crew/
    └── .gitkeep (reserved for Phase 9 CrewAI workflow)
```

### 2. Plugin Startup Integration

**InitPlugins Loader (internal/plugins/loader.go):**
- Discovers plugins from configured directory using LoadRegistry
- Syncs each discovered plugin to database with upsert pattern:
  - New plugins: creates Plugin record
  - Existing plugins: updates metadata from YAML (version, description, etc.)
- Marshals capabilities and default_config to JSONB
- Returns populated Registry for application use
- Logs discovery count and individual sync results

**Application Wiring (cmd/server/main.go):**
- Plugin initialization runs after database migrations and seed data
- Before mode branching (worker vs web server)
- Non-fatal error handling: logs warning but doesn't block startup
- Registry stored in `pluginRegistry` variable for future use

**Configuration (internal/config/config.go):**
- Added `PluginDir` field to Config struct
- Default value: "./plugins"
- Override via PLUGIN_DIR environment variable

**Seed Data (internal/database/seed.go):**
- Creates UserPluginConfig for dev user with daily-news-digest
- Settings: daily frequency at 07:00, topics [technology, business]
- Conditional creation: only if plugin exists (non-blocking)
- Updated seed log to include plugin config count

## Verification Results

### Build Verification
- Application compiles successfully with plugin system integrated
- No import cycles or type errors
- All new code follows Go conventions

### File Verification
- plugin.yaml contains all required fields (name, version, schema_version)
- settings.schema.json is valid JSON Schema Draft 2020-12
- All referenced files exist at expected paths

### Integration Points
- main.go imports plugins package correctly
- LoadRegistry called from InitPlugins
- Plugin models referenced in seed.go
- Config field flows through to InitPlugins call

## Deviations from Plan

None - plan executed exactly as written.

## Technical Details

### Database Sync Pattern

The upsert implementation uses GORM's Where + First + Create/Updates pattern:

```go
var dbPlugin Plugin
result := db.Where("name = ?", meta.Name).First(&dbPlugin)

if result.Error == gorm.ErrRecordNotFound {
    // Create new plugin
    dbPlugin = Plugin{ ... }
    return db.Create(&dbPlugin).Error
}

// Update existing plugin
updates := map[string]interface{}{ ... }
return db.Model(&dbPlugin).Updates(updates).Error
```

This ensures plugin metadata stays in sync with YAML files across application restarts.

### Startup Sequence

```
1. Load config (includes PluginDir)
2. Initialize database connection
3. Run migrations (creates plugins tables)
4. Seed dev data (creates user, briefings)
5. Initialize plugins (discovers, syncs to DB) ← NEW
6. Branch to worker or web server mode
```

Plugins are initialized after seed data so that the Plugin table exists when seed tries to create UserPluginConfig.

### Non-Fatal Error Handling

Plugin initialization failures log warnings but don't crash the application. This allows:
- App to serve existing v1.0 functionality even if plugin system has issues
- Gradual rollout of plugin features
- Recovery from plugin directory misconfiguration

## Files Changed

### Created
- `plugins/daily-news-digest/plugin.yaml` - Example plugin metadata
- `plugins/daily-news-digest/settings.schema.json` - User settings schema
- `plugins/daily-news-digest/crew/.gitkeep` - Reserved directory
- `internal/plugins/loader.go` - Startup integration logic

### Modified
- `cmd/server/main.go` - Added plugin initialization step
- `internal/database/seed.go` - Added UserPluginConfig seed data
- `internal/config/config.go` - Added PluginDir configuration field

## Next Steps

This plan completes Phase 08 Wave 2. The plugin framework is now:
- ✅ Defined (registry, discovery, validation)
- ✅ Persisted (database models, migrations)
- ✅ Integrated (startup loading, example plugin)

**Ready for Phase 09:** CrewAI workflow implementation for the daily-news-digest plugin. The crew/ directory is reserved and the plugin metadata is in place.

**Ready for Phase 10:** Per-user plugin scheduling using the UserPluginConfig records.

**Ready for Phase 11:** Plugin settings UI can now query Plugin and UserPluginConfig tables.

## Commits

- `538496f`: feat(08-03): create daily-news-digest example plugin
- `0c9dab9`: feat(08-03): wire plugin discovery into application startup

## Self-Check: PASSED

Verified all created files exist and commits are in history:

✅ plugins/daily-news-digest/plugin.yaml
✅ plugins/daily-news-digest/settings.schema.json
✅ plugins/daily-news-digest/crew/.gitkeep
✅ internal/plugins/loader.go
✅ Commit 538496f present
✅ Commit 0c9dab9 present
