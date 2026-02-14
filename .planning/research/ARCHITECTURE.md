# Architecture Research: v1.1 Plugin Integration

**Domain:** Go Web Application Plugin Architecture with CrewAI Integration
**Researched:** 2026-02-13
**Confidence:** MEDIUM-HIGH

## Integration Architecture Overview

The v1.1 plugin architecture integrates with the existing Go/Gin/GORM/Asynq stack by adding a plugin framework layer, a CrewAI Python sidecar, and per-user scheduling capabilities. The architecture preserves existing patterns while extending them to support pluggable briefing types.

```
┌─────────────────────────────────────────────────────────────────────┐
│                         User Browser                                 │
│  HTMX 2.0 + Tailwind CSS + Liquid Glass Design System              │
└────────────────────────────┬────────────────────────────────────────┘
                             │ HTTP
┌────────────────────────────▼────────────────────────────────────────┐
│                    Gin HTTP Server (EXISTING)                        │
│  ┌────────────────────┐  ┌────────────────────┐  ┌────────────────┐│
│  │ Auth Handlers      │  │ Dashboard Handler  │  │ Settings       ││
│  │ (EXISTING)         │  │ (MODIFIED - tiles) │  │ (NEW)          ││
│  └────────────────────┘  └────────────────────┘  └────────────────┘│
│                                  │                       │           │
│  ┌──────────────────────────────▼───────────────────────▼─────────┐ │
│  │            Plugin Manager Service (NEW)                         │ │
│  │  - Plugin discovery & lifecycle                                 │ │
│  │  - User plugin configuration                                    │ │
│  │  - Plugin metadata validation                                   │ │
│  └──────────────────────────────┬──────────────────────────────────┘ │
└─────────────────────────────────┼───────────────────────────────────┘
                                  │
┌─────────────────────────────────▼───────────────────────────────────┐
│                   Database Layer (GORM + Postgres)                   │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐              │
│  │ User         │  │ Plugin       │  │ UserPlugin   │              │
│  │ (MODIFIED)   │  │ (NEW)        │  │ Config (NEW) │              │
│  └──────────────┘  └──────────────┘  └──────────────┘              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐              │
│  │ Briefing     │  │ AccountTier  │  │ PluginRun    │              │
│  │ (MODIFIED)   │  │ (NEW)        │  │ (NEW)        │              │
│  └──────────────┘  └──────────────┘  └──────────────┘              │
└─────────────────────────────────┬───────────────────────────────────┘
                                  │
┌─────────────────────────────────▼───────────────────────────────────┐
│              Asynq Scheduler + Worker (MODIFIED)                     │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │  Per-User Scheduler (every minute)                          │    │
│  │  - Query users due for briefing                             │    │
│  │  - Check enabled plugins per user                           │    │
│  │  - Enqueue plugin execution tasks                           │    │
│  └───────────────────────┬─────────────────────────────────────┘    │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │  Plugin Execution Handlers (NEW)                            │    │
│  │  - plugin:execute task handler                              │    │
│  │  - Calls CrewAI sidecar per plugin                          │    │
│  │  - Stores results in Briefing table                         │    │
│  └───────────────────────┬─────────────────────────────────────┘    │
└─────────────────────────┼───────────────────────────────────────────┘
                          │ HTTP (internal)
┌─────────────────────────▼───────────────────────────────────────────┐
│                   CrewAI Sidecar (FastAPI + Python)                  │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │  FastAPI Endpoints                                          │    │
│  │  POST /plugins/{plugin_id}/execute                          │    │
│  │  GET  /plugins/{plugin_id}/health                           │    │
│  └───────────────────────┬─────────────────────────────────────┘    │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │  CrewAI Plugin Executors                                    │    │
│  │  - Plugin 1: Weather Briefing (agents + tasks)              │    │
│  │  - Plugin 2: News Briefing (agents + tasks)                 │    │
│  │  - Plugin 3: Calendar Briefing (agents + tasks)             │    │
│  └───────────────────────┬─────────────────────────────────────┘    │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │  CrewAI Core                                                │    │
│  │  - Agent orchestration                                      │    │
│  │  - Task execution                                           │    │
│  │  - External API calls                                       │    │
│  └─────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────┘
```

## New Packages Required

### 1. `/internal/plugins` (NEW)

Core plugin framework implementation.

```
internal/plugins/
├── manager.go           # Plugin discovery, lifecycle, registration
├── registry.go          # In-memory registry of loaded plugins
├── metadata.go          # Plugin metadata parsing (YAML)
├── executor.go          # Interface to CrewAI sidecar
├── validator.go         # Plugin validation logic
├── loader.go            # Plugin loading from filesystem
└── crewai_client.go     # HTTP client for CrewAI sidecar
```

**Responsibilities:**
- Discover plugins from `/plugins` directory at startup
- Parse plugin metadata (YAML files)
- Validate plugin configuration schemas
- Execute plugins via CrewAI sidecar HTTP calls
- Handle plugin execution errors and retries
- Manage plugin lifecycle (load, reload, disable)

### 2. `/internal/tiers` (NEW)

Account tier and constraint enforcement.

```
internal/tiers/
├── models.go            # AccountTier model (GORM)
├── service.go           # Tier constraint checking
├── middleware.go        # HTTP middleware for tier enforcement
└── constants.go         # Tier definitions (free, pro, enterprise)
```

**Responsibilities:**
- Define tier limits (plugin count, execution frequency)
- Check user tier before plugin operations
- Enforce constraints at API and worker levels
- Provide tier upgrade information to UI

### 3. `/internal/scheduling` (NEW)

Per-user scheduling logic extracted from worker package.

```
internal/scheduling/
├── scheduler.go         # Per-user schedule evaluation
├── matcher.go           # Cron expression matching per timezone
└── queue.go             # Task queueing for due plugins
```

**Responsibilities:**
- Evaluate which users have plugins due for execution
- Handle per-user timezones and schedules
- Determine next execution time for plugins
- Enqueue plugin execution tasks

### 4. `/internal/settings` (NEW)

Settings UI handlers and services.

```
internal/settings/
├── handlers.go          # HTTP handlers for settings pages
├── templates.templ      # Templ components for settings UI
├── service.go           # Settings business logic
└── validation.go        # Settings validation
```

**Responsibilities:**
- Render plugin management dashboard
- Handle plugin enable/disable actions
- Manage per-plugin user configuration
- Validate user settings against plugin schemas

### 5. `/plugins` (NEW - not in internal)

Plugin definitions directory (outside Go code).

```
plugins/
├── weather-briefing/
│   ├── plugin.yaml      # Metadata, settings schema, version
│   └── crewai/          # CrewAI Python code
│       ├── agents.py    # CrewAI agents
│       ├── tasks.py     # CrewAI tasks
│       └── main.py      # Plugin entrypoint
├── news-briefing/
│   ├── plugin.yaml
│   └── crewai/
│       ├── agents.py
│       ├── tasks.py
│       └── main.py
└── calendar-briefing/
    ├── plugin.yaml
    └── crewai/
        ├── agents.py
        ├── tasks.py
        └── main.py
```

**Responsibilities:**
- Plugin metadata declarations
- CrewAI workflow definitions (Python code)
- Plugin versioning and dependencies
- Settings schema definitions

### 6. `/cmd/crewai-sidecar` (NEW)

CrewAI sidecar entrypoint (separate process).

```
cmd/crewai-sidecar/
├── main.py              # FastAPI app entrypoint
├── requirements.txt     # Python dependencies (crewai, fastapi)
└── config.py            # Configuration loading
```

**Responsibilities:**
- Start FastAPI server
- Load plugin executors dynamically
- Expose HTTP endpoints for plugin execution
- Handle CrewAI orchestration

## Database Schema Changes

### New Models

```go
// Plugin represents a registered plugin in the system
type Plugin struct {
    gorm.Model
    Slug         string `gorm:"uniqueIndex;not null"`        // e.g., "weather-briefing"
    Name         string `gorm:"not null"`                     // Display name
    Description  string
    Version      string `gorm:"not null"`                     // Semantic version
    Enabled      bool   `gorm:"not null;default:true"`        // System-wide enable
    SettingsSchema datatypes.JSON `gorm:"type:jsonb"`        // JSON Schema for settings
    Metadata     datatypes.JSON `gorm:"type:jsonb"`          // Full plugin.yaml
}

// UserPluginConfig represents a user's configuration for a specific plugin
type UserPluginConfig struct {
    gorm.Model
    UserID    uint           `gorm:"not null;index:idx_user_plugin,unique"`
    PluginID  uint           `gorm:"not null;index:idx_user_plugin,unique"`
    User      User           `gorm:"constraint:OnDelete:CASCADE;"`
    Plugin    Plugin         `gorm:"constraint:OnDelete:CASCADE;"`
    Enabled   bool           `gorm:"not null;default:false"`
    Schedule  string         `gorm:"not null;default:'0 6 * * *'"`  // Cron expression
    Timezone  string         `gorm:"not null;default:'UTC'"`
    Settings  datatypes.JSON `gorm:"type:jsonb"`                    // Plugin-specific settings
}

// PluginRun tracks plugin execution history
type PluginRun struct {
    gorm.Model
    UserID       uint      `gorm:"not null;index"`
    PluginID     uint      `gorm:"not null;index"`
    BriefingID   *uint     `gorm:"index"`                  // NULL if failed
    User         User      `gorm:"constraint:OnDelete:CASCADE;"`
    Plugin       Plugin    `gorm:"constraint:OnDelete:CASCADE;"`
    Briefing     *Briefing `gorm:"constraint:OnDelete:SET NULL;"`
    Status       string    `gorm:"not null;index"`         // pending, processing, completed, failed
    ErrorMessage string    `gorm:"type:text"`
    StartedAt    *time.Time
    CompletedAt  *time.Time
    Duration     int       `gorm:"default:0"`              // Milliseconds
}

// AccountTier defines user account tier constraints
type AccountTier struct {
    gorm.Model
    Name             string `gorm:"uniqueIndex;not null"`  // "free", "pro", "enterprise"
    MaxPlugins       int    `gorm:"not null"`              // Max enabled plugins
    MaxDailyRuns     int    `gorm:"not null"`              // Max plugin executions per day
    MinScheduleInterval int `gorm:"not null"`             // Minimum minutes between runs
}
```

### Modified Models

```go
// User (MODIFIED - add tier relationship)
type User struct {
    gorm.Model
    Email                string     `gorm:"uniqueIndex:idx_users_email_not_deleted,where:deleted_at IS NULL;not null"`
    Name                 string     `gorm:"not null;default:''"`
    Timezone             string     `gorm:"not null;default:'UTC'"`
    PreferredBriefingTime string    `gorm:"not null;default:'06:00'"` // DEPRECATED in v1.1
    Role                 string     `gorm:"not null;default:'user'"`
    AccountTierID        uint       `gorm:"not null;default:1"`  // NEW - FK to AccountTier
    AccountTier          AccountTier `gorm:"constraint:OnDelete:RESTRICT;"` // NEW
    LastLoginAt          *time.Time
    LastBriefingAt       *time.Time

    // Associations
    AuthIdentities   []AuthIdentity     `gorm:"constraint:OnDelete:CASCADE;"`
    Briefings        []Briefing         `gorm:"constraint:OnDelete:CASCADE;"`
    PluginConfigs    []UserPluginConfig `gorm:"constraint:OnDelete:CASCADE;"` // NEW
}

// Briefing (MODIFIED - add plugin relationship)
type Briefing struct {
    gorm.Model
    UserID       uint           `gorm:"not null;index"`
    PluginID     *uint          `gorm:"index"`  // NEW - NULL for legacy briefings
    User         User           `gorm:"constraint:OnDelete:CASCADE;"`
    Plugin       *Plugin        `gorm:"constraint:OnDelete:SET NULL;"` // NEW
    Content      datatypes.JSON `gorm:"type:jsonb"`
    Status       string         `gorm:"not null;default:'pending';index"`
    ErrorMessage string         `gorm:"column:error_message;type:text"`
    GeneratedAt  *time.Time
    ReadAt       *time.Time
}
```

### Migration Strategy

**Phase 1: Additive Migrations (v1.1.0)**
```go
// Add new tables (Plugin, UserPluginConfig, PluginRun, AccountTier)
// Add nullable PluginID column to Briefing
// Add AccountTierID column to User with default value
// Seed default AccountTier records
```

**Phase 2: Data Migration (v1.1.1)**
```go
// Create default plugins from /plugins directory
// Migrate existing User.PreferredBriefingTime to UserPluginConfig records
// Mark existing Briefings with PluginID = NULL (legacy)
```

**Phase 3: Cleanup (v1.2.0 - future)**
```go
// Remove User.PreferredBriefingTime column (deprecated)
// Add NOT NULL constraint to Briefing.PluginID after grace period
```

## Modified Existing Code

### 1. `/internal/worker/scheduler.go` (MAJOR MODIFICATION)

**Before:** Single cron schedule triggers `TaskScheduledBriefingGeneration`

**After:** Per-minute scheduler evaluates users and enqueues per-plugin tasks

```go
// OLD: Single cron registration
scheduler.Register(cfg.BriefingSchedule, taskForAllUsers)

// NEW: Per-minute scheduler with per-user evaluation
scheduler.Register("* * * * *", asynq.NewTask(TaskEvaluateSchedules, nil))
```

### 2. `/internal/worker/worker.go` (MODIFIED)

**Remove:**
- `handleScheduledBriefingGeneration` (replaced by scheduling package)
- `handleGenerateBriefing` using n8n webhook

**Add:**
- `handleEvaluateSchedules` - determine which user+plugin combos are due
- `handleExecutePlugin` - call CrewAI sidecar and store result

### 3. `/internal/briefings/handlers.go` (MODIFIED)

**Remove:**
- Manual "Generate Briefing" endpoint (POST `/api/briefings`)

**Modify:**
- GET `/api/briefings/:id/status` - still needed for polling plugin execution
- POST `/api/briefings/:id/read` - still works for plugin-generated briefings

### 4. `/internal/templates/dashboard.templ` (MAJOR MODIFICATION)

**Before:** Single briefing card

**After:** Tile-based grid showing all enabled plugins

```templ
templ DashboardPage(user User, tiles []PluginTile) {
    @Layout("Dashboard - First Sip") {
        <div class="plugin-grid">
            for _, tile := range tiles {
                @PluginTileCard(tile)
            }
        </div>
    }
}

templ PluginTileCard(tile PluginTile) {
    <div class="glass-card plugin-tile">
        <h3>{tile.PluginName}</h3>
        if tile.LatestBriefing != nil {
            @BriefingContent(tile.LatestBriefing)
        } else {
            <p class="empty-state">No briefing yet</p>
        }
        <div class="tile-footer">
            <span class="status">{tile.Status}</span>
            <span class="next-run">Next: {tile.NextRun}</span>
        </div>
    </div>
}
```

### 5. `/cmd/server/main.go` (MINOR MODIFICATION)

**Add:**
- Plugin manager initialization
- Plugin discovery at startup
- CrewAI sidecar health check (optional)

```go
// After database migrations
pluginManager := plugins.NewManager(db, cfg.PluginsDir)
if err := pluginManager.DiscoverAndRegister(); err != nil {
    log.Fatalf("Failed to discover plugins: %v", err)
}

// Add to Gin context for handlers
r.Use(func(c *gin.Context) {
    c.Set("plugin_manager", pluginManager)
    c.Next()
})
```

## CrewAI Sidecar Integration Pattern

### Communication: HTTP (Internal Network)

**Why HTTP over gRPC:**
- Simpler deployment (no protobuf compilation)
- Easier debugging with curl/Postman
- FastAPI has excellent HTTP support out of box
- No cross-language schema sync issues
- Sufficient performance for briefing generation (not high-throughput)

**Trade-offs:**
- Lower performance than gRPC (acceptable for async job use case)
- No streaming support (not needed for briefings)
- Manual JSON schema validation (vs protobuf type safety)

### API Contract

```python
# FastAPI endpoints in CrewAI sidecar

@app.post("/plugins/{plugin_slug}/execute")
async def execute_plugin(
    plugin_slug: str,
    request: PluginExecutionRequest
) -> PluginExecutionResponse:
    """
    Execute a plugin's CrewAI workflow.

    Request body:
    {
        "user_id": 123,
        "settings": {
            "location": "San Francisco",
            "include_forecast": true
        }
    }

    Response:
    {
        "status": "success",
        "content": {
            "weather": {...},
            "forecast": [...]
        },
        "metadata": {
            "execution_time_ms": 3421,
            "crew_agents_used": 2
        }
    }
    """
    pass

@app.get("/plugins/{plugin_slug}/health")
async def plugin_health(plugin_slug: str) -> dict:
    """Check if plugin is loaded and operational."""
    pass
```

### Go Client Implementation

```go
// internal/plugins/crewai_client.go

type CrewAIClient struct {
    baseURL    string
    httpClient *http.Client
}

type ExecutionRequest struct {
    UserID   uint                   `json:"user_id"`
    Settings map[string]interface{} `json:"settings"`
}

type ExecutionResponse struct {
    Status   string                 `json:"status"`
    Content  map[string]interface{} `json:"content"`
    Metadata map[string]interface{} `json:"metadata"`
    Error    string                 `json:"error,omitempty"`
}

func (c *CrewAIClient) ExecutePlugin(ctx context.Context, pluginSlug string, req ExecutionRequest) (*ExecutionResponse, error) {
    url := fmt.Sprintf("%s/plugins/%s/execute", c.baseURL, pluginSlug)

    jsonData, _ := json.Marshal(req)
    httpReq, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
    httpReq.Header.Set("Content-Type", "application/json")

    resp, err := c.httpClient.Do(httpReq)
    if err != nil {
        return nil, fmt.Errorf("crewai request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("crewai returned %d: %s", resp.StatusCode, string(body))
    }

    var result ExecutionResponse
    json.NewDecoder(resp.Body).Decode(&result)
    return &result, nil
}
```

### Deployment Patterns

**Development (Docker Compose):**
```yaml
services:
  app:
    build: .
    environment:
      CREWAI_SIDECAR_URL: http://crewai:8000
    depends_on:
      - crewai

  crewai:
    build: ./cmd/crewai-sidecar
    ports:
      - "8000:8000"
    volumes:
      - ./plugins:/app/plugins
    environment:
      PLUGINS_DIR: /app/plugins
```

**Production (Kubernetes):**
```yaml
# Sidecar container pattern (shared pod)
apiVersion: apps/v1
kind: Deployment
metadata:
  name: first-sip-worker
spec:
  template:
    spec:
      containers:
      - name: worker
        image: first-sip-worker:latest
        env:
        - name: CREWAI_SIDECAR_URL
          value: "http://localhost:8000"
      - name: crewai-sidecar
        image: first-sip-crewai:latest
        ports:
        - containerPort: 8000
```

**Why Sidecar Pattern:**
- Worker and CrewAI always deployed together
- Localhost communication (no network latency)
- Simplified service discovery
- Scales together (1:1 ratio)
- No separate CrewAI service to manage

## Per-User Scheduling Architecture

### Current Scheduler (v1.0)
```go
// Single global cron
scheduler.Register("0 6 * * *", taskForAllUsers)

// Task handler loops users
func handleScheduledBriefingGeneration() {
    users := db.Find(&users)
    for _, user := range users {
        // Generate for each user
    }
}
```

**Problems:**
- All users briefed at same time
- No timezone support
- No per-user schedule customization
- Single point of failure

### New Scheduler (v1.1)

**Pattern:** Periodic evaluation instead of cron-based triggering

```go
// Run every minute
scheduler.Register("* * * * *", asynq.NewTask(TaskEvaluateSchedules, nil))

func handleEvaluateSchedules(ctx context.Context, task *asynq.Task) error {
    now := time.Now()

    // Query users with enabled plugins that are due
    var configs []UserPluginConfig
    db.Preload("User").Preload("Plugin").
       Where("enabled = ?", true).
       Find(&configs)

    for _, config := range configs {
        // Check if due based on config.Schedule and config.Timezone
        if scheduling.IsDue(config, now) {
            // Enqueue plugin execution
            worker.EnqueuePluginExecution(config.UserID, config.PluginID)
        }
    }

    return nil
}
```

**Schedule Evaluation Logic:**

```go
// internal/scheduling/matcher.go

func IsDue(config UserPluginConfig, now time.Time) bool {
    // Parse user's timezone
    location, err := time.LoadLocation(config.Timezone)
    if err != nil {
        location = time.UTC
    }

    // Convert now to user's timezone
    userNow := now.In(location)

    // Parse cron expression
    schedule, err := cron.ParseStandard(config.Schedule)
    if err != nil {
        return false
    }

    // Get next scheduled time after last run
    lastRun := getLastPluginRun(config.UserID, config.PluginID)
    nextScheduled := schedule.Next(lastRun)

    // Due if current time >= next scheduled time
    return userNow.After(nextScheduled) || userNow.Equal(nextScheduled)
}
```

**Benefits:**
- Per-user timezone support
- Per-plugin schedules
- Graceful degradation (missed schedules caught on next tick)
- Easy to add constraints (tier-based limits)

**Trade-offs:**
- Higher database query frequency (every minute vs daily)
- Need index on `user_plugin_configs(enabled, user_id, plugin_id)`
- Potential duplicate task prevention needed

**Optimization: Redis Cache**
```go
// Cache last run times in Redis to avoid DB queries every minute
key := fmt.Sprintf("plugin_run:last:%d:%d", userID, pluginID)
lastRun := redisClient.Get(key)
if lastRun == "" {
    // Fetch from DB
    lastRun = getLastPluginRunFromDB(userID, pluginID)
    redisClient.SetEx(key, lastRun, 1*time.Hour)
}
```

## Data Flow: Plugin Execution

### End-to-End Flow

```
1. Scheduler (every minute)
   → handleEvaluateSchedules
   → Query enabled UserPluginConfigs
   → Check if due (IsDue function)
   → Enqueue plugin:execute task

2. Asynq Worker
   → Dequeue plugin:execute task
   → Create PluginRun record (status=pending)
   → Call internal/plugins/executor.go

3. Plugin Executor
   → Load plugin metadata
   → Validate user settings against schema
   → Call CrewAI sidecar HTTP endpoint

4. CrewAI Sidecar (Python)
   → Load plugin's agents.py and tasks.py
   → Execute CrewAI workflow
   → Return JSON result

5. Plugin Executor (Go)
   → Parse CrewAI response
   → Create Briefing record (status=completed, content=JSON)
   → Update PluginRun (status=completed, briefing_id=X)
   → Update User.LastBriefingAt

6. User Dashboard (next pageload)
   → Query latest briefings per plugin
   → Render plugin tiles
   → Display briefing content
```

### Error Handling

```go
func (e *PluginExecutor) Execute(ctx context.Context, userID, pluginID uint) error {
    // Create PluginRun record
    run := &PluginRun{
        UserID:   userID,
        PluginID: pluginID,
        Status:   "pending",
    }
    db.Create(run)

    defer func() {
        if r := recover(); r != nil {
            db.Model(run).Updates(map[string]interface{}{
                "status":        "failed",
                "error_message": fmt.Sprintf("panic: %v", r),
                "completed_at":  time.Now(),
            })
        }
    }()

    // Update to processing
    run.Status = "processing"
    run.StartedAt = time.Now()
    db.Save(run)

    // Execute plugin
    result, err := e.crewaiClient.ExecutePlugin(ctx, pluginSlug, request)
    if err != nil {
        db.Model(run).Updates(map[string]interface{}{
            "status":        "failed",
            "error_message": err.Error(),
            "completed_at":  time.Now(),
        })
        return err
    }

    // Create briefing
    briefing := &Briefing{
        UserID:      userID,
        PluginID:    &pluginID,
        Content:     result.Content,
        Status:      "completed",
        GeneratedAt: time.Now(),
    }
    db.Create(briefing)

    // Update run as completed
    db.Model(run).Updates(map[string]interface{}{
        "status":       "completed",
        "briefing_id":  briefing.ID,
        "completed_at": time.Now(),
        "duration":     time.Since(*run.StartedAt).Milliseconds(),
    })

    return nil
}
```

## Plugin Metadata Format

### plugin.yaml Structure

```yaml
# plugins/weather-briefing/plugin.yaml

slug: weather-briefing
name: Weather Briefing
description: Daily weather forecast and conditions for your location
version: 1.0.0
author: First Sip Team
category: lifestyle

# CrewAI entrypoint
crewai:
  entrypoint: crewai/main.py
  python_version: "3.11"

# User-configurable settings
settings_schema:
  type: object
  properties:
    location:
      type: string
      title: Location
      description: City or zip code for weather data
      default: ""
    include_forecast:
      type: boolean
      title: Include 5-day forecast
      default: true
    units:
      type: string
      title: Temperature units
      enum: [fahrenheit, celsius]
      default: fahrenheit
  required: [location]

# Default schedule (user can override)
default_schedule:
  cron: "0 6 * * *"
  timezone: UTC

# Tier requirements
requires:
  min_tier: free
  capabilities: []

# Display configuration
tile:
  icon: weather-icon
  color: "#4A90E2"
  default_visible: true
```

### Validation in Go

```go
// internal/plugins/validator.go

type PluginMetadata struct {
    Slug        string            `yaml:"slug"`
    Name        string            `yaml:"name"`
    Description string            `yaml:"description"`
    Version     string            `yaml:"version"`
    CrewAI      CrewAIConfig      `yaml:"crewai"`
    SettingsSchema json.RawMessage `yaml:"settings_schema"`
    DefaultSchedule ScheduleConfig `yaml:"default_schedule"`
    Requires    Requirements      `yaml:"requires"`
}

func (v *Validator) ValidatePlugin(path string) error {
    // Parse plugin.yaml
    data, err := os.ReadFile(filepath.Join(path, "plugin.yaml"))
    if err != nil {
        return fmt.Errorf("missing plugin.yaml: %w", err)
    }

    var metadata PluginMetadata
    if err := yaml.Unmarshal(data, &metadata); err != nil {
        return fmt.Errorf("invalid yaml: %w", err)
    }

    // Validate slug format
    if !regexp.MustCompile(`^[a-z0-9-]+$`).MatchString(metadata.Slug) {
        return fmt.Errorf("invalid slug format")
    }

    // Validate version (semantic versioning)
    if !semver.IsValid("v" + metadata.Version) {
        return fmt.Errorf("invalid version")
    }

    // Validate settings schema is valid JSON Schema
    var schema map[string]interface{}
    if err := json.Unmarshal(metadata.SettingsSchema, &schema); err != nil {
        return fmt.Errorf("invalid settings schema: %w", err)
    }

    // Check CrewAI entrypoint exists
    entrypointPath := filepath.Join(path, metadata.CrewAI.Entrypoint)
    if _, err := os.Stat(entrypointPath); err != nil {
        return fmt.Errorf("crewai entrypoint not found: %s", entrypointPath)
    }

    return nil
}
```

## Build Order & Dependencies

### Dependency Graph

```
Phase 1: Database Schema
  ├─ Add new models (Plugin, UserPluginConfig, PluginRun, AccountTier)
  ├─ Modify existing models (User, Briefing)
  └─ Run migrations

Phase 2: Plugin Framework (Core)
  ├─ Depends on: Phase 1
  ├─ Create /internal/plugins package
  ├─ Implement metadata parsing
  ├─ Implement plugin registry
  └─ Add plugin discovery to main.go

Phase 3: CrewAI Sidecar
  ├─ Depends on: Phase 2 (for plugin metadata format)
  ├─ Create /cmd/crewai-sidecar
  ├─ Implement FastAPI endpoints
  ├─ Create example plugin (weather-briefing)
  └─ Add to Docker Compose

Phase 4: Plugin Executor
  ├─ Depends on: Phase 2, Phase 3
  ├─ Implement CrewAI client in Go
  ├─ Create plugin executor service
  └─ Add worker task handler (plugin:execute)

Phase 5: Per-User Scheduling
  ├─ Depends on: Phase 1, Phase 4
  ├─ Create /internal/scheduling package
  ├─ Modify worker scheduler (per-minute evaluation)
  ├─ Implement IsDue logic with timezone support
  └─ Remove old global scheduler

Phase 6: Dashboard UI (Tiles)
  ├─ Depends on: Phase 4
  ├─ Modify dashboard.templ (tile-based layout)
  ├─ Create PluginTile components
  ├─ Update dashboard handler (query per plugin)
  └─ Remove manual "Generate Briefing" button

Phase 7: Settings UI
  ├─ Depends on: Phase 2, Phase 6
  ├─ Create /internal/settings package
  ├─ Implement plugin management page
  ├─ Add plugin enable/disable handlers
  ├─ Dynamic settings form generation from schema
  └─ Add settings routes to main.go

Phase 8: Account Tiers (Scaffold)
  ├─ Depends on: Phase 1
  ├─ Create /internal/tiers package
  ├─ Seed default tiers (free, pro)
  ├─ Add tier checking to plugin operations
  └─ Add tier enforcement middleware
```

### Suggested Build Order

**Iteration 1: Plugin Foundation (1-2 days)**
- Phase 1: Database Schema
- Phase 2: Plugin Framework (Core)
- Phase 3: CrewAI Sidecar (minimal FastAPI stub)

**Goal:** Load plugins from /plugins directory, parse metadata, register in DB

**Iteration 2: Plugin Execution (2-3 days)**
- Phase 4: Plugin Executor
- Implement one working plugin end-to-end (weather-briefing)

**Goal:** Manual execution of a plugin via API call (bypass scheduler)

**Iteration 3: Scheduling (1-2 days)**
- Phase 5: Per-User Scheduling
- Migrate existing users to new model

**Goal:** Automated plugin execution based on per-user schedules

**Iteration 4: UI (2-3 days)**
- Phase 6: Dashboard UI (Tiles)
- Phase 7: Settings UI

**Goal:** Users can enable plugins and see results in tiles

**Iteration 5: Tiers (1 day)**
- Phase 8: Account Tiers (Scaffold)

**Goal:** Free tier limits enforced, upgrade messaging in UI

**Total estimated time:** 7-11 days

### Critical Path

```
Database Schema → Plugin Framework → Plugin Executor → Scheduling → Dashboard UI
```

Settings UI and Account Tiers can be built in parallel after Dashboard UI.

## Scaling Considerations

| Concern | v1.1 (100 users) | v1.2 (1K users) | v2.0 (10K users) |
|---------|------------------|-----------------|------------------|
| **Scheduler** | Per-minute DB query OK | Add Redis cache for last run times | Partition evaluation (shard users by ID % 10) |
| **CrewAI Sidecar** | Single sidecar per worker | Same (scales with workers) | Add sidecar pool (1:N worker:sidecar ratio) |
| **Plugin Execution** | Synchronous HTTP calls | Add timeout + circuit breaker | Queue plugin executions separately from evaluation |
| **Database** | UserPluginConfig index on (enabled, user_id) | Composite index on (enabled, user_id, plugin_id) | Read replica for dashboard queries |
| **Plugin Registry** | In-memory registry OK | Same | Add Redis-backed registry for multi-instance |

### Bottleneck Analysis

**First bottleneck:** CrewAI sidecar execution time (2-5s per plugin)
- **Solution:** Increase worker concurrency, add more worker pods
- **When:** 500+ active users with 3+ plugins each = 1500+ executions/day

**Second bottleneck:** Per-minute scheduler query overhead
- **Solution:** Redis cache for last run times, reduce DB round trips
- **When:** 5000+ user+plugin combinations

**Third bottleneck:** Plugin execution failures impact scheduler performance
- **Solution:** Separate Asynq queues (high priority: evaluation, low priority: execution)
- **When:** Failure rate > 10% and plugin count > 1000

## Anti-Patterns

### Anti-Pattern 1: Plugin Code in Go Binary

**What people do:** Implement plugin logic in Go, compile into binary
**Why it's wrong:**
- Forces app rebuild for every plugin change
- No runtime plugin loading
- CrewAI is Python-native, bridging to Go adds complexity
**Do this instead:**
- Plugin logic lives in Python (CrewAI)
- Go binary only orchestrates via HTTP
- Plugins deployed as code in /plugins directory

### Anti-Pattern 2: Synchronous Plugin Execution in HTTP Handler

**What people do:**
```go
func CreateBriefing(c *gin.Context) {
    result := executor.ExecutePlugin(userID, pluginID) // Blocks 5+ seconds
    c.JSON(200, result)
}
```
**Why it's wrong:**
- Request timeout
- No retry on failure
- Can't show progress to user
**Do this instead:**
- Enqueue Asynq task
- Return 202 Accepted immediately
- HTMX polls for status

### Anti-Pattern 3: Shared CrewAI Sidecar Service

**What people do:** Deploy single CrewAI service for all workers
**Why it's wrong:**
- Network latency (workers → service over network)
- Service becomes bottleneck
- Complex service discovery
**Do this instead:**
- Sidecar pattern (CrewAI container in same pod as worker)
- Localhost communication
- Scales linearly with workers

### Anti-Pattern 4: Plugin Settings in Code

**What people do:** Hardcode plugin settings in Go structs
**Why it's wrong:**
- Settings not configurable per user
- Adding settings requires code change
- No UI generation possible
**Do this instead:**
- Settings schema in plugin.yaml (JSON Schema)
- Dynamic settings validation in Go
- Auto-generate settings forms from schema

### Anti-Pattern 5: Global Scheduler with User Loop

**What people do:** Keep global cron, add timezone conversion in loop
**Why it's wrong:**
- All users processed at once (load spike)
- No per-user schedule customization
- Single failure point
**Do this instead:**
- Per-minute evaluation scheduler
- Check which users are due
- Spread load naturally across time

## Integration Points

### External Integrations

| Service | Integration Pattern | Location | Notes |
|---------|---------------------|----------|-------|
| **CrewAI APIs** | HTTP from Python sidecar | CrewAI plugin code | Each plugin configures its own API keys |
| **Google OAuth** | Existing via Goth | /internal/auth | No changes needed |
| **Weather APIs** | CrewAI plugin | weather-briefing/crewai/ | Plugin-specific, not app-level |
| **News APIs** | CrewAI plugin | news-briefing/crewai/ | Plugin-specific, not app-level |

### Internal Boundaries

| Boundary | Communication | Notes |
|----------|---------------|-------|
| **Gin Server ↔ Plugin Manager** | Direct function calls | Plugin manager injected via Gin context |
| **Worker ↔ Plugin Executor** | Direct function calls | Executor is internal package |
| **Plugin Executor ↔ CrewAI Sidecar** | HTTP (localhost) | JSON request/response |
| **Scheduler ↔ Database** | GORM | Query UserPluginConfigs every minute |
| **Dashboard ↔ Plugin Registry** | Direct function calls | Registry provides metadata for UI |

### Data Ownership

| Data | Owner | Access Pattern |
|------|-------|----------------|
| **Plugin Metadata** | Plugin Manager | Read-only after discovery |
| **User Plugin Settings** | Settings Service | CRUD via settings UI |
| **Briefing Content** | Briefing Service | Created by executor, read by dashboard |
| **Execution History** | Plugin Executor | PluginRun records for observability |
| **Account Tier Limits** | Tiers Service | Read-only enforcement checks |

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Go plugin architecture | HIGH | Standard patterns, based on existing codebase |
| CrewAI HTTP integration | MEDIUM-HIGH | HTTP is well-understood, CrewAI is Python-native |
| Per-user scheduling | HIGH | Asynq patterns well-established |
| Database schema | HIGH | GORM migrations, standard relational model |
| Sidecar deployment | HIGH | K8s pattern, used in production widely |
| CrewAI internals | MEDIUM | Dependent on CrewAI library stability (not verified) |
| Tile-based UI | HIGH | Standard HTMX patterns, existing design system |

## Sources

Based on:
- Existing codebase analysis (cmd/server/main.go, internal/worker/, internal/models/)
- Go plugin architecture patterns (training data)
- Asynq scheduler best practices (training data)
- FastAPI + Go integration patterns (training data)
- Kubernetes sidecar patterns (training data)
- TODO documents (redesign-system-around-plugin-based-briefing-architecture.md)

**Unable to verify without external access:**
- Latest CrewAI API patterns (2026)
- CrewAI production deployment best practices
- Specific CrewAI performance characteristics

---
*Architecture research for: First Sip v1.1 Plugin Integration*
*Researched: 2026-02-13*
*Confidence: MEDIUM-HIGH (patterns verified from codebase, CrewAI specifics based on training data)*
