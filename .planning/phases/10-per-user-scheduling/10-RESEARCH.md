# Phase 10: Per-User Scheduling - Research

**Researched:** 2026-02-18
**Domain:** Database-backed per-user per-plugin scheduling, cron evaluation, timezone handling, Redis caching
**Confidence:** HIGH

## Summary

Phase 10 replaces the current global Asynq cron scheduler (`worker.StartScheduler`) with a database-backed per-minute evaluation loop. Instead of one global cron registered with Asynq Scheduler, a single Asynq task fires every minute, queries the database for enabled user-plugin configs that have a cron schedule due at the current time in the user's timezone, and enqueues `plugin:execute` tasks for matching pairs. This avoids O(users x plugins) Redis entries and keeps schedule configuration in Postgres.

The critical technical challenge is timezone-aware cron evaluation. The codebase already includes `robfig/cron/v3` as a transitive dependency (via Asynq), so there is no new library to install. The `Schedule` interface from `robfig/cron/v3` exposes a `Next(t time.Time) time.Time` method that can be used standalone — without running the cron runner — to compute whether a cron expression was due within the past minute, in the user's local timezone.

Redis caching for last-run times uses the existing `github.com/redis/go-redis/v9` client (already in go.mod). A hash key `scheduler:last_run` with field `{userID}:{pluginID}` stores the Unix timestamp of the last successful enqueue. The per-minute task reads this cache before evaluating cron expressions, writing back after a successful enqueue to prevent double-scheduling.

**Primary recommendation:** Parse cron with `robfig/cron/v3` standalone, evaluate `Next(lastRun)` against `time.Now()` in the user's timezone, cache last-run timestamps in Redis HSET, and replace `StartScheduler` with a single `* * * * *` Asynq task that runs `handlePerMinuteScheduler`.

---

## Standard Stack

### Core (all already in go.mod — no new dependencies)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/robfig/cron/v3` | v3.0.1 | Cron expression parsing + `Schedule.Next()` | Already a transitive dep via Asynq; provides the `Parser` and `Schedule` interface used standalone |
| `github.com/hibiken/asynq` | v0.26.0 | Per-minute task dispatch + plugin:execute enqueue | Already in use; `* * * * *` task drives the evaluation loop |
| `github.com/redis/go-redis/v9` | v9.17.3 | Redis HSET/HGET for last-run timestamp cache | Already in use for Redis Streams; same client used for caching |
| `gorm.io/gorm` | v1.31.1 | Query `user_plugin_configs` for enabled, scheduled rows | Already in use |

### No New Dependencies Required

All required libraries are already direct or transitive dependencies. No `go get` needed.

**Installation:** None — all packages already present in `go.mod`.

---

## Architecture Patterns

### Recommended Project Structure Changes

```
internal/
├── worker/
│   ├── scheduler.go          # REPLACE: remove StartScheduler; add StartPerMinuteScheduler
│   ├── tasks.go              # ADD: TaskPerMinuteScheduler constant
│   └── worker.go             # ADD: mux handler for TaskPerMinuteScheduler
├── plugins/
│   └── models.go             # EXTEND: add CronExpression + Timezone fields to UserPluginConfig
internal/database/
└── migrations/
    └── 000006_add_schedule_to_user_plugin_configs.up.sql   # new migration
```

### Pattern 1: Per-Minute Asynq Task as Evaluation Loop

**What:** A single Asynq task registered with `* * * * *` replaces the global cron. The task handler queries the database and fires per-user-plugin plugin:execute tasks.

**When to use:** When you need database-backed dynamic schedules rather than static Asynq Scheduler entries.

**Example:**

```go
// Source: hibiken/asynq Scheduler API (pkg.go.dev/github.com/hibiken/asynq)
// In StartPerMinuteScheduler (replaces StartScheduler):
scheduler := asynq.NewScheduler(redisOpt, &asynq.SchedulerOpts{
    Location: time.UTC, // scheduler itself runs in UTC; TZ is per-user
    Logger:   &asynqLoggerAdapter{logger: logger},
})

task := asynq.NewTask(
    TaskPerMinuteScheduler,
    nil,
    asynq.MaxRetry(0),         // Don't retry — next minute will catch up
    asynq.Timeout(50*time.Second), // Must complete before next minute fires
    asynq.Unique(55*time.Second),  // Prevent overlapping runs
)

entryID, err := scheduler.Register("* * * * *", task)
```

### Pattern 2: Standalone Cron Expression Evaluation with robfig/cron/v3

**What:** Use `robfig/cron/v3` Parser to parse cron expressions and evaluate `Next(lastRun)` against the current time, all in the user's timezone. No cron runner is started.

**When to use:** For per-user-per-plugin schedule evaluation inside the per-minute task handler.

**Example:**

```go
// Source: pkg.go.dev/github.com/robfig/cron/v3 — Parser.Parse, Schedule.Next
import "github.com/robfig/cron/v3"

func isDue(cronExpr string, timezone string, lastRunAt time.Time) (bool, error) {
    loc, err := time.LoadLocation(timezone)
    if err != nil {
        loc = time.UTC
    }

    // Prepend CRON_TZ so the parser applies timezone automatically
    specWithTZ := "CRON_TZ=" + timezone + " " + cronExpr

    parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
    schedule, err := parser.Parse(specWithTZ)
    if err != nil {
        return false, fmt.Errorf("invalid cron expression %q: %w", cronExpr, err)
    }

    now := time.Now().In(loc)

    // The window: was the next scheduled time AFTER lastRun but BEFORE now+1min?
    nextAfterLast := schedule.Next(lastRunAt)
    return !nextAfterLast.After(now), nil
}
```

**Critical note:** `Next(t)` returns the next scheduled time STRICTLY AFTER `t`. The check `!nextAfterLast.After(now)` is correct: it means "the next run after last-run has already passed (or is now), so this is due."

### Pattern 3: Redis HSET for Last-Run Timestamp Cache

**What:** One Redis hash key `scheduler:last_run` with field `{userID}:{pluginID}` stores the Unix timestamp of the last successful enqueue for each user-plugin pair.

**When to use:** To avoid re-querying the database for last-run times on every per-minute tick.

**Example:**

```go
// Source: redis.io/docs/latest/commands/hset + pkg.go.dev/github.com/redis/go-redis/v9
const lastRunHashKey = "scheduler:last_run"

func fieldKey(userID, pluginID uint) string {
    return fmt.Sprintf("%d:%d", userID, pluginID)
}

// Get last run time from cache (fallback to zero time if not found)
func getLastRun(ctx context.Context, rdb *redis.Client, userID, pluginID uint) time.Time {
    val, err := rdb.HGet(ctx, lastRunHashKey, fieldKey(userID, pluginID)).Result()
    if err != nil {
        return time.Time{} // zero = never ran
    }
    ts, err := strconv.ParseInt(val, 10, 64)
    if err != nil {
        return time.Time{}
    }
    return time.Unix(ts, 0)
}

// Set last run time in cache after successful enqueue
func setLastRun(ctx context.Context, rdb *redis.Client, userID, pluginID uint, t time.Time) {
    rdb.HSet(ctx, lastRunHashKey, fieldKey(userID, pluginID), t.Unix())
}
```

**Key detail:** Cache is write-through only — the source of truth for schedule configuration is the database. The cache exists purely to avoid DB reads during evaluation. If the cache is cold, `lastRunAt = time.Time{}` (zero) is safe: `Next(zero)` returns the first valid run time, which will almost certainly be in the past, so the pair fires on first evaluation.

### Pattern 4: Database Migration to Add Schedule Fields

**What:** Add `cron_expression` and `timezone` columns to `user_plugin_configs`. Use `golang-migrate` SQL files (same as existing migrations 000001–000005).

**Example:**

```sql
-- 000006_add_schedule_to_user_plugin_configs.up.sql
ALTER TABLE user_plugin_configs
    ADD COLUMN cron_expression VARCHAR(100),
    ADD COLUMN timezone VARCHAR(100) NOT NULL DEFAULT 'UTC';

-- Index for the scheduler query: only query enabled+scheduled rows
CREATE INDEX idx_user_plugin_configs_scheduled
    ON user_plugin_configs(enabled, cron_expression)
    WHERE deleted_at IS NULL AND enabled = true AND cron_expression IS NOT NULL;
```

```sql
-- 000006_add_schedule_to_user_plugin_configs.down.sql
DROP INDEX IF EXISTS idx_user_plugin_configs_scheduled;
ALTER TABLE user_plugin_configs
    DROP COLUMN IF EXISTS cron_expression,
    DROP COLUMN IF EXISTS timezone;
```

### Pattern 5: Handler Structure for Per-Minute Task

**What:** `handlePerMinuteScheduler` is the Asynq task handler that: (1) queries `user_plugin_configs` for enabled rows with a non-null cron_expression, (2) evaluates each pair, (3) enqueues `plugin:execute` tasks for due pairs.

**Example:**

```go
// Source: pattern derived from existing handleScheduledBriefingGeneration in worker.go
func handlePerMinuteScheduler(logger *slog.Logger, db *gorm.DB, rdb *redis.Client) func(context.Context, *asynq.Task) error {
    return func(ctx context.Context, task *asynq.Task) error {
        // Query enabled configs with a cron schedule set
        var configs []plugins.UserPluginConfig
        if err := db.WithContext(ctx).
            Preload("Plugin").
            Preload("User").
            Where("enabled = ? AND cron_expression IS NOT NULL AND cron_expression != ''", true).
            Find(&configs).Error; err != nil {
            return fmt.Errorf("failed to query scheduled configs: %w", err)
        }

        for _, cfg := range configs {
            lastRun := getLastRun(ctx, rdb, cfg.UserID, cfg.PluginID)
            due, err := isDue(cfg.CronExpression, cfg.Timezone, lastRun)
            if err != nil {
                logger.Warn("Invalid cron expression, skipping",
                    "user_id", cfg.UserID, "plugin_id", cfg.PluginID, "error", err)
                continue
            }
            if !due {
                continue
            }

            // Build settings from the config
            var settings map[string]interface{}
            if err := json.Unmarshal(cfg.Settings, &settings); err != nil {
                settings = map[string]interface{}{}
            }

            if err := EnqueueExecutePlugin(cfg.PluginID, cfg.UserID, cfg.Plugin.Name, settings); err != nil {
                logger.Error("Failed to enqueue plugin execution",
                    "user_id", cfg.UserID, "plugin_id", cfg.PluginID, "error", err)
                continue
            }

            setLastRun(ctx, rdb, cfg.UserID, cfg.PluginID, time.Now())
            logger.Info("Enqueued scheduled plugin execution",
                "user_id", cfg.UserID, "plugin_name", cfg.Plugin.Name)
        }
        return nil
    }
}
```

### Anti-Patterns to Avoid

- **Per-user Asynq Scheduler entries:** Registering one Asynq Scheduler cron entry per user-plugin pair creates O(users x plugins) Redis keys and requires scheduler restart to pick up config changes. The phase explicitly rejects this.
- **Using `asynq.Unique` duration longer than 1 minute on the per-minute task:** Unique locks must expire before the next fire. Use 55 seconds, not 1 hour.
- **Evaluating `Next(time.Now())` instead of `Next(lastRun)`:** `Next(now)` tells you when the NEXT run is, not whether one was missed. Always pass `lastRun` (or zero if never run).
- **Hard-coding UTC in the cron parser:** The user's `Timezone` field from the database must be applied. A user who sets "6:00 AM" expects 6 AM in their local time, not UTC.
- **Not preloading Plugin and User associations:** The handler needs `plugin.Name` for `EnqueueExecutePlugin` and `user.Timezone` for evaluation. Use `Preload` or join.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Cron expression parsing | Custom regex/tokenizer for cron strings | `robfig/cron/v3` Parser | Handles edge cases: month names, day-of-week aliases, ranges, steps, L/W/# for some parsers; already a transitive dep |
| Timezone-aware time comparison | Manual UTC offset math | `time.LoadLocation` + cron `CRON_TZ=` prefix | Go's `time` package handles DST transitions; robfig handles IANA timezone names |
| Distributed scheduler deduplication | Custom locking in code | `asynq.Unique(55 * time.Second)` on the per-minute task | Asynq's Unique option uses Redis atomic operations to prevent overlapping runs |
| Last-run persistence across restarts | In-memory map in the process | Redis HSET (persisted) | In-memory state is lost on restart; Redis survives restarts and is already required |

**Key insight:** The cron-evaluation logic is ~20 lines using robfig's standalone Parser. Building this from scratch would miss DST, month-end edge cases, and leap year handling baked into the library.

---

## Common Pitfalls

### Pitfall 1: Evaluating Cron in UTC Instead of User's Timezone

**What goes wrong:** User sets "0 6 * * *" (6 AM), but `isDue` evaluates it in UTC. A PST user's schedule fires at 2 PM instead of 6 AM.
**Why it happens:** `time.Now()` returns local server time, and passing it to `Next()` uses the timezone embedded in the cron parser — which defaults to `time.Local` if no CRON_TZ prefix is given.
**How to avoid:** Always prepend `CRON_TZ=` + user timezone to the cron expression before parsing. Load and validate timezone from DB at startup to catch invalid values early.
**Warning signs:** Users in non-UTC timezones report wrong briefing times; schedule fires at UTC-equivalent of requested local time.

### Pitfall 2: Cold Cache Causes Mass Enqueue on First Start

**What goes wrong:** On first deployment (or after Redis flush), `lastRun = time.Time{}` (zero) for all pairs. `Next(zero)` returns a time far in the past, so ALL scheduled pairs appear due simultaneously. This floods the Asynq queue.
**Why it happens:** The zero value of `time.Time` is Jan 1, year 1. `Next(zero)` returns the earliest valid cron time, which is always in the past.
**How to avoid:** On cache miss, treat the current time as the baseline: `lastRun = time.Now()` for the first evaluation, OR use `time.Now().Truncate(time.Minute)` so any schedule whose next run after "now" is still in the future won't trigger immediately.
**Warning signs:** Mass plugin:execute tasks enqueued on first startup after Redis is cleared.

### Pitfall 3: Per-Minute Task Timeout Exceeding One Minute

**What goes wrong:** If the handler takes more than 60 seconds (e.g., 100+ users with slow DB queries), the next minute's task fires before the current one completes, causing concurrent scheduler runs.
**Why it happens:** Asynq allows concurrent task processing by default. The per-minute task has no inherent mutual exclusion beyond `asynq.Unique`.
**How to avoid:** Set `asynq.Timeout(50 * time.Second)` and `asynq.Unique(55 * time.Second)` on the per-minute task. Add a database index on `(enabled, cron_expression) WHERE deleted_at IS NULL` to keep the query fast.
**Warning signs:** Duplicate `plugin:execute` tasks for the same user-plugin pair in the same minute.

### Pitfall 4: `UserPluginConfig` Missing Plugin Name in Handler

**What goes wrong:** `EnqueueExecutePlugin` requires `plugin_name` (string), but `UserPluginConfig` only has `PluginID`. Without Preload, `cfg.Plugin.Name` is empty string.
**Why it happens:** GORM does not auto-load associations; you must explicitly Preload.
**How to avoid:** Use `db.Preload("Plugin").Find(&configs)` in the scheduler query.
**Warning signs:** `EnqueueExecutePlugin` called with empty plugin_name; CrewAI receives empty plugin name and fails.

### Pitfall 5: Removing the Global Scheduler Without Wiring the New One

**What goes wrong:** `StartScheduler` is deleted but `StartPerMinuteScheduler` is not wired into `main.go` embedded mode OR worker mode. No briefings are generated.
**Why it happens:** Phase requires modifying TWO startup paths in main.go: the `*workerMode` branch and the `cfg.Env == "development"` embedded branch.
**Warning signs:** After the phase is complete, no plugin execution tasks appear in Asynq queue.

### Pitfall 6: CronExpression Stored Without Validation

**What goes wrong:** A user or admin saves a malformed cron expression to the database. The scheduler silently skips it every minute, with no feedback.
**Why it happens:** No validation at write time; validation only happens at evaluation time.
**How to avoid:** Validate cron expressions before storing (use `parser.Parse()` as a validator). Return an error to the UI if invalid. Log and skip (don't fail the whole task) at evaluation time.
**Warning signs:** Users report schedule "not working" with no error messages.

---

## Code Examples

### Cron Expression Validation at Write Time

```go
// Source: pkg.go.dev/github.com/robfig/cron/v3 — Parser API
func validateCronExpression(expr string) error {
    parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
    _, err := parser.Parse(expr)
    return err
}
```

### Full isDue Function with Timezone Support

```go
// Source: pkg.go.dev/github.com/robfig/cron/v3 — CRON_TZ prefix, Schedule.Next
func isDue(cronExpr string, timezone string, lastRunAt time.Time) (bool, error) {
    if cronExpr == "" {
        return false, nil
    }
    if timezone == "" {
        timezone = "UTC"
    }

    // Validate timezone first; fall back to UTC if invalid
    _, err := time.LoadLocation(timezone)
    if err != nil {
        timezone = "UTC"
    }

    specWithTZ := "CRON_TZ=" + timezone + " " + cronExpr
    parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
    schedule, err := parser.Parse(specWithTZ)
    if err != nil {
        return false, fmt.Errorf("invalid cron expression: %w", err)
    }

    // If never run, use now minus 1 minute to avoid mass-fire on first start
    if lastRunAt.IsZero() {
        lastRunAt = time.Now().Add(-time.Minute)
    }

    nextRun := schedule.Next(lastRunAt)
    return !nextRun.After(time.Now()), nil
}
```

### Redis Client for Scheduler (reusing existing go-redis/v9)

```go
// Source: pkg.go.dev/github.com/redis/go-redis/v9
func newSchedulerCache(redisURL string) (*redis.Client, error) {
    opts, err := redis.ParseURL(redisURL)
    if err != nil {
        return nil, fmt.Errorf("failed to parse redis URL: %w", err)
    }
    return redis.NewClient(opts), nil
}
```

### GORM Model Extension for UserPluginConfig

```go
// Source: codebase internal/plugins/models.go — extend existing struct
type UserPluginConfig struct {
    gorm.Model
    UserID         uint           `gorm:"not null;uniqueIndex:idx_user_plugin"`
    PluginID       uint           `gorm:"not null;uniqueIndex:idx_user_plugin"`
    Settings       datatypes.JSON `gorm:"type:jsonb"`
    Enabled        bool           `gorm:"default:false"`
    CronExpression string         `gorm:"column:cron_expression"`    // NEW: e.g. "0 6 * * *"
    Timezone       string         `gorm:"column:timezone;default:'UTC'"` // NEW: IANA name e.g. "America/Los_Angeles"
    User           models.User    `gorm:"constraint:OnDelete:CASCADE;"`
    Plugin         Plugin         `gorm:"constraint:OnDelete:CASCADE;"`
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Global Asynq Scheduler cron for all users | Per-minute DB evaluation loop | Phase 10 | Enables per-user schedules with no Redis key explosion |
| `cfg.BriefingSchedule` + `cfg.BriefingTimezone` global config | `UserPluginConfig.CronExpression` + `.Timezone` per user-plugin | Phase 10 | Config decoupled from environment; users can set their own schedule |
| `TaskScheduledBriefingGeneration` — queries all users | `TaskPerMinuteScheduler` — queries enabled scheduled configs | Phase 10 | More targeted; only processes configs with a cron set |

**Deprecated/outdated after this phase:**
- `worker.StartScheduler`: Remove entirely. Replaced by `StartPerMinuteScheduler`.
- `cfg.BriefingSchedule` and `cfg.BriefingTimezone` config fields: No longer read by the scheduler (may be removed or left as dead config).
- `TaskScheduledBriefingGeneration` task constant and handler: Remove from `tasks.go` and `worker.go`. (The old "brief all users" mechanism is superseded by per-plugin per-user scheduling.)

---

## What Already Exists (Critical Context for Planner)

The following are already built and must not be rebuilt:

| Already Built | Location | Relevant to Phase 10 |
|---------------|----------|----------------------|
| `Plugin`, `UserPluginConfig`, `PluginRun` models | `internal/plugins/models.go` | `UserPluginConfig` needs 2 new columns |
| `EnqueueExecutePlugin()` | `internal/worker/tasks.go` | Used by per-minute scheduler to fire executions |
| `redis.Client` pattern (go-redis/v9) | `internal/streams/producer.go` | Same pattern for scheduler Redis cache |
| `golang-migrate` SQL migrations | `internal/database/migrations/` | Next file is `000006_*.sql` |
| `StartScheduler` in `worker/scheduler.go` | `internal/worker/scheduler.go` | **Delete and replace** |
| `TaskScheduledBriefingGeneration` + handler | `internal/worker/tasks.go`, `worker.go` | **Delete** (global approach retired) |
| `handleExecutePlugin` | `internal/worker/worker.go` | Keep as-is; per-minute scheduler calls EnqueueExecutePlugin which triggers this |
| `User.Timezone` field | `internal/models/user.go` | Can be used as fallback if `UserPluginConfig.Timezone` is empty |

---

## Open Questions

1. **Should `UserPluginConfig.Timezone` fall back to `User.Timezone`?**
   - What we know: `User` has a `Timezone` field (`gorm:"not null;default:'UTC'"`). `UserPluginConfig` does not yet have one.
   - What's unclear: Whether a user expects their plugin to use the same timezone as their user profile by default, or whether per-plugin timezone override is even needed initially.
   - Recommendation: Default `UserPluginConfig.Timezone` to `''` (empty string) and fall back to `User.Timezone` at evaluation time. Add Preload of User in the scheduler query.

2. **Does the per-minute Asynq task need its own queue?**
   - What we know: All tasks currently run on the `default` queue with concurrency 5.
   - What's unclear: Whether a high-priority `scheduler` queue would be better to prevent the per-minute meta-task from being delayed by plugin:execute tasks.
   - Recommendation: Assign `TaskPerMinuteScheduler` to a separate `critical` queue with higher priority in the Asynq server config. This keeps the scheduler from being backed up behind long-running plugin tasks.

3. **What happens to the existing `TaskScheduledBriefingGeneration` for the old briefing flow?**
   - What we know: Phase 10 retires the global scheduler. The old briefing generation flow (webhook-based, not CrewAI) used this task.
   - What's unclear: Whether any user still depends on the old briefing generation path.
   - Recommendation: Delete `TaskScheduledBriefingGeneration` and its handler in this phase, per the requirement "Global cron scheduler removed."

---

## Sources

### Primary (HIGH confidence)
- `github.com/robfig/cron/v3` (in go.mod as transitive dep) — `Parser.Parse`, `Schedule.Next`, `CRON_TZ=` prefix: verified via pkg.go.dev and parser.go source
- `github.com/hibiken/asynq` v0.26.0 (in go.mod as direct dep) — `asynq.NewScheduler`, `scheduler.Register`, `asynq.Unique`: verified via existing codebase usage in `scheduler.go`
- `github.com/redis/go-redis/v9` v9.17.3 (in go.mod as direct dep) — `HSet`, `HGet`: verified via existing usage in `producer.go`
- Codebase inspection — `internal/plugins/models.go`, `internal/worker/scheduler.go`, `internal/worker/worker.go`, `internal/worker/tasks.go`, `internal/database/migrations/000005_create_plugins.up.sql`, `internal/models/user.go`

### Secondary (MEDIUM confidence)
- [robfig/cron v3 pkg.go.dev](https://pkg.go.dev/github.com/robfig/cron/v3) — Schedule interface, CRON_TZ documentation
- [Redis HSET docs](https://redis.io/docs/latest/commands/hset/) — confirmed O(1) hash field operations

### Tertiary (LOW confidence)
- WebSearch results on "per-minute scheduler database-backed pattern" — general architectural patterns, not verified against a specific authoritative source

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all libraries already in go.mod; no new dependencies
- Architecture (per-minute Asynq task): HIGH — directly follows existing patterns in `scheduler.go` and `worker.go`
- Cron evaluation with robfig: HIGH — verified parser.go source and pkg.go.dev docs
- Redis cache pattern: HIGH — same client, standard HSET/HGET
- Pitfalls: MEDIUM — cold-cache mass-fire and timezone bugs are well-known; duplicate-run race condition verified via Asynq Unique semantics

**Research date:** 2026-02-18
**Valid until:** 2026-03-18 (robfig/cron v3 is stable; Asynq v0.26 is pinned; no fast-moving dependencies)
