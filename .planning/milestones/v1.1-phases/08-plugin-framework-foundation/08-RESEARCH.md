# Phase 08: Plugin Framework Foundation - Research

**Researched:** 2026-02-14
**Domain:** Plugin architecture with YAML metadata, JSON Schema validation, GORM models, Redis Streams communication
**Confidence:** HIGH

## Summary

Phase 8 establishes the foundation for a plugin system where plugins are defined by YAML metadata files in a `/plugins` directory, validated against JSON Schema for settings, persisted in PostgreSQL via GORM models, and communicate with CrewAI Python workflows via Redis Streams.

The standard approach uses directory-based plugin discovery (scanning for `plugin.yaml` files), gopkg.in/yaml.v3 for metadata parsing with struct tag validation, kaptinlin/jsonschema for dynamic settings validation (Draft 2020-12 compliant), GORM AutoMigrate for initial model creation with a manual migration strategy for production, and go-redis for Redis Streams producer pattern to enqueue plugin execution requests.

This phase prioritizes schema versioning from day one to prevent metadata/state mismatches, uses database-backed scheduling (not per-user Redis cron entries) to avoid O(users × plugins) scaling issues, and creates one complete example plugin (daily news digest) to validate the entire architecture end-to-end.

**Primary recommendation:** Use gopkg.in/yaml.v3 with KnownFields() validation for metadata, kaptinlin/jsonschema for user settings validation, GORM models with explicit schema versioning fields, and robinjoseph08/redisqueue or direct go-redis XAdd for Redis Streams producers. Start with AutoMigrate during development but prepare migration scripts for production deployment.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| gopkg.in/yaml.v3 | latest (stable) | YAML metadata parsing | Official Go YAML library, YAML 1.2 compliant, struct tag validation with KnownFields() |
| github.com/kaptinlin/jsonschema | latest | JSON Schema validation for plugin settings | Draft 2020-12 compliant, zero-copy struct validation, Google-backed quality |
| gorm.io/gorm | v1.31.1 (current) | Plugin/UserPluginConfig/PluginRun models | Already in project, proven migration support via Atlas |
| github.com/redis/go-redis/v9 | v9.14.1 (current) | Redis Streams producer (XAdd) | Already in project, official Redis client for Go |
| github.com/google/uuid | v1.6.0 (current) | Plugin execution run IDs | Already in project, RFC 4122 UUID generation |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| github.com/robinjoseph08/redisqueue | latest | Higher-level Redis Streams queue abstraction | If team prefers Producer/Consumer API over raw XAdd/XReadGroup |
| filepath | stdlib | Plugin directory discovery | Scanning /plugins for plugin.yaml files |
| encoding/json | stdlib | Marshaling plugin metadata for storage | Storing parsed YAML in database as JSON |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| gopkg.in/yaml.v3 | goccy/go-yaml | goccy has colored output and better errors, but gopkg.in/yaml.v3 is more widely adopted and stable |
| kaptinlin/jsonschema | github.com/santhosh-tekuri/jsonschema/v5 | santhosh-tekuri supports multiple drafts, but kaptinlin has better performance (zero-copy validation) |
| Database-backed scheduling | Per-user Asynq cron entries | Per-user cron scales poorly (O(users × plugins) Redis memory), database-backed allows query-based scheduling |
| Redis Streams | Asynq tasks directly | Streams provide better decoupling for Go → Python communication with consumer groups and persistence |

**Installation:**
```bash
# Add to go.mod (yaml.v3 and go-redis already available via transitive deps)
go get github.com/kaptinlin/jsonschema
# redisqueue is optional if using higher-level queue API
go get github.com/robinjoseph08/redisqueue
```

## Architecture Patterns

### Recommended Project Structure
```
/plugins/                              # Plugin definitions directory
  /daily-news-digest/                  # Example plugin (one per subdirectory)
    plugin.yaml                        # Plugin metadata manifest
    settings.schema.json               # JSON Schema for user-configurable settings
    crew/                              # CrewAI workflow (Python)
      main.py                          # Crew entry point
      agents.yaml                      # Agent definitions
      tasks.yaml                       # Task definitions
      requirements.txt                 # Python dependencies

/internal/plugins/                     # Go plugin infrastructure
  discovery.go                         # Directory scanning, YAML loading
  registry.go                          # In-memory plugin registry
  models.go                            # GORM models: Plugin, UserPluginConfig, PluginRun
  validator.go                         # JSON Schema validation for settings
  producer.go                          # Redis Streams producer for execution requests
```

### Pattern 1: Plugin Metadata Loading with Validation
**What:** Load YAML metadata with struct tag validation to catch unknown fields early
**When to use:** Every plugin load at startup
**Example:**
```go
// Source: https://pkg.go.dev/gopkg.in/yaml.v3
package plugins

import (
    "gopkg.in/yaml.v3"
    "os"
)

type PluginMetadata struct {
    Name         string            `yaml:"name"`
    Description  string            `yaml:"description"`
    Owner        string            `yaml:"owner"`
    Version      string            `yaml:"version"`          // Semantic version
    SchemaVersion string           `yaml:"schema_version"`   // Metadata schema version (v1, v2, etc.)
    Capabilities []string          `yaml:"capabilities"`     // e.g., ["briefing", "scheduled"]
    DefaultConfig map[string]interface{} `yaml:"default_config"`
    SettingsSchemaPath string      `yaml:"settings_schema_path"` // Relative path to JSON Schema file
}

func LoadPluginMetadata(path string) (*PluginMetadata, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }

    var meta PluginMetadata
    decoder := yaml.NewDecoder(bytes.NewReader(data))
    decoder.KnownFields(true) // CRITICAL: Fail on unknown YAML keys

    if err := decoder.Decode(&meta); err != nil {
        return nil, err // Returns TypeError if fields don't match struct
    }

    // Validate required fields
    if meta.SchemaVersion == "" {
        meta.SchemaVersion = "v1" // Default for backward compatibility
    }

    return &meta, nil
}
```

### Pattern 2: JSON Schema Validation for User Settings
**What:** Validate user-provided plugin settings against plugin's JSON Schema before saving
**When to use:** When user updates plugin configuration via UI
**Example:**
```go
// Source: https://github.com/kaptinlin/jsonschema
package plugins

import (
    "encoding/json"
    "github.com/kaptinlin/jsonschema"
)

func ValidateUserSettings(schemaPath string, userSettings map[string]interface{}) error {
    // Load schema from plugin directory
    schemaData, err := os.ReadFile(schemaPath)
    if err != nil {
        return fmt.Errorf("failed to load schema: %w", err)
    }

    // Compile schema
    compiler := jsonschema.NewCompiler()
    schema, err := compiler.Compile(schemaData)
    if err != nil {
        return fmt.Errorf("invalid schema: %w", err)
    }

    // Convert user settings to JSON for validation
    settingsJSON, err := json.Marshal(userSettings)
    if err != nil {
        return err
    }

    // Validate
    result := schema.ValidateJSON(settingsJSON)
    if !result.IsValid() {
        // Collect validation errors
        var errMsgs []string
        for field, err := range result.Errors {
            errMsgs = append(errMsgs, fmt.Sprintf("%s: %s", field, err.Message))
        }
        return fmt.Errorf("validation failed: %s", strings.Join(errMsgs, "; "))
    }

    return nil
}
```

### Pattern 3: Directory-Based Plugin Discovery
**What:** Scan /plugins directory for subdirectories containing plugin.yaml files
**When to use:** Application startup, before worker initialization
**Example:**
```go
// Source: Standard Go plugin discovery pattern
package plugins

import (
    "path/filepath"
    "os"
)

func DiscoverPlugins(pluginDir string) ([]*PluginMetadata, error) {
    var plugins []*PluginMetadata

    // Walk plugin directory
    entries, err := os.ReadDir(pluginDir)
    if err != nil {
        return nil, err
    }

    for _, entry := range entries {
        if !entry.IsDir() {
            continue
        }

        // Look for plugin.yaml in each subdirectory
        manifestPath := filepath.Join(pluginDir, entry.Name(), "plugin.yaml")
        if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
            continue // Skip directories without plugin.yaml
        }

        meta, err := LoadPluginMetadata(manifestPath)
        if err != nil {
            log.Printf("Failed to load plugin %s: %v", entry.Name(), err)
            continue // Log and skip invalid plugins
        }

        plugins = append(plugins, meta)
    }

    return plugins, nil
}
```

### Pattern 4: Redis Streams Producer for Plugin Execution
**What:** Enqueue plugin execution requests to Redis Streams for CrewAI Python workers
**When to use:** When scheduled plugin execution is triggered or user manually runs plugin
**Example:**
```go
// Source: https://redis.io/docs/latest/commands/xadd/ and https://pkg.go.dev/github.com/redis/go-redis/v9
package plugins

import (
    "context"
    "encoding/json"
    "github.com/redis/go-redis/v9"
    "time"
)

type PluginExecutionRequest struct {
    PluginRunID  string                 `json:"plugin_run_id"`  // UUID for tracking
    PluginName   string                 `json:"plugin_name"`
    UserID       uint                   `json:"user_id"`
    UserSettings map[string]interface{} `json:"user_settings"`
    ScheduledAt  time.Time              `json:"scheduled_at"`
}

func EnqueuePluginExecution(ctx context.Context, rdb *redis.Client, req *PluginExecutionRequest) error {
    streamKey := fmt.Sprintf("plugin:execution:%s", req.PluginName)

    // Marshal request to JSON
    payload, err := json.Marshal(req)
    if err != nil {
        return err
    }

    // Add to Redis Stream
    args := &redis.XAddArgs{
        Stream: streamKey,
        Values: map[string]interface{}{
            "payload": string(payload),
        },
    }

    _, err = rdb.XAdd(ctx, args).Result()
    return err
}
```

### Pattern 5: GORM Models with Schema Versioning
**What:** Database models for plugins with explicit schema_version field for compatibility tracking
**When to use:** Initial model creation and all plugin metadata/config persistence
**Example:**
```go
// Source: GORM best practices + project's existing models/briefing.go pattern
package models

import (
    "gorm.io/datatypes"
    "gorm.io/gorm"
)

// Plugin metadata stored in database after discovery
type Plugin struct {
    gorm.Model
    Name          string `gorm:"uniqueIndex;not null"`
    Description   string `gorm:"type:text"`
    Owner         string
    Version       string         `gorm:"not null"` // Plugin implementation version (e.g., "1.2.0")
    SchemaVersion string         `gorm:"not null"` // Metadata schema version (e.g., "v1")
    Capabilities  datatypes.JSON `gorm:"type:jsonb"` // ["briefing", "scheduled"]
    DefaultConfig datatypes.JSON `gorm:"type:jsonb"` // Default settings
    Enabled       bool           `gorm:"default:true"`
}

// Per-user plugin configuration
type UserPluginConfig struct {
    gorm.Model
    UserID   uint           `gorm:"not null;index"`
    PluginID uint           `gorm:"not null;index"`
    Settings datatypes.JSON `gorm:"type:jsonb"` // User-specific settings (validated against schema)
    Enabled  bool           `gorm:"default:false"`

    // Associations
    User   User   `gorm:"constraint:OnDelete:CASCADE;"`
    Plugin Plugin `gorm:"constraint:OnDelete:CASCADE;"`
}

// Plugin execution run tracking
type PluginRun struct {
    gorm.Model
    PluginRunID  string         `gorm:"uniqueIndex;not null"` // UUID for external tracking
    UserID       uint           `gorm:"not null;index"`
    PluginID     uint           `gorm:"not null;index"`
    Status       string         `gorm:"not null;default:'pending';index"` // pending, processing, completed, failed
    Input        datatypes.JSON `gorm:"type:jsonb"` // User settings at execution time
    Output       datatypes.JSON `gorm:"type:jsonb"` // Result from CrewAI
    ErrorMessage string         `gorm:"type:text"`
    StartedAt    *time.Time
    CompletedAt  *time.Time

    // Associations
    User   User   `gorm:"constraint:OnDelete:CASCADE;"`
    Plugin Plugin `gorm:"constraint:OnDelete:CASCADE;"`
}
```

### Anti-Patterns to Avoid
- **Hardcoding plugin list:** Always use directory-based discovery, never hardcode plugin names in Go code
- **Skipping schema versioning:** Always include schema_version field in metadata from day one to support future migrations
- **Storing raw YAML in database:** Parse YAML to structs, validate, then store as JSON in JSONB columns for queryability
- **Per-user cron entries:** Use database-backed scheduling with periodic query jobs, not O(users × plugins) Redis cron entries
- **Mixing plugin business logic in Go:** Keep Go layer thin (discovery, validation, persistence, queueing), delegate AI/content work to CrewAI Python workers

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| YAML parsing with validation | Custom YAML parser with manual field checking | gopkg.in/yaml.v3 with KnownFields() | Handles edge cases (anchors, multi-doc, type coercion), mature error reporting |
| JSON Schema validation | Custom validation rules per plugin | kaptinlin/jsonschema | Draft 2020-12 compliance, supports $ref/$defs, internationalized errors, battle-tested |
| Plugin directory scanning | Recursive filepath.Walk with manual filtering | filepath.ReadDir + explicit plugin.yaml check | Simpler, more explicit, avoids deep directory traversal surprises |
| Redis Streams producer | Custom Redis protocol implementation | go-redis XAdd or robinjoseph08/redisqueue | Producer idempotency (Redis 8.6+), automatic reconnection, context support |
| Database migrations | Custom SQL schema versioning system | GORM AutoMigrate (dev) + Atlas (prod) | Atlas generates migrations from GORM models, prevents drift, supports rollback |

**Key insight:** Plugin systems have many failure modes (invalid YAML, schema drift, settings validation, execution tracking). Use proven libraries for parsing/validation and focus effort on business logic (plugin scheduling, result handling, UI).

## Common Pitfalls

### Pitfall 1: YAML Unknown Fields Silently Ignored
**What goes wrong:** By default, yaml.v3 Unmarshal silently ignores unknown fields in YAML. A typo like `plugin_version` instead of `version` passes validation but breaks assumptions.
**Why it happens:** Default Unmarshaler behavior is permissive to support forward compatibility
**How to avoid:** Always use `decoder.KnownFields(true)` which returns an error on unknown fields
**Warning signs:** Plugin metadata loads successfully but expected fields are empty/zero values

### Pitfall 2: GORM AutoMigrate Doesn't Track Versions
**What goes wrong:** GORM's AutoMigrate has no concept of migration history. Adding/removing columns works, but renaming columns causes AutoMigrate to ADD new column and KEEP old column (data loss risk).
**Why it happens:** GORM doesn't track "what migrations have run" — it only compares current DB schema to model structs
**How to avoid:** Use AutoMigrate during development only. Switch to Atlas for production (atlas migrate diff --env gorm generates versioned SQL migrations)
**Warning signs:** Production database has orphaned columns, data appears in wrong columns after model field renames

### Pitfall 3: JSON Schema Validation After Database Save
**What goes wrong:** Storing invalid user settings in database, then validating on read. Leads to corrupt plugin configs that can't be executed.
**Why it happens:** Validation logic placed in wrong layer (worker instead of API handler)
**How to avoid:** Validate user settings with jsonschema BEFORE saving to UserPluginConfig.Settings. Return 400 Bad Request on validation failure.
**Warning signs:** PluginRun records with status="failed" and error_message="invalid settings" — should have been caught earlier

### Pitfall 4: Schema Versioning as Afterthought
**What goes wrong:** Adding schema_version field later requires backfilling existing records and handling NULL cases. Template code littered with "if schema_version is NULL, assume v1" checks.
**Why it happens:** "We only have one version now, we'll add versioning when we need it"
**How to avoid:** Include schema_version field in Plugin model from day one. Default to "v1" explicitly in LoadPluginMetadata if missing from YAML.
**Warning signs:** Migration to add schema_version column with NOT NULL constraint fails because existing rows are NULL

### Pitfall 5: Redis Streams Without Consumer Groups
**What goes wrong:** Using XREAD instead of XREADGROUP means messages aren't acknowledged. If Python worker crashes mid-execution, message is lost (no retry).
**Why it happens:** XREAD is simpler API, consumer groups add complexity
**How to avoid:** Always use consumer groups with XREADGROUP + XACK pattern, even for single-consumer scenarios. Enables visibility timeout and automatic retry via XPENDING/XCLAIM.
**Warning signs:** Plugin executions occasionally go missing (no PluginRun record created), no way to detect stuck jobs

### Pitfall 6: Tight Coupling Between Go and Python via Struct Versioning
**What goes wrong:** Changing Go PluginExecutionRequest struct fields breaks Python worker compatibility. Deploy Go → Python workers crash.
**Why it happens:** JSON marshaling between Go and Python with no schema contract
**How to avoid:** Version the message format explicitly (e.g., "message_version": "v1" in JSON payload). Python workers check version and reject unknown versions with clear error.
**Warning signs:** Cryptic JSON unmarshal errors in Python worker logs after Go deployment

## Code Examples

Verified patterns from official sources:

### Example Plugin Metadata YAML (plugin.yaml)
```yaml
# Source: Mattermost plugin manifest + semantic versioning best practices
name: daily-news-digest
description: Generates a personalized daily news digest based on user interests
owner: first-sip-team
version: 1.0.0
schema_version: v1

capabilities:
  - briefing
  - scheduled

default_config:
  frequency: daily
  preferred_time: "06:00"
  topics:
    - technology
    - business

settings_schema_path: settings.schema.json
```

### Example Settings JSON Schema (settings.schema.json)
```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "frequency": {
      "type": "string",
      "enum": ["daily", "weekly"],
      "default": "daily"
    },
    "preferred_time": {
      "type": "string",
      "pattern": "^([0-1][0-9]|2[0-3]):[0-5][0-9]$",
      "default": "06:00"
    },
    "topics": {
      "type": "array",
      "items": {
        "type": "string",
        "enum": ["technology", "business", "science", "health"]
      },
      "minItems": 1,
      "maxItems": 5,
      "default": ["technology"]
    }
  },
  "required": ["frequency", "preferred_time", "topics"]
}
```

### CrewAI Plugin Structure (daily-news-digest/crew/)
```python
# main.py - Source: CrewAI quickstart patterns
from crewai import Agent, Task, Crew, Process
import json
import sys

def run_news_digest(user_settings: dict) -> dict:
    """Execute news digest crew with user settings"""

    # Define agents (can also load from agents.yaml)
    researcher = Agent(
        role="News Researcher",
        goal=f"Find latest news on topics: {user_settings['topics']}",
        backstory="Expert at discovering relevant news from trusted sources"
    )

    writer = Agent(
        role="Briefing Writer",
        goal="Create concise, engaging news digest",
        backstory="Skilled at summarizing complex topics clearly"
    )

    # Define tasks
    research_task = Task(
        description=f"Research latest news on {user_settings['topics']}",
        agent=researcher,
        expected_output="List of 5-10 relevant news articles with summaries"
    )

    write_task = Task(
        description="Write engaging daily news digest from research",
        agent=writer,
        expected_output="Formatted briefing with headlines and summaries",
        context=[research_task]  # Depends on research_task output
    )

    # Create crew with sequential process
    crew = Crew(
        agents=[researcher, writer],
        tasks=[research_task, write_task],
        process=Process.sequential
    )

    # Execute crew
    result = crew.kickoff(inputs=user_settings)

    return {
        "status": "completed",
        "content": result.raw,  # Or result.json() for structured output
        "tokens_used": result.token_usage
    }

if __name__ == "__main__":
    # Read input from Redis Stream payload
    input_json = sys.argv[1]
    user_settings = json.loads(input_json)

    output = run_news_digest(user_settings)
    print(json.dumps(output))
```

### Database Migration (migrations/008_add_plugin_models.sql)
```sql
-- Source: GORM AutoMigrate output + schema versioning best practice
-- Generate via: atlas migrate diff --env gorm

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
    enabled BOOLEAN NOT NULL DEFAULT true
);

CREATE INDEX idx_plugins_deleted_at ON plugins (deleted_at);

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

CREATE INDEX idx_user_plugin_configs_user_id ON user_plugin_configs (user_id);
CREATE INDEX idx_user_plugin_configs_plugin_id ON user_plugin_configs (plugin_id);
CREATE INDEX idx_user_plugin_configs_deleted_at ON user_plugin_configs (deleted_at);

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

CREATE INDEX idx_plugin_runs_user_id ON plugin_runs (user_id);
CREATE INDEX idx_plugin_runs_plugin_id ON plugin_runs (plugin_id);
CREATE INDEX idx_plugin_runs_status ON plugin_runs (status);
CREATE INDEX idx_plugin_runs_deleted_at ON plugin_runs (deleted_at);
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Go native plugin (.so files) | Out-of-process RPC or directory-based config loading | ~2021 | .so plugins require exact dependency version matching (brittle), community prefers IPC or config-driven plugins |
| JSON Schema Draft 7 | JSON Schema Draft 2020-12 | 2020 (adopted widely by 2024) | New $defs keyword replaces definitions, better $ref handling, improved validation keywords |
| Per-user cron entries in Redis | Database-backed scheduling with periodic queries | ~2024 | Reduces Redis memory from O(users × plugins) to O(scheduled_jobs), easier to audit/modify schedules |
| AutoMigrate in production | Versioned migrations via Atlas/migrate | ~2023 | AutoMigrate doesn't track history or support rollback, Atlas generates migrations from GORM models |
| XREAD for Redis Streams | XREADGROUP with consumer groups | Redis 5.0+ (2018) | Consumer groups enable acknowledged delivery, visibility timeout, automatic retry, dead-letter queue pattern |

**Deprecated/outdated:**
- **go plugin package (.so files):** Requires all dependencies match exactly between app and plugin (same source code build). Too brittle for dynamic plugin systems. Use directory-based metadata loading instead.
- **JSON Schema Draft 4/6:** kaptinlin/jsonschema and Google's jsonschema-go both target Draft 2020-12. Use $defs not definitions.
- **Storing settings as TEXT:** Use JSONB columns for queryability (can index nested fields, validate with CHECK constraints using jsonb_typeof)

## Open Questions

1. **Python environment isolation per plugin**
   - What we know: Each plugin has requirements.txt, but all run in same Python process (CrewAI worker)
   - What's unclear: Should each plugin's crew run in a virtualenv? Or accept shared dependency conflicts?
   - Recommendation: Start with shared environment, document conflicts if they arise. Consider Docker-per-plugin if isolation becomes critical.

2. **Plugin update/reload without restart**
   - What we know: Plugin metadata loaded at startup
   - What's unclear: Should plugin registry support hot reload when plugin.yaml changes?
   - Recommendation: Defer hot reload to later phase. For v1.1, require app restart for plugin metadata changes.

3. **Plugin settings schema evolution**
   - What we know: Schema versioning exists for plugin metadata, but user settings schemas can also change
   - What's unclear: How to handle user with settings from v1 schema when plugin upgrades to v2 schema?
   - Recommendation: Store schema version with UserPluginConfig.Settings. Validate against plugin's current schema, but support "migration functions" per plugin to upgrade old settings.

4. **Redis Stream consumer architecture**
   - What we know: Go produces to Redis Streams, Python consumes
   - What's unclear: One Python worker per plugin type, or single worker handling all plugins?
   - Recommendation: Start with one stream per plugin (plugin:execution:daily-news-digest) with dedicated consumer group. Allows independent scaling per plugin.

## Sources

### Primary (HIGH confidence)
- [gopkg.in/yaml.v3 - Go Packages](https://pkg.go.dev/gopkg.in/yaml.v3) - YAML parsing API, struct tags, KnownFields validation
- [kaptinlin/jsonschema - GitHub](https://github.com/kaptinlin/jsonschema) - JSON Schema Draft 2020-12 validator, zero-copy validation
- [GORM Migration Documentation](https://gorm.io/docs/migration.html) - AutoMigrate capabilities and limitations
- [Redis Streams Documentation](https://redis.io/docs/latest/develop/data-types/streams/) - XADD, XREADGROUP, consumer groups
- [CrewAI Quickstart](https://docs.crewai.com/en/quickstart) - Agent/Task/Crew structure, inputs/outputs
- [go-redis v9 API](https://pkg.go.dev/github.com/redis/go-redis/v9) - XAdd, XReadGroup methods

### Secondary (MEDIUM confidence)
- [robinjoseph08/redisqueue - GitHub](https://github.com/robinjoseph08/redisqueue) - Higher-level Redis Streams queue abstraction (verified via official Redis patterns)
- [Atlas GORM Integration](https://atlasgo.io/guides/orms/gorm/getting-started) - Automatic migration planning from GORM models
- [CrewAI Examples Repository](https://github.com/crewAIInc/crewAI-examples) - Community examples showing YAML-based agent/task configuration
- [Mattermost Plugin Manifest Reference](https://developers.mattermost.com/integrate/plugins/manifest-reference/) - Production plugin metadata schema patterns
- [Google JSON Schema Blog Post](https://opensource.googleblog.com/2026/01/a-json-schema-package-for-go.html) - Context on Google's official JSON Schema package for Go

### Tertiary (LOW confidence - requires validation)
- Plugin discovery patterns from HashiCorp go-plugin (may not apply to config-based plugins)
- Redis Streams idempotent message processing (Redis 8.6+ feature, need to verify current Redis version in project)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All libraries are official/widely-adopted with stable APIs, current versions verified in go.mod
- Architecture: HIGH - Patterns sourced from official docs (YAML, JSON Schema, GORM, Redis) and proven project structure (models, worker, tasks)
- Pitfalls: MEDIUM-HIGH - GORM AutoMigrate limitations documented officially, YAML KnownFields() from official docs, Redis Streams consumer groups from official patterns, plugin coupling issues from community experience
- CrewAI integration: MEDIUM - Official docs confirm agents/tasks/crews structure and YAML config support, but specific input/output patterns need validation with example plugin

**Research date:** 2026-02-14
**Valid until:** ~2026-03-16 (30 days - stack is relatively stable, but CrewAI is fast-moving and may introduce breaking changes)
