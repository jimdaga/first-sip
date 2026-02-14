# Pitfalls Research: v1.1 Plugin Architecture

**Domain:** Adding plugin framework + CrewAI sidecar + per-user scheduling to Go web app
**Researched:** 2026-02-13
**Confidence:** MEDIUM (based on training data + architectural patterns, no current web verification)

## Critical Pitfalls

### Pitfall 1: Plugin Metadata YAML vs Runtime State Mismatch

**What goes wrong:**
Plugin YAML defines 3 settings fields. User saves config to database. Developer updates YAML to add a 4th field. Settings page loads, iterates YAML schema, tries to render value for new field that doesn't exist in user's saved JSON. Template crashes with nil pointer. User sees blank settings page.

**Why it happens:**
Schema evolution is natural during development. YAML is treated as "truth" but database holds stale user data. No schema version tracking. No migration path for settings changes. Template assumes all YAML fields have database values.

**How to avoid:**
- Add `schema_version` field to plugin YAML metadata
- Store schema version with user settings in database: `{version: 1, values: {...}}`
- Write schema migration functions per plugin for version upgrades
- Template code MUST handle missing fields gracefully (default values)
- Add validation: check schema version on settings save, reject mismatches
- Never remove fields from schema (only deprecate), breaking change requires new plugin version

**Example defensive pattern:**
```go
// In settings rendering
func GetSettingValue(userSettings map[string]interface{}, field SchemaField) interface{} {
    if val, ok := userSettings[field.Name]; ok {
        return val
    }
    return field.Default // YAML schema must define defaults
}

// Schema version check
if userSettings.SchemaVersion != plugin.Metadata.SchemaVersion {
    // Run migration or reject with "Please reconfigure plugin"
}
```

**Warning signs:**
- Blank settings pages after YAML updates
- Nil pointer errors in Templ rendering
- User complaints about "settings reset themselves"
- Inconsistent settings validation errors

**Phase to address:**
Phase 1 (Plugin Framework) — Design schema versioning before first plugin ships. Add migration infrastructure.

**Confidence:** HIGH (common configuration management issue)

---

### Pitfall 2: CrewAI Python Process Orphaned on Go Service Restart

**What goes wrong:**
Go service crashes or restarts (deploy, OOM kill, manual restart). CrewAI Python sidecar continues running. New Go instance starts, spawns NEW CrewAI process. Now 2+ Python processes run concurrently, both polling Redis for tasks. Race conditions. Duplicate task processing. Memory leak as processes accumulate.

**Why it happens:**
Python subprocess spawned with `exec.Command()` doesn't die when parent Go process exits unexpectedly. No PID tracking. No health check. Docker/K8s doesn't know about the subprocess. Each restart orphans another process.

**How to avoid:**
- Use shared process namespace in Docker Compose / K8s: `shareProcessNamespace: true`
- Go process MUST register signal handlers (SIGTERM, SIGINT) that kill Python subprocess
- Track CrewAI PID, write to file or Redis, check on startup and kill stale PIDs
- Python process should heartbeat to Redis, Go should monitor heartbeat and restart if dead
- Use process supervisor (systemd, supervisord, or Kubernetes sidecar pattern properly)
- Health check endpoint should verify Python process is alive
- Set subprocess PID group to ensure child processes die with parent: `cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}`

**Example pattern:**
```go
func StartCrewAI(ctx context.Context) (*exec.Cmd, error) {
    // Check for stale PID
    if pid := readPIDFile(); pid != 0 {
        syscall.Kill(pid, syscall.SIGTERM) // Kill stale process
    }

    cmd := exec.CommandContext(ctx, "python", "crewai_worker.py")
    cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

    if err := cmd.Start(); err != nil {
        return nil, err
    }

    writePIDFile(cmd.Process.Pid)

    // Cleanup goroutine
    go func() {
        <-ctx.Done()
        syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM) // Kill process group
        removePIDFile()
    }()

    return cmd, nil
}
```

**Warning signs:**
- Multiple CrewAI processes in `ps aux | grep crewai`
- Memory usage grows after each deploy
- Duplicate briefing generation
- Redis connection count increases over time
- "Address already in use" errors if CrewAI binds port

**Phase to address:**
Phase 3 (CrewAI Integration) — MUST implement before first production deploy. Add integration test that kills Go process and verifies Python cleanup.

**Confidence:** HIGH (common subprocess management issue)

---

### Pitfall 3: Per-User Asynq Scheduler Creates O(users × plugins) Redis Entries

**What goes wrong:**
100 users, each with 5 plugins = 500 scheduled tasks in Redis. Asynq scheduler stores each as individual sorted set entry. Redis memory grows linearly with users. Scheduler scans all entries every tick. CPU spikes. At 10k users with 10 plugins = 100k scheduler entries. Redis OOM.

**Why it happens:**
Naïve approach: create Asynq periodic task per user-plugin pair. Asynq scheduler isn't designed for massive per-user schedules. It's meant for application-level cron (5-50 tasks, not thousands).

**How to avoid:**
- DO NOT use Asynq scheduler per user-plugin
- INSTEAD: Single cron task that queries database for "due now" user-plugin schedules
- Store schedule config in Postgres: `user_plugin_schedules` table with `schedule_cron`, `next_run_at`, `enabled`
- Single scheduler task runs every 5 minutes, queries `WHERE next_run_at <= NOW() AND enabled = true`
- Enqueue individual generation tasks for matched user-plugins
- Update `next_run_at` after enqueuing (calculate next from cron expression)

**Example architecture:**
```go
// WRONG APPROACH (O(users))
for _, user := range users {
    for _, plugin := range user.EnabledPlugins {
        scheduler.Register(plugin.Schedule, CreateTask(user.ID, plugin.ID))
    }
}

// CORRECT APPROACH (O(1) scheduler entries)
scheduler.Register("*/5 * * * *", DispatchScheduledBriefings)

func DispatchScheduledBriefings(ctx context.Context, t *asynq.Task) error {
    // Query database
    var due []UserPluginSchedule
    db.Where("next_run_at <= ? AND enabled = true", time.Now()).Find(&due)

    // Enqueue tasks
    for _, schedule := range due {
        EnqueuePluginExecution(schedule.UserID, schedule.PluginID)

        // Calculate next run
        nextRun := cronexpr.MustParse(schedule.Cron).Next(time.Now())
        db.Model(&schedule).Update("next_run_at", nextRun)
    }
}
```

**Warning signs:**
- Redis memory grows linearly with users
- Scheduler lag increases over time
- Redis SLOWLOG shows sorted set operations
- "Too many keys" errors from Redis

**Phase to address:**
Phase 2 (Per-User Scheduler) — Design with database-backed schedules from start. CRITICAL architecture decision.

**Confidence:** HIGH (scalability anti-pattern)

---

### Pitfall 4: Tile Dashboard N+1 Query for "Latest Briefing Per Plugin"

**What goes wrong:**
Dashboard loads. Queries plugins: `SELECT * FROM plugins`. For each plugin tile, queries: `SELECT * FROM briefings WHERE plugin_id = ? ORDER BY created_at DESC LIMIT 1`. User has 8 plugins = 9 queries. Works fine at low scale. With 50 users viewing dashboard = 450 queries/second. Database CPU spikes.

**Why it happens:**
Templ templates iterate plugins, each template queries for latest briefing. GORM makes this too easy. Developers don't notice until load testing. Classic N+1.

**How to avoid:**
- Pre-fetch latest briefings in handler BEFORE rendering
- Use single query with window function:
  ```sql
  SELECT DISTINCT ON (plugin_id) * FROM briefings
  WHERE user_id = ?
  ORDER BY plugin_id, created_at DESC
  ```
- Or use subquery with join:
  ```sql
  SELECT b.* FROM briefings b
  INNER JOIN (
    SELECT plugin_id, MAX(created_at) as latest
    FROM briefings WHERE user_id = ? GROUP BY plugin_id
  ) latest ON b.plugin_id = latest.plugin_id AND b.created_at = latest.latest
  ```
- Pass `map[PluginID]Briefing` to template, template lookups don't query
- Add query count middleware in dev to panic on >5 queries per request

**Example handler pattern:**
```go
func DashboardHandler(db *gorm.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        userID := c.GetInt("user_id")

        // Get plugins
        var plugins []Plugin
        db.Where("enabled = true").Find(&plugins)

        // Get latest briefing per plugin IN ONE QUERY
        type LatestBriefing struct {
            PluginID uint
            Briefing models.Briefing
        }

        var latestBriefings []models.Briefing
        db.Raw(`
            SELECT DISTINCT ON (plugin_id) * FROM briefings
            WHERE user_id = ? AND plugin_id IN (?)
            ORDER BY plugin_id, created_at DESC
        `, userID, pluginIDs).Scan(&latestBriefings)

        // Build map for template
        briefingMap := make(map[uint]models.Briefing)
        for _, b := range latestBriefings {
            briefingMap[b.PluginID] = b
        }

        // Render with map (no queries in template)
        templates.TileDashboard(plugins, briefingMap).Render(...)
    }
}
```

**Warning signs:**
- Database query count scales with plugin count
- Slow dashboard load with many plugins
- Database CPU high during peak traffic
- GORM logs show repeated similar queries

**Phase to address:**
Phase 4 (Tile Dashboard) — Implement before load testing. Add to review checklist.

**Confidence:** HIGH (classic N+1 pattern)

---

### Pitfall 5: HTMX Dynamic Settings Form Loses Client State on Validation Error

**What goes wrong:**
User fills 8-field settings form. Submits. Server validates, finds error in field 3. Returns error HTML fragment with `hx-swap="outerHTML"`. HTMX replaces entire form. User's input for fields 4-8 vanished. User rage quits.

**Why it happens:**
Server-side validation re-renders entire form from scratch. HTMX swaps out DOM with user's unsubmitted input. Standard form flow but terrible UX for multi-field forms.

**How to avoid:**
- Bind ALL form fields on server, including invalid ones
- Re-render form with user's input preserved (pre-fill fields)
- Use `hx-swap="outerHTML"` but include all form values in re-rendered HTML
- Alternative: client-side validation first (JavaScript), server-side as backup
- Alternative: per-field validation with `hx-target` per input (preserve other fields)
- Store partial form state in session/Redis for multi-step flows

**Example pattern:**
```go
func SavePluginSettings(c *gin.Context) {
    var input PluginSettingsInput
    if err := c.ShouldBind(&input); err != nil {
        // Validation failed - re-render form WITH user input
        templates.SettingsForm(plugin, input, err).Render(...)
        return
    }

    // Save and show success
}

// In Templ template
templ SettingsForm(plugin Plugin, input PluginSettingsInput, validationErr error) {
    <form hx-post="/settings" hx-swap="outerHTML">
        @for _, field := range plugin.Schema.Fields {
            <input
                name={field.Name}
                value={input.Get(field.Name)} // Pre-fill with user input
                class={validationErr.Has(field.Name) ? "error" : ""}
            />
            @if validationErr.Has(field.Name) {
                <span class="error">{validationErr.Message(field.Name)}</span>
            }
        }
    </form>
}
```

**Warning signs:**
- User complaints about "form keeps resetting"
- High form abandonment rate
- Support requests about "settings not saving"
- Users re-typing same data multiple times

**Phase to address:**
Phase 5 (Settings Page) — Design form flow before implementation. Test validation UX early.

**Confidence:** MEDIUM-HIGH (HTMX-specific UX pattern)

---

### Pitfall 6: Plugin YAML Loaded from Filesystem Doesn't Update Without Restart

**What goes wrong:**
Developer updates plugin YAML (adds new field, changes description). File saves. Settings page still shows old metadata. Restart app, now it shows new version. Users during deploy see inconsistent states. CI/CD deploys new pod, some requests hit old pod (old YAML), some hit new pod (new YAML).

**Why it happens:**
Plugin metadata loaded at startup into memory. File changes don't trigger reload. No hot reload mechanism. K8s rolling deploy means version skew during rollout window.

**How to avoid:**
- Accept that YAML updates require deploy (document this)
- Load YAML fresh on each request in DEV mode only: `if cfg.Environment == "dev" { reloadYAML() }`
- Add plugin metadata version to database, migrate on startup, reject stale versions
- Use config hash in metadata table, detect changes on startup
- For production: embrace immutability, YAML changes = new deploy
- Kubernetes: use `maxSurge: 0, maxUnavailable: 1` for rolling updates to prevent version skew
- Alternative: move plugin metadata to database, make it configurable via admin UI (future)

**Example startup pattern:**
```go
func LoadPlugins(db *gorm.DB, pluginDir string) error {
    files, _ := os.ReadDir(pluginDir)

    for _, file := range files {
        metadata := parseYAML(file)

        // Check if plugin exists in DB
        var existing Plugin
        result := db.Where("id = ?", metadata.ID).First(&existing)

        if result.Error != nil {
            // New plugin - insert
            db.Create(&Plugin{Metadata: metadata})
        } else if existing.MetadataHash != metadata.Hash() {
            // Metadata changed - update
            slog.Warn("Plugin metadata changed, updating", "plugin", metadata.ID)
            db.Model(&existing).Update("metadata", metadata)
        }
    }
}
```

**Warning signs:**
- Settings page shows stale descriptions
- Inconsistent behavior during deploys
- "Restart fixes it" reports
- Schema validation errors after YAML updates

**Phase to address:**
Phase 1 (Plugin Framework) — Document reload behavior. Add hash-based change detection.

**Confidence:** MEDIUM (deployment/config management pattern)

---

### Pitfall 7: CrewAI Task Queue Separate from Asynq Creates Duplicate Queue Logic

**What goes wrong:**
Asynq handles Go task queue. CrewAI has its own task queue (Celery, or custom). Two queues. Two retry configs. Two monitoring dashboards. Two sets of dead letter queues. Task flow: Asynq task → HTTP call → CrewAI queue → Python task. Debugging which queue failed is nightmare. Monitoring shows incomplete picture.

**Why it happens:**
Each system brings its own queue. Python ecosystem defaults to Celery. Developers don't question it. "Two queues" seems normal until operations.

**How to avoid:**
- Use Asynq as SINGLE source of truth for all tasks
- CrewAI tasks are enqueued by Asynq, not by separate Python queue
- Pattern 1: Asynq calls Python HTTP endpoint synchronously (task blocks until Python completes)
- Pattern 2: Asynq writes to Redis queue that Python polls (Redis list as simple queue)
- Pattern 3: Python has no queue, runs as long-lived process that Go calls via HTTP/gRPC per task
- Monitoring: Single dashboard (Asynqmon) shows all task states
- Retry logic: Asynq retries, Python just returns success/failure

**Recommended architecture:**
```
User request → Asynq enqueue → Asynq worker → HTTP POST to CrewAI service → CrewAI runs workflow → Return result → Asynq marks complete
```

NOT:
```
User request → Asynq → HTTP to Python → Python enqueues to Celery → Celery worker → ...
```

**Python service pattern (no queue):**
```python
# crewai_service.py - Flask/FastAPI endpoint
@app.post("/workflow/news_digest")
def run_news_digest(payload: NewsDigestInput):
    crew = NewsDigestCrew(payload)
    result = crew.kickoff()  # Runs synchronously
    return {"status": "success", "content": result}
```

**Warning signs:**
- Two different queue dashboards
- Inconsistent retry behavior
- "Task succeeded in one queue but failed in other"
- Complex monitoring setup
- Duplicate task tracking logic

**Phase to address:**
Phase 3 (CrewAI Integration) — Decide queue strategy before implementation. CRITICAL architecture decision.

**Confidence:** HIGH (distributed systems anti-pattern)

---

### Pitfall 8: Account Tier Checks in Templates Create Logic Duplication

**What goes wrong:**
Free tier limited to 3 plugins. Logic added to settings page template: `@if user.Tier == "free" && enabledCount >= 3 { disable }`. Later added to API handler validation. Then added to cron scheduler. Three places. Developer updates limit to 5, changes template, forgets handler. Users exploit via API.

**Why it happens:**
Templates are "just display" but encode business logic. Easy to add `if` statements. No single source of truth. Copy-paste between template/handler/worker.

**How to avoid:**
- Business logic NEVER in templates
- Create tier service: `tierService.CanEnablePlugin(user)` returns bool + reason
- Call from handler, pass result to template: `templates.Settings(canEnable, reason)`
- Template only shows/hides based on passed boolean, no logic
- Tier limits in config or database, not hardcoded
- Handler validation is authoritative, template is UX hint only
- Always enforce server-side (API/handler), template just provides better UX

**Example pattern:**
```go
// Service layer
type TierService struct{}

func (s *TierService) CanEnablePlugin(user User, currentCount int) (bool, string) {
    limit := tierLimits[user.Tier].MaxPlugins
    if currentCount >= limit {
        return false, fmt.Sprintf("Free tier limited to %d plugins", limit)
    }
    return true, ""
}

// Handler
func SettingsPage(c *gin.Context) {
    user := getCurrentUser(c)
    enabledCount := getEnabledPluginCount(user)

    canEnable, reason := tierService.CanEnablePlugin(user, enabledCount)

    templates.SettingsPage(user, canEnable, reason).Render(...)
}

// Template (no logic!)
templ SettingsPage(user User, canEnable bool, reason string) {
    @if !canEnable {
        <div class="glass-alert glass-alert-error">{reason}</div>
    }
    <button disabled?={!canEnable}>Enable Plugin</button>
}
```

**Warning signs:**
- Same `if tier == ...` logic in 3+ files
- Tier limit changes require multi-file edits
- Security issues where template allows but API rejects (or vice versa)
- Inconsistent error messages for same condition

**Phase to address:**
Phase 6 (Account Tiers) — Create tier service immediately. Enforce single source of truth.

**Confidence:** HIGH (separation of concerns principle)

---

### Pitfall 9: Tile Dashboard Layout Breaks with Long Plugin Names or Missing Data

**What goes wrong:**
Plugin name "Daily International Technology & Business News Digest" wraps across 4 lines in tile. Tile height grows. Grid layout breaks. Or: plugin has no latest briefing. Template tries to render `briefing.Content`, nil pointer error. Entire dashboard blank.

**Why it happens:**
Fixed grid layout assumes uniform content. Real-world data has extreme variance. Templates don't handle nil gracefully. CSS doesn't constrain text overflow.

**How to avoid:**
- CSS: Limit plugin name to 2 lines with ellipsis: `line-clamp: 2; overflow: hidden; text-overflow: ellipsis`
- Validate plugin names on save: max 50 characters
- Tile grid: use `auto-fit` with `minmax()` for responsive flexibility
- Template MUST handle missing briefing: `@if briefing != nil { } else { <EmptyState /> }`
- Test with edge cases: no data, very long names, 1 plugin, 50 plugins
- Set minimum tile height to prevent layout collapse

**CSS pattern:**
```css
.plugin-tile-title {
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
    text-overflow: ellipsis;
    max-height: 3em; /* 2 lines */
}

.tile-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
    gap: 1.5rem;
}
```

**Template pattern:**
```templ
templ PluginTile(plugin Plugin, briefing *Briefing) {
    <div class="glass-card plugin-tile">
        <h3 class="plugin-tile-title">{plugin.Name}</h3>
        @if briefing != nil {
            <p>{truncate(briefing.Content, 150)}</p>
            <span class="badge">{briefing.Status}</span>
        } else {
            <div class="empty-state">No briefings yet</div>
        }
    </div>
}
```

**Warning signs:**
- Layout breaks on staging but not dev
- Specific plugins cause page errors
- Grid alignment issues
- Users report "weird spacing"
- Nil pointer panics in templates

**Phase to address:**
Phase 4 (Tile Dashboard) — Add constraints during implementation. Test with edge cases before merge.

**Confidence:** MEDIUM (frontend layout patterns)

---

### Pitfall 10: Plugin Schema Type Coercion Fails on Form Submit

**What goes wrong:**
Plugin schema defines `max_items: {type: "integer", default: 10}`. Settings form renders `<input type="number">`. User changes to "20", submits. Form data arrives as string `"20"`. JSON unmarshaling to schema struct fails (expects int). Validation error: "invalid type". User sees cryptic error.

**Why it happens:**
HTML forms send everything as strings. Schema defines types (int, bool, array). No automatic coercion. Developer forgets to convert before validation.

**How to avoid:**
- Parse form data with type coercion BEFORE schema validation
- For each schema field, convert string to expected type:
  - `integer` → `strconv.Atoi()`
  - `boolean` → `value == "true" || value == "on"`
  - `array` → split by delimiter or parse JSON
- Use schema field type to drive conversion logic
- Validation happens AFTER coercion
- Return user-friendly errors: "Max items must be a number" not "type mismatch"

**Example coercion logic:**
```go
func CoerceFormValue(value string, fieldType string) (interface{}, error) {
    switch fieldType {
    case "integer":
        return strconv.Atoi(value)
    case "number":
        return strconv.ParseFloat(value, 64)
    case "boolean":
        return value == "true" || value == "on" || value == "1", nil
    case "array":
        if value == "" {
            return []string{}, nil
        }
        return strings.Split(value, ","), nil
    default:
        return value, nil // string
    }
}

func ParseSettings(form url.Values, schema Schema) (map[string]interface{}, error) {
    settings := make(map[string]interface{})

    for _, field := range schema.Fields {
        rawValue := form.Get(field.Name)
        coerced, err := CoerceFormValue(rawValue, field.Type)
        if err != nil {
            return nil, fmt.Errorf("%s: %w", field.Label, err)
        }
        settings[field.Name] = coerced
    }

    return settings, nil
}
```

**Warning signs:**
- "Invalid type" errors on valid user input
- Integer fields reject numbers
- Boolean checkboxes don't work
- Form values stored as strings in database
- JavaScript required to make forms work

**Phase to address:**
Phase 5 (Settings Page) — Implement coercion before first settings form. Add test cases for all schema types.

**Confidence:** MEDIUM-HIGH (form handling pattern)

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| No plugin schema versioning | Faster initial development | Can't evolve settings, breaking changes on updates | Never - add from start |
| Hardcoded plugin list in Go | Simpler than YAML loading | Every new plugin requires code change + deploy | MVP only - 1-2 plugins |
| CrewAI runs as subprocess not sidecar | Single container, simpler deploy | Process management issues, harder to scale | Local dev only, never production |
| Settings validation only client-side | Better UX, less server load | Security risk, easily bypassed | Never - always validate server-side |
| Tile dashboard uses client-side grid (Masonry.js) | Easier than CSS grid with dynamic content | Requires JavaScript, layout shift, not SSR-friendly | If CSS grid truly can't handle layout |
| Store all plugin metadata in YAML | Easy to version in Git | Can't update without deploy | Good for v1.1 - move to DB in v1.2+ |
| Per-user Asynq scheduler instances | Simple to implement | O(users) Redis memory, doesn't scale | Never - use database-backed scheduling |
| Manual plugin registration (no auto-discovery) | Explicit, safer | Forgetting to register = invisible plugin | Small plugin count (<5), then add discovery |

---

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| **Plugin YAML + GORM** | Load YAML every request | Load at startup, cache in memory, reload on deploy |
| **CrewAI + Asynq** | Two separate task queues | Asynq enqueues, calls CrewAI HTTP endpoint synchronously |
| **Templ + Dynamic Schema** | Loop in template with DB queries | Pre-fetch all data in handler, pass to template |
| **HTMX + Form Validation** | Replace entire form on error | Re-render with user's input pre-filled |
| **Tile Grid + Missing Data** | Assume all tiles have data | Handle nil briefings gracefully in template |
| **Schema Types + HTML Forms** | Expect typed data from form | Coerce string form data to schema types |
| **Per-User Schedule + Redis** | Create Asynq schedule per user | Database table + single cron that queries due schedules |
| **Plugin Settings + User Model** | Store as TEXT in user table | Separate `user_plugin_settings` table with JSONB |
| **Account Tier + Template Logic** | Check tier in template | Tier service in handler, pass boolean to template |
| **Python Subprocess + K8s** | Spawn subprocess, hope it dies with parent | Use shared process namespace or proper sidecar pattern |

---

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| N+1 latest briefing queries | Dashboard slow with many plugins | Single query with window function or subquery | >5 plugins per user |
| Plugin YAML loaded per request | Every request reads filesystem | Load at startup, cache in memory | Any production traffic |
| CrewAI HTTP calls without timeout | Requests hang forever | Set timeout: `http.Client{Timeout: 60*time.Second}` | First external API slowdown |
| All plugins run serially | Daily cron takes 5min × 8 plugins = 40min | Enqueue plugin tasks in parallel, workers process concurrently | >3 plugins per user |
| Settings schema validation on every render | Template rendering slow | Validate on save only, render assumes valid | Complex schemas (>10 fields) |
| Tile dashboard fetches all briefing content | Large page size, slow load | Fetch metadata only, lazy-load content on expand | >20 tiles |
| Per-user scheduler with polling | CPU spikes every minute | Use database index on `next_run_at`, efficient query | >1000 users |

---

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| **Plugin YAML path traversal** | User provides `../../../../etc/passwd` as plugin ID | Validate plugin ID against allowed characters (alphanumeric + dash only) |
| **Settings schema allows code execution** | Schema type "eval" or "function" enables RCE | Restrict schema types to primitives: string, integer, boolean, array |
| **No tier enforcement server-side** | Free users bypass limits via API | Always enforce tier checks in handler, template is UX only |
| **CrewAI API key in task payload** | API key logged in Asynq/Redis | Store API keys encrypted in database, fetch in task handler |
| **Plugin-to-plugin data access** | One plugin reads another's settings | Scope settings queries by user ID + plugin ID |
| **User can set other user's schedule** | Authorization bypass | Check `user_id` from session matches schedule's owner |
| **Python code injection via settings** | User provides `"; os.system('rm -rf /')` in setting | Escape/validate all settings before passing to Python |
| **Tile dashboard exposes all users' data** | Missing user filter on briefings query | ALWAYS filter by `user_id` from session |

---

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| **No loading state on manual trigger** | User clicks "Generate Now", nothing happens for 30s | Show spinner + "Generating..." immediately, poll for status |
| **Tile shows "No briefing" vs "Loading"** | User thinks plugin broken, actually generating | Distinguish states: loading (spinner), empty (no data), error (red badge) |
| **Settings form loses data on error** | User re-fills 8 fields after typo | Re-render form with submitted values pre-filled |
| **Plugin enable/disable requires page reload** | Clunky UX, feels broken | Use HTMX swap to update tile in-place |
| **No feedback when schedule changes** | User unsure if save worked | Show success toast or update tile with "Next run: tomorrow 7am" |
| **Account tier limit error after 5min of settings config** | User rage quits | Check tier BEFORE showing settings form, disable upgrade-required plugins |
| **Tile dashboard has no empty state** | Blank page for new users | Show "Enable your first plugin" with CTA |

---

## "Looks Done But Isn't" Checklist

- [ ] **Plugin YAML Schema:** Often missing default values — verify every field has a default for graceful degradation
- [ ] **CrewAI Process Management:** Often missing signal handlers — verify Python process dies when Go restarts
- [ ] **Per-User Scheduling:** Often missing timezone handling — verify schedule respects user's timezone, not server's
- [ ] **Settings Form Validation:** Often missing type coercion — verify integer/boolean fields work, not just strings
- [ ] **Tile Dashboard:** Often missing nil checks — verify empty briefing state renders without errors
- [ ] **Account Tier Checks:** Often only in template — verify enforcement in API handlers, not just UI
- [ ] **Plugin Metadata Changes:** Often missing migration path — verify schema version updates don't break existing settings
- [ ] **Asynq Task Uniqueness:** Often missing for manual triggers — verify duplicate "Generate Now" clicks don't create duplicate tasks
- [ ] **HTMX Polling Cleanup:** Often missing stop condition — verify polling stops when tile shows completed briefing
- [ ] **Python Virtualenv:** Often missing in Docker — verify CrewAI runs in venv, not system Python

---

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Schema version mismatch breaks settings | MEDIUM | 1. Add `schema_version` to DB, 2. Write migration for old settings, 3. Deploy with version check |
| Orphaned Python processes | LOW | 1. `pkill -f crewai`, 2. Add signal handlers, 3. Redeploy |
| O(users) scheduler entries | HIGH | 1. Stop scheduler, 2. Clear Redis scheduled tasks, 3. Implement DB-backed scheduling, 4. Migrate cron configs to DB, 5. Redeploy |
| N+1 queries on dashboard | LOW | 1. Add query to pre-fetch latest briefings, 2. Pass map to template, 3. Remove template queries, 4. Deploy |
| Duplicate task queues (Asynq + Celery) | HIGH | 1. Decide on single queue, 2. Refactor Python service to HTTP endpoint, 3. Update Asynq tasks to call Python, 4. Remove Celery, 5. Redeploy both services |
| Settings logic in templates | MEDIUM | 1. Extract tier service, 2. Move checks to handlers, 3. Update templates to use booleans, 4. Add tests, 5. Deploy |
| Form data type mismatch | LOW | 1. Add coercion function, 2. Call before validation, 3. Add test cases, 4. Deploy |
| Tile layout breaks | LOW | 1. Add CSS constraints (line-clamp, min-height), 2. Handle nil in templates, 3. Test with edge cases, 4. Deploy |

---

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| Schema version mismatch | Phase 1 (Plugin Framework) | Settings page loads after YAML update without errors |
| Orphaned Python processes | Phase 3 (CrewAI Integration) | Kill Go process, verify Python dies within 5s |
| O(users) scheduler entries | Phase 2 (Per-User Scheduler) | Load test with 1000 users shows <100 Redis keys |
| Dashboard N+1 queries | Phase 4 (Tile Dashboard) | Query count middleware shows <5 queries regardless of plugin count |
| Form state loss on validation | Phase 5 (Settings Page) | Submit invalid form, verify all valid fields preserve values |
| YAML reload issues | Phase 1 (Plugin Framework) | Update YAML, verify old pods serve old metadata until new pods ready |
| Duplicate task queues | Phase 3 (CrewAI Integration) | Single Asynqmon dashboard shows all tasks, no Celery |
| Tier logic duplication | Phase 6 (Account Tiers) | Grep codebase for tier checks, find only in service layer |
| Type coercion failures | Phase 5 (Settings Page) | Test suite includes integer, boolean, array fields |
| Tile layout breaks | Phase 4 (Tile Dashboard) | Test with 0 plugins, 1 plugin, 50 plugins, long names, nil briefings |

---

## Sources

- **Asynq Documentation:** https://github.com/hibiken/asynq (scheduling, uniqueness, retries)
- **GORM Documentation:** https://gorm.io/docs/ (queries, window functions, N+1 prevention)
- **HTMX Documentation:** https://htmx.org/docs/ (form handling, swaps, OOB)
- **Templ Best Practices:** https://templ.guide/ (component patterns, nil handling)
- **CrewAI Documentation:** https://docs.crewai.com/ (workflow patterns)
- **Go Subprocess Management:** https://pkg.go.dev/os/exec (process groups, signal handling)
- **Training Data:** Plugin architecture patterns, schema versioning, process management, form handling, distributed task queues

**Confidence Assessment:**
- Plugin framework pitfalls: MEDIUM (schema versioning is common, plugin-specific is newer pattern)
- CrewAI integration: MEDIUM (subprocess management is HIGH, CrewAI-specific patterns MEDIUM due to newer tool)
- Per-user scheduling: HIGH (well-established scalability anti-patterns)
- Dynamic settings forms: MEDIUM-HIGH (form handling HIGH, schema-driven is MEDIUM)
- Tile dashboard: MEDIUM (layout patterns HIGH, integration-specific MEDIUM)
- Account tiers: HIGH (separation of concerns is fundamental)

**Limitations:**
- Unable to verify with current 2026 sources (WebSearch unavailable)
- CrewAI production patterns less established (newer tool)
- Plugin framework confidence lower due to application-specific patterns
- Python-Go integration patterns based on general subprocess management, not CrewAI-specific production experience

---

*Pitfalls research for: Adding plugin architecture + CrewAI sidecar + per-user scheduling to Go/Gin/Templ/HTMX/GORM/Asynq web app*
*Researched: 2026-02-13*
*Confidence: MEDIUM overall (HIGH for established patterns like N+1 and subprocess management, MEDIUM for newer integration patterns like CrewAI and schema-driven settings)*
