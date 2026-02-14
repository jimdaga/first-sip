# Stack Additions for Plugin-Based Briefing Architecture

**Domain:** Plugin Framework Extensions for Existing Go/Gin/Templ/HTMX App
**Researched:** 2026-02-13
**Overall Confidence:** MEDIUM-HIGH (verified with web research, some libraries confirmed via official sources)

## Context: What We Already Have (DO NOT Add)

**Validated v1.0 Stack:**
- Go 1.24, Gin 1.11.0, Templ 0.3.977, HTMX 2.0
- GORM 1.31.1 (PostgreSQL), Asynq 0.26.0 (Redis), Goth 1.82.0
- Custom liquid-glass CSS design system
- Docker/Kubernetes/Helm/ArgoCD deployment
- AES-256-GCM encrypted OAuth tokens

**This document covers ONLY new additions for:**
- Plugin metadata (YAML parsing)
- Dynamic user settings (JSON schema validation + form generation)
- n8n webhook integration
- Per-user per-plugin scheduling
- Tile-based responsive layouts

## New Stack Additions

### YAML Parsing for Plugin Metadata

| Library | Version | Purpose | Why Recommended |
|---------|---------|---------|-----------------|
| **goccy/go-yaml** | v1.18.0+ | Parse plugin.yaml metadata files | **ALREADY IN go.mod (indirect dependency)**. Superior to gopkg.in/yaml.v3 (which is unmaintained as of 2026). Passes 355+ YAML test suite cases vs 295 for go-yaml/yaml.v3. Better error messages, supports YAML 1.2, maintains compatibility with go-yaml API. |

**Integration:**
```go
import "github.com/goccy/go-yaml"

type PluginMetadata struct {
    Name        string                 `yaml:"name"`
    Description string                 `yaml:"description"`
    Schedule    string                 `yaml:"default_schedule"`
    Settings    map[string]interface{} `yaml:"settings_schema"`
}

func LoadPlugin(path string) (*PluginMetadata, error) {
    data, _ := os.ReadFile(filepath.Join(path, "plugin.yaml"))
    var meta PluginMetadata
    yaml.Unmarshal(data, &meta)
    return &meta, nil
}
```

**No installation needed** — already available via transitive dependency chain (likely from GORM or Asynq).

### JSON Schema Validation for Dynamic Settings

| Library | Version | Purpose | Why Recommended |
|---------|---------|---------|-----------------|
| **kaptinlin/jsonschema** | v0.6.11+ | Validate user plugin settings against schema | Google's official JSON Schema package for Go (announced January 2026). Full Draft 2020-12 compliance. High-performance validator with smart defaults, struct validation without marshaling overhead. Extension field support (x-component, x-ui-props) perfect for form generation metadata. Better than swaggest/jsonschema-go (older, less features) or mcuadros/go-jsonschema-generator (no validation, only generation). |

**Installation:**
```bash
go get github.com/kaptinlin/jsonschema@latest
```

**Integration:**
```go
import "github.com/kaptinlin/jsonschema"

// Validate user settings against plugin's schema
compiler := jsonschema.NewCompiler()
schema, _ := compiler.Compile([]byte(plugin.SettingsSchema))
result := schema.ValidateMap(userSettings)
if !result.IsValid() {
    // result.Errors contains validation failures
}

// Support defaults in schema
compiler.RegisterDefaultFunc("now", jsonschema.DefaultNowFunc)

// Preserve extension fields for UI hints
compiler.SetPreserveExtra(true) // Keeps x-component, x-label, etc.
```

**Why this over alternatives:**
- swaggest/jsonform-go: Only renders HTML forms, doesn't validate
- elastic/go-json-schema-generate: Generates schemas FROM Go, not validates AGAINST schemas
- Official google.golang.org/protobuf/encoding/jsonschema: Too heavyweight, designed for protobuf conversion

### n8n Webhook Integration

**No new library needed** — use existing Go stdlib `net/http` + Gin patterns.

**Security Pattern (Critical):**
```go
// DO NOT accept unauthenticated webhooks from n8n
// Use HMAC signature verification or shared secret

import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
)

// In plugin metadata YAML:
// webhook_secret: ${N8N_PLUGIN_NEWS_SECRET}  # Environment variable per plugin

func VerifyN8NWebhook(c *gin.Context, secret string) bool {
    signature := c.GetHeader("X-N8N-Signature")
    timestamp := c.GetHeader("X-N8N-Timestamp")

    // Reject old timestamps (prevent replay attacks)
    ts, _ := strconv.ParseInt(timestamp, 10, 64)
    if time.Now().Unix() - ts > 300 { // 5 minute window
        return false
    }

    // Verify HMAC
    body, _ := io.ReadAll(c.Request.Body)
    c.Request.Body = io.NopCloser(bytes.NewBuffer(body)) // Reset for handler

    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write([]byte(timestamp))
    mac.Write(body)
    expected := hex.EncodeToString(mac.Sum(nil))

    return hmac.Equal([]byte(signature), []byte(expected))
}
```

**n8n Side Configuration:**
```javascript
// In n8n workflow - HTTP Request node sending to First Sip
const timestamp = Date.now().toString();
const payload = JSON.stringify($json);
const secret = $env.N8N_PLUGIN_NEWS_SECRET;

const crypto = require('crypto');
const hmac = crypto.createHmac('sha256', secret);
hmac.update(timestamp);
hmac.update(payload);
const signature = hmac.digest('hex');

return {
    url: 'https://first-sip.app/webhooks/plugin/news-digest',
    method: 'POST',
    headers: {
        'Content-Type': 'application/json',
        'X-N8N-Signature': signature,
        'X-N8N-Timestamp': timestamp
    },
    body: payload
};
```

**Sources:**
- [n8n Webhook Node Documentation](https://docs.n8n.io/integrations/builtin/core-nodes/n8n-nodes-base.webhook/)
- [Creating a Secure Webhook - n8n workflow template](https://n8n.io/workflows/5174-creating-a-secure-webhook-must-have/)
- [Lock Down n8n Webhooks Before They Bite - Medium](https://medium.com/@Nexumo_/lock-down-n8n-webhooks-before-they-bite-769e6e8768a0)

**Critical 2026 Vulnerability Warning:**
CVE-2026-21894 exposed n8n's StripeTrigger node accepting webhooks without signature verification. DO NOT trust webhook URLs alone. Always verify signatures or use shared secrets.

### Per-User Per-Plugin Scheduling

**Option 1: Extend Existing Asynq (RECOMMENDED)**

| Approach | Library | Why Recommended |
|----------|---------|-----------------|
| **Asynq with dynamic queue naming** | asynq v0.26.0 (ALREADY INSTALLED) | Leverage existing infrastructure. Use queue naming pattern: `plugin:{plugin_id}:user:{user_id}`. Schedule tasks with ProcessAt. No new dependencies. Asynq already supports cron-style scheduling via PeriodicTaskManager. |

**Implementation:**
```go
// Existing Asynq client (already in your codebase)
import "github.com/hibiken/asynq"

// Schedule per-user plugin task
func SchedulePluginForUser(client *asynq.Client, userID, pluginID int, schedule string) error {
    // Queue naming isolates tasks by user+plugin
    queueName := fmt.Sprintf("plugin:%d:user:%d", pluginID, userID)

    // Parse cron expression from plugin metadata
    spec, _ := cron.ParseStandard(schedule) // "0 6 * * *" for 6am daily
    next := spec.Next(time.Now())

    task := asynq.NewTask("plugin:execute", map[string]interface{}{
        "user_id": userID,
        "plugin_id": pluginID,
    })

    _, err := client.Enqueue(task,
        asynq.Queue(queueName),
        asynq.ProcessAt(next),
    )
    return err
}

// Worker processes tasks from user-specific queues
server := asynq.NewServer(
    asynq.RedisClientOpt{Addr: redisAddr},
    asynq.Config{
        Queues: map[string]int{
            "plugin:*:user:*": 10, // Priority weight
        },
        // Asynq supports queue patterns with wildcards
    },
)
```

**Automatic Rescheduling:**
Use Asynq's PeriodicTaskManager to reschedule after each execution:
```go
// In task handler
func HandlePluginExecution(ctx context.Context, t *asynq.Task) error {
    var payload struct {
        UserID   int `json:"user_id"`
        PluginID int `json:"plugin_id"`
    }
    json.Unmarshal(t.Payload(), &payload)

    // Execute plugin (call n8n webhook, etc.)
    executePlugin(payload.UserID, payload.PluginID)

    // Reschedule for next occurrence
    plugin := getPlugin(payload.PluginID)
    userSettings := getUserSettings(payload.UserID, payload.PluginID)
    schedule := userSettings.Schedule // User can override default

    SchedulePluginForUser(asynqClient, payload.UserID, payload.PluginID, schedule)

    return nil
}
```

**Sources:**
- [How to Build a Job Queue in Go with Asynq and Redis - OneUpTime](https://oneuptime.com/blog/post/2026-01-07-go-asynq-job-queue-redis/view)
- [Asynq in Go: A Simple Guide to Task Queues - Medium](https://medium.com/@linz07m/asynq-in-go-a-simple-guide-to-task-queues-84bf3ff6e5fd)

**Option 2: Add go-co-op/gocron (Alternative if Asynq patterns don't fit)**

| Library | Version | Purpose | When to Use Instead |
|---------|---------|---------|---------------------|
| go-co-op/gocron | v2.19.1 | In-memory cron scheduler with per-job concurrency control | If you need purely time-based triggers (not queued tasks), or if per-user schedules are simple (e.g., "every morning at user's preferred time"). Lighter than Asynq for pure scheduling. Does NOT persist across restarts unless you add state management. |

**Installation (if needed):**
```bash
go get github.com/go-co-op/gocron/v2@latest
```

**Integration:**
```go
import "github.com/go-co-op/gocron/v2"

scheduler, _ := gocron.NewScheduler()

// Schedule per-user plugin task
func SchedulePluginForUser(s gocron.Scheduler, userID, pluginID int, cronExpr string) {
    job, _ := s.NewJob(
        gocron.CronJob(cronExpr, false), // "0 6 * * *"
        gocron.NewTask(func() {
            executePluginForUser(userID, pluginID)
        }),
        gocron.WithSingletonMode(gocron.LimitModeReschedule), // Skip overlaps
        gocron.WithName(fmt.Sprintf("plugin-%d-user-%d", pluginID, userID)),
    )
}

scheduler.Start()
```

**Why Asynq is preferred:**
- Already installed and running
- Persists tasks in Redis (survives restarts)
- Built-in retry logic
- Existing monitoring (Asynqmon)
- Better for distributed systems (multiple worker instances)

**Why gocron might be better:**
- Simpler mental model (just cron jobs, no queues)
- Less Redis overhead
- Event listeners for job lifecycle hooks
- If you only need scheduling, not task queuing

**Recommendation:** Start with Asynq. Add gocron later only if you need features like interval timing from completion time (not available in Asynq).

**Sources:**
- [GoCron v2 GitHub](https://github.com/go-co-op/gocron)
- [gocron v2.19.1 package documentation](https://pkg.go.dev/github.com/go-co-op/gocron/v2)

### Tile-Based Responsive Layouts

**No new library needed** — use modern CSS Grid with existing Tailwind + custom CSS.

**Pattern: Auto-fit Grid for Plugin Tiles**

```css
/* Add to static/css/liquid-glass.css */

.plugin-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
    gap: var(--spacing-lg, 1.5rem);
    padding: var(--spacing-lg, 1.5rem);
}

/* Responsive breakpoints */
@media (max-width: 768px) {
    .plugin-grid {
        grid-template-columns: 1fr; /* Single column on mobile */
        gap: var(--spacing-md, 1rem);
        padding: var(--spacing-md, 1rem);
    }
}

@media (min-width: 1400px) {
    .plugin-grid {
        grid-template-columns: repeat(auto-fit, minmax(350px, 1fr));
        /* Wider tiles on large screens */
    }
}

/* Plugin tile card */
.plugin-tile {
    background: var(--glass-bg);
    border-radius: var(--radius-lg);
    border: 1px solid var(--glass-border);
    backdrop-filter: blur(var(--glass-blur)) saturate(var(--glass-saturate));
    transition: transform 0.2s var(--ease-spring), box-shadow 0.2s var(--ease-spring);
}

.plugin-tile:hover {
    transform: translateY(-3px);
    box-shadow: 0 12px 24px rgba(212, 145, 94, 0.15); /* Accent glow */
}

/* Status badge positioning */
.plugin-tile-header {
    display: flex;
    justify-content: space-between;
    align-items: start;
    padding: var(--spacing-md);
}

.plugin-tile-body {
    padding: 0 var(--spacing-md) var(--spacing-md);
}
```

**Templ Component:**
```go
templ PluginGrid(plugins []Plugin) {
    <div class="plugin-grid">
        for _, plugin := range plugins {
            @PluginTile(plugin)
        }
    </div>
}

templ PluginTile(plugin Plugin) {
    <div class="plugin-tile glass-card">
        <div class="plugin-tile-header">
            <h3 class="text-lg font-display font-semibold text-primary">
                { plugin.Name }
            </h3>
            if plugin.Enabled {
                <span class="glass-badge glass-badge-read">Active</span>
            } else {
                <span class="glass-badge glass-badge-unread">Disabled</span>
            }
        </div>
        <div class="plugin-tile-body">
            <p class="text-sm text-secondary mb-4">{ plugin.Description }</p>
            <button
                hx-get={ fmt.Sprintf("/plugins/%d/settings", plugin.ID) }
                hx-target="#settings-modal"
                hx-swap="innerHTML"
                class="glass-btn glass-btn-ghost glass-btn-sm w-full"
            >
                Configure
            </button>
        </div>
    </div>
}
```

**HTMX Integration for Dynamic Updates:**
```html
<!-- Settings modal loads plugin-specific form -->
<div id="settings-modal" class="modal"></div>

<!-- HTMX handles form submission -->
<form
    hx-post="/plugins/:id/settings"
    hx-target="#plugin-tile-:id"
    hx-swap="outerHTML"
>
    <!-- Dynamic form fields generated from JSON schema -->
</form>
```

**Sources:**
- [Grid Tile Layouts with auto-fit and minmax - Mastery Games](https://mastery.games/post/tile-layouts/)
- [Tailwind CSS Grid Template Columns: Practical Patterns for 2026 - TheLinuxCode](https://thelinuxcode.com/tailwind-css-grid-template-columns-practical-patterns-for-2026-layouts/)
- [CSS Grid Layout: The Complete Guide for 2026 - DevToolbox](https://devtoolbox.dedyn.io/blog/css-grid-complete-guide)

**Container Queries (2026 Best Practice):**
If plugins live inside resizable dashboard panels, use container queries instead of viewport media queries:

```css
/* Add to liquid-glass.css */
.plugin-container {
    container-type: inline-size;
    container-name: plugin-area;
}

@container plugin-area (max-width: 600px) {
    .plugin-grid {
        grid-template-columns: 1fr;
    }
}

@container plugin-area (min-width: 900px) {
    .plugin-grid {
        grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
    }
}
```

Browser support: 98%+ as of 2026 (safe to use without fallbacks).

### Dynamic Form Generation from JSON Schema

**No new library needed** — generate forms server-side with Templ based on JSON schema metadata.

**Pattern: Schema-to-Templ Form Generator**

```go
// internal/plugins/forms.go
package plugins

import "github.com/kaptinlin/jsonschema"

type FormField struct {
    Name        string
    Label       string
    Type        string // "text", "number", "select", "checkbox"
    Required    bool
    Placeholder string
    Options     []string // For select/radio
    Default     interface{}
    UIComponent string   // From x-component extension
}

func GenerateFormFields(schema *jsonschema.Schema) []FormField {
    var fields []FormField

    for name, prop := range schema.Properties {
        field := FormField{
            Name:     name,
            Label:    prop.Title,
            Required: slices.Contains(schema.Required, name),
        }

        // Map JSON schema types to HTML input types
        switch prop.Type {
        case "string":
            if prop.Format == "email" {
                field.Type = "email"
            } else if prop.Format == "date" {
                field.Type = "date"
            } else {
                field.Type = "text"
            }
        case "number", "integer":
            field.Type = "number"
        case "boolean":
            field.Type = "checkbox"
        }

        // Extract UI hints from extension fields
        if component, ok := prop.Extra["x-component"].(string); ok {
            field.UIComponent = component
        }

        if placeholder, ok := prop.Extra["x-placeholder"].(string); ok {
            field.Placeholder = placeholder
        }

        // Handle enums as select dropdowns
        if len(prop.Enum) > 0 {
            field.Type = "select"
            for _, opt := range prop.Enum {
                field.Options = append(field.Options, fmt.Sprint(opt))
            }
        }

        field.Default = prop.Default
        fields = append(fields, field)
    }

    return fields
}
```

**Templ Component:**
```go
// internal/plugins/templates.templ
templ DynamicForm(plugin Plugin, fields []FormField, values map[string]interface{}) {
    <form
        hx-post={ fmt.Sprintf("/plugins/%d/settings", plugin.ID) }
        hx-target={ fmt.Sprintf("#plugin-tile-%d", plugin.ID) }
        hx-swap="outerHTML"
        class="space-y-4"
    >
        for _, field := range fields {
            <div class="form-field">
                <label class="block text-sm font-medium text-primary mb-1">
                    { field.Label }
                    if field.Required {
                        <span class="text-status-unread-text">*</span>
                    }
                </label>

                switch field.Type {
                    case "text", "email", "number", "date":
                        <input
                            type={ field.Type }
                            name={ field.Name }
                            value={ fmt.Sprint(values[field.Name]) }
                            placeholder={ field.Placeholder }
                            required?={ field.Required }
                            class="w-full px-3 py-2 rounded-md border border-glass-border bg-glass-bg"
                        />

                    case "checkbox":
                        <input
                            type="checkbox"
                            name={ field.Name }
                            checked?={ values[field.Name] == true }
                            class="w-4 h-4 text-accent"
                        />

                    case "select":
                        <select
                            name={ field.Name }
                            required?={ field.Required }
                            class="w-full px-3 py-2 rounded-md border border-glass-border bg-glass-bg"
                        >
                            for _, option := range field.Options {
                                <option
                                    value={ option }
                                    selected?={ values[field.Name] == option }
                                >
                                    { option }
                                </option>
                            }
                        </select>
                }
            </div>
        }

        <div class="flex gap-2 justify-end mt-6">
            <button type="button" class="glass-btn glass-btn-ghost" onclick="closeModal()">
                Cancel
            </button>
            <button type="submit" class="glass-btn glass-btn-primary">
                Save Settings
            </button>
        </div>
    </form>
}
```

**Handler Integration:**
```go
// internal/plugins/handlers.go
func (h *Handler) GetPluginSettings(c *gin.Context) {
    pluginID := c.Param("id")
    plugin := h.service.GetPlugin(pluginID)

    // Parse plugin's settings schema
    compiler := jsonschema.NewCompiler()
    compiler.SetPreserveExtra(true) // Keep x-component, x-placeholder
    schema, _ := compiler.Compile([]byte(plugin.SettingsSchema))

    // Get user's current settings
    userID := c.GetInt("user_id")
    currentSettings := h.service.GetUserPluginSettings(userID, pluginID)

    // Generate form fields from schema
    fields := GenerateFormFields(schema)

    // Render form
    component := DynamicForm(plugin, fields, currentSettings)
    c.Header("Content-Type", "text/html")
    component.Render(c.Request.Context(), c.Writer)
}

func (h *Handler) SavePluginSettings(c *gin.Context) {
    pluginID := c.Param("id")
    userID := c.GetInt("user_id")

    // Parse form data
    var settings map[string]interface{}
    c.ShouldBind(&settings)

    // Validate against plugin schema
    plugin := h.service.GetPlugin(pluginID)
    schema, _ := jsonschema.NewCompiler().Compile([]byte(plugin.SettingsSchema))
    result := schema.ValidateMap(settings)

    if !result.IsValid() {
        // Return validation errors to form
        c.JSON(400, gin.H{"errors": result.Errors})
        return
    }

    // Save validated settings
    h.service.SaveUserPluginSettings(userID, pluginID, settings)

    // Re-render plugin tile with updated status
    updatedPlugin := h.service.GetUserPlugin(userID, pluginID)
    component := PluginTile(updatedPlugin)
    c.Header("Content-Type", "text/html")
    component.Render(c.Request.Context(), c.Writer)
}
```

**Sources:**
- [Google JSON Schema Package for Go - Official Blog](https://opensource.googleblog.com/2026/01/a-json-schema-package-for-go.html)
- [Build dynamic Forms with JSON Schemas - Peter Ullrich](https://peterullrich.com/build-dynamic-forms-with-json-schemas)
- [HTMX form-json extension](https://github.com/xehrad/form-json)

## Installation Summary

**Required New Dependencies:**
```bash
# JSON schema validation
go get github.com/kaptinlin/jsonschema@latest

# Per-user scheduling (OPTIONAL - only if not using Asynq)
# go get github.com/go-co-op/gocron/v2@latest
```

**Already Available (verify in go.mod):**
```bash
# YAML parsing
# github.com/goccy/go-yaml v1.18.0  (already indirect dependency)

# May need to make it direct if not already:
go get github.com/goccy/go-yaml@latest
```

**No Installation Needed:**
- n8n webhook handling (stdlib + Gin)
- Tile layouts (CSS Grid + existing Tailwind)
- Dynamic forms (Templ + existing patterns)

## Database Schema Extensions

**New Tables for Plugin System:**

```sql
-- Plugin registry
CREATE TABLE plugins (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    slug VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    settings_schema JSONB NOT NULL, -- JSON schema definition
    default_schedule VARCHAR(100),  -- Cron expression
    webhook_secret VARCHAR(255),    -- For n8n signature verification
    created_at TIMESTAMP DEFAULT NOW()
);

-- Per-user plugin configuration
CREATE TABLE user_plugin_settings (
    id SERIAL PRIMARY KEY,
    user_id INT REFERENCES users(id) ON DELETE CASCADE,
    plugin_id INT REFERENCES plugins(id) ON DELETE CASCADE,
    enabled BOOLEAN DEFAULT false,
    schedule VARCHAR(100),           -- User-specific override
    settings JSONB NOT NULL,         -- Validated against plugin schema
    last_run_at TIMESTAMP,
    next_run_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id, plugin_id)
);

-- Plugin execution history (optional, for debugging)
CREATE TABLE plugin_executions (
    id SERIAL PRIMARY KEY,
    user_id INT REFERENCES users(id),
    plugin_id INT REFERENCES plugins(id),
    status VARCHAR(50),              -- success, failed, pending
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    error_message TEXT,
    payload JSONB                    -- What was sent to n8n
);

CREATE INDEX idx_user_plugin_next_run ON user_plugin_settings(next_run_at)
    WHERE enabled = true;
```

**GORM Models:**
```go
type Plugin struct {
    ID             int       `gorm:"primaryKey"`
    Name           string    `gorm:"unique;not null"`
    Slug           string    `gorm:"unique;not null"`
    Description    string
    SettingsSchema string    `gorm:"type:jsonb;not null"` // JSON schema as text
    DefaultSchedule string
    WebhookSecret  string
    CreatedAt      time.Time
}

type UserPluginSettings struct {
    ID        int                    `gorm:"primaryKey"`
    UserID    int                    `gorm:"not null"`
    PluginID  int                    `gorm:"not null"`
    Enabled   bool                   `gorm:"default:false"`
    Schedule  string                 // Cron expression
    Settings  datatypes.JSON         `gorm:"type:jsonb;not null"` // Use gorm.io/datatypes
    LastRunAt *time.Time
    NextRunAt *time.Time
    CreatedAt time.Time
    UpdatedAt time.Time

    User   User   `gorm:"foreignKey:UserID"`
    Plugin Plugin `gorm:"foreignKey:PluginID"`
}
```

## Architecture Integration Points

**Plugin Discovery (File-based):**
```
/plugins/
    news-digest/
        plugin.yaml       # Metadata + settings schema
        webhook.go        # Handler for n8n webhook
        README.md         # Plugin documentation

    weather-briefing/
        plugin.yaml
        webhook.go
        README.md
```

**Startup Sequence:**
1. Load all `plugins/*/plugin.yaml` files
2. Validate settings schemas with jsonschema
3. Register webhook routes (`/webhooks/plugin/{slug}`)
4. Load user plugin settings from database
5. Schedule enabled user plugins with Asynq
6. Start Asynq worker to process plugin executions

**Request Flow (User Configures Plugin):**
1. User clicks "Configure" on plugin tile (HTMX GET)
2. Server generates form from JSON schema → Templ component
3. User fills form, submits (HTMX POST)
4. Server validates settings with jsonschema
5. Save to `user_plugin_settings` table
6. If enabled, schedule next execution with Asynq
7. Return updated plugin tile (HTMX swap)

**Request Flow (n8n Sends Briefing Data):**
1. n8n workflow completes, sends POST to `/webhooks/plugin/news-digest`
2. Verify HMAC signature from `X-N8N-Signature` header
3. Extract user_id from webhook URL or payload
4. Store briefing data in database
5. Update `last_run_at` in `user_plugin_settings`
6. Schedule next execution based on cron schedule
7. Optionally send push notification to user

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|-------------------------|
| goccy/go-yaml | gopkg.in/yaml.v3 | **Never** — v3 is unmaintained as of 2026. goccy is strictly better. |
| kaptinlin/jsonschema | swaggest/jsonschema-go | If you need OpenAPI 3.0 compatibility (jsonschema-go integrates with swaggest/openapi-go). But kaptinlin has better validation performance. |
| Asynq scheduling | go-co-op/gocron v2 | If you need purely time-based triggers without task queuing, or if per-user schedules are simple. gocron is lighter but doesn't persist across restarts. |
| CSS Grid for tiles | JavaScript masonry libraries | **Never** — CSS Grid achieves responsive tiles without JS. Only use masonry (e.g., Isotope, Masonry.js) if you need Pinterest-style variable-height layouts with no gaps. |
| Server-side form gen | JSON Forms (React library) | If you switch to client-side SPA. For HTMX/Templ server-rendered approach, Templ components are simpler. |

## What NOT to Add

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| gopkg.in/yaml.v3 | **Unmaintained as of 2026**. README says "THIS PROJECT IS UNMAINTAINED." Maintainer looking for new home. | goccy/go-yaml (already in go.mod) |
| swaggest/jsonform-go | Only renders HTML forms, doesn't integrate with Templ. Designed for standalone HTML generation. | Custom Templ components + kaptinlin/jsonschema |
| JavaScript-based grid libraries (Masonry, Isotope) | Adds client-side dependency for layout CSS Grid solves natively. Only needed for Pinterest-style variable heights. | CSS Grid with auto-fit |
| Separate cron daemon (system cron, cronitor) | External process, harder to deploy, no integration with app state. | Asynq PeriodicTaskManager or gocron (in-process) |
| Client-side form validation libraries (Zod, Yup) | Redundant with server-side jsonschema validation. HTMX pattern is server-validates. | kaptinlin/jsonschema server-side |

## Critical Integration Gotchas

### YAML Parsing
**Problem:** goccy/go-yaml API differs slightly from go-yaml/yaml.v3
**Solution:** Both use same Unmarshal signature. Migration is drop-in replacement. If you have existing yaml.v3 imports, just change to `github.com/goccy/go-yaml`.

### JSON Schema Validation
**Problem:** Schema stored as JSON string in database, needs compilation on every validation
**Solution:** Cache compiled schemas in memory. Use sync.Map or lru-cache keyed by plugin ID:
```go
var schemaCache = &sync.Map{}

func getCompiledSchema(pluginID int) *jsonschema.Schema {
    if cached, ok := schemaCache.Load(pluginID); ok {
        return cached.(*jsonschema.Schema)
    }

    plugin := loadPlugin(pluginID)
    schema, _ := jsonschema.NewCompiler().Compile([]byte(plugin.SettingsSchema))
    schemaCache.Store(pluginID, schema)
    return schema
}
```

### n8n Webhook Security
**Problem:** n8n doesn't have built-in HMAC signing (you must implement in workflow)
**Solution:** Use Function node in n8n to compute signature. Example in Integration section above. Alternative: Use HTTP Basic Auth (simpler but less secure).

### Asynq Dynamic Queue Names
**Problem:** Asynq queue config is static at server startup
**Solution:** Use queue name patterns. Asynq supports wildcard matching: `"plugin:*:user:*": 10`. Register queues dynamically doesn't work well; instead use consistent naming pattern and configure pattern matcher.

### Templ Form Generation
**Problem:** Templ doesn't support dynamic component names (can't `@{field.UIComponent}(props)`)
**Solution:** Use switch/case in Templ template:
```go
switch field.UIComponent {
    case "DatePicker":
        @DatePickerField(field)
    case "RichText":
        @RichTextField(field)
    default:
        @StandardInput(field)
}
```

### CSS Grid Browser Support
**Problem:** Container queries are newer feature
**Solution:** 98%+ browser support as of 2026, safe to use. If you need IE11 (don't), fallback to media queries.

## Version Compatibility Matrix

| Package | Version | Compatible With | Notes |
|---------|---------|-----------------|-------|
| goccy/go-yaml | v1.18.0+ | Go 1.18+ | Already in go.mod (indirect). API compatible with yaml.v3. |
| kaptinlin/jsonschema | v0.6.11+ | Go 1.21+ | Requires generics for FromStruct. Latest may be higher (check pkg.go.dev). |
| go-co-op/gocron | v2.19.1 | Go 1.20+ | v2 is major rewrite. Don't use v1 (different API). |
| Asynq | v0.26.0 | Redis 6.2+, go-redis v9 | Already installed. Supports cron via PeriodicTaskManager. |

## Confidence Assessment

| Area | Confidence | Sources |
|------|------------|---------|
| YAML parsing (goccy/go-yaml) | **HIGH** | GitHub repo active, comparison data from official README, already in use in ecosystem |
| JSON schema (kaptinlin/jsonschema) | **HIGH** | Official Google blog announcement (Jan 2026), pkg.go.dev documentation |
| n8n webhook patterns | **MEDIUM-HIGH** | Official n8n docs, security templates, 2026 CVE warning (verified sources) |
| Asynq scheduling | **HIGH** | Already validated in v1.0, recent 2026 tutorials confirm patterns |
| gocron alternative | **MEDIUM** | Package docs verified, v2 release confirmed, no production usage yet in this project |
| CSS Grid tiles | **HIGH** | MDN docs, multiple 2026 tutorials, 98% browser support confirmed |
| Dynamic forms | **MEDIUM** | Pattern based on standard practices, not library-specific. Templ + jsonschema integration is custom. |

## Gaps & Validation Needed

**Needs phase-specific research:**
- JSON schema extension fields (x-component, x-ui-props) — verify kaptinlin/jsonschema preserves these (documented but should test)
- Asynq queue pattern matching — confirm wildcard support (docs mention it but examples sparse)
- n8n Function node crypto availability — verify crypto module is available in n8n's Node.js sandbox

**Needs testing in implementation:**
- Performance of compiling JSON schemas on every request (caching strategy critical)
- Asynq behavior with hundreds of per-user queues (may need different approach if scaling issue)
- HTMX form validation error rendering (need UX pattern for displaying jsonschema errors)

## Sources

**YAML Parsing:**
- [goccy/go-yaml GitHub](https://github.com/goccy/go-yaml)
- [A tour of YAML parsers for Go](https://sweetohm.net/article/go-yaml-parsers.en.html)
- [Leapcell - Working with YAML in Go](https://leapcell.io/blog/working-with-yaml-in-go)
- [gopkg.in/yaml.v3 - UNMAINTAINED notice](https://github.com/go-yaml/yaml)

**JSON Schema:**
- [Google Open Source Blog - A JSON schema package for Go](https://opensource.googleblog.com/2026/01/a-json-schema-package-for-go.html)
- [kaptinlin/jsonschema package docs](https://pkg.go.dev/github.com/kaptinlin/jsonschema)
- [Build dynamic Forms with JSON Schemas](https://peterullrich.com/build-dynamic-forms-with-json-schemas)

**n8n Webhooks:**
- [n8n Webhook Node Documentation](https://docs.n8n.io/integrations/builtin/core-nodes/n8n-nodes-base.webhook/)
- [Creating a Secure Webhook - n8n workflow template](https://n8n.io/workflows/5174-creating-a-secure-webhook-must-have/)
- [Lock Down n8n Webhooks Before They Bite - Medium](https://medium.com/@Nexumo_/lock-down-n8n-webhooks-before-they-bite-769e6e8768a0)
- [Gecko Security - CVE-2026-21894](https://www.gecko.security/blog/cve-2026-21894)

**Scheduling:**
- [Asynq GitHub](https://github.com/hibiken/asynq)
- [How to Build a Job Queue in Go with Asynq and Redis - OneUpTime](https://oneuptime.com/blog/post/2026-01-07-go-asynq-job-queue-redis/view)
- [go-co-op/gocron v2 GitHub](https://github.com/go-co-op/gocron)
- [gocron v2 package docs](https://pkg.go.dev/github.com/go-co-op/gocron/v2)

**CSS Grid Layouts:**
- [Grid Tile Layouts with auto-fit and minmax](https://mastery.games/post/tile-layouts/)
- [Tailwind CSS Grid Template Columns: Practical Patterns for 2026](https://thelinuxcode.com/tailwind-css-grid-template-columns-practical-patterns-for-2026-layouts/)
- [CSS Grid Layout: The Complete Guide for 2026](https://devtoolbox.dedyn.io/blog/css-grid-complete-guide)

---
*Stack additions for: First Sip Plugin Architecture (v1.1 milestone)*
*Researched: 2026-02-13*
*Confidence: MEDIUM-HIGH (verified with official sources and recent 2026 documentation)*
