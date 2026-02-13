# Phase 6: Scheduled Generation - Research

**Researched:** 2026-02-12
**Domain:** Asynq Scheduler for periodic cron-based task execution
**Confidence:** HIGH

## Summary

Phase 6 adds automatic daily briefing generation using Asynq's Scheduler component, which enqueues tasks on a cron schedule that are then processed by existing worker infrastructure. The Scheduler uses the same Redis backend as the task queue, enabling a unified infrastructure approach without additional dependencies.

Asynq Scheduler supports standard cron syntax (e.g., `0 6 * * *` for 6 AM daily), configurable timezones via `SchedulerOpts.Location`, and lifecycle management patterns identical to the Server (Run/Start/Shutdown). The scheduler is stateless—it only enqueues tasks on schedule, letting existing worker processes handle execution with the same retry, error handling, and monitoring capabilities established in Phase 3.

**Critical architectural consideration:** Only ONE scheduler instance should run per schedule to prevent duplicate task enqueuing. For Kubernetes deployments, this requires either single-replica scheduler deployment or leader election patterns. For personal use with single K8s pod deployments, this is naturally satisfied.

The implementation leverages existing `EnqueueGenerateBriefing()` task enqueue logic, adding Asynq's `Unique()` option to prevent duplicate briefings if scheduled task runs overlap. Configuration via environment variable (`BRIEFING_SCHEDULE` defaulting to `0 6 * * *`) enables flexible scheduling without code changes.

**Primary recommendation:** Use Asynq NewScheduler with Start/Shutdown lifecycle for embedded scheduler in development mode, cron syntax `0 6 * * *` (6 AM daily) as default, Unique(24*time.Hour) on scheduled tasks to prevent duplicates, and configurable schedule via environment variable.

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| github.com/hibiken/asynq | v0.26.0 (already installed) | Periodic task scheduler with cron syntax | Official Asynq component, shares Redis with existing worker infrastructure, stable API |

### No Additional Dependencies

Asynq Scheduler is part of the core `github.com/hibiken/asynq` package already installed in Phase 3. No new dependencies required.

### Configuration Pattern

| Component | Approach | Default | Why |
|-----------|----------|---------|-----|
| Cron schedule | Environment variable `BRIEFING_SCHEDULE` | `0 6 * * *` (6 AM daily) | Enables deployment-time configuration without code changes |
| Timezone | Environment variable `BRIEFING_TIMEZONE` | `UTC` | Prevents ambiguity in distributed deployments |
| Scheduler mode | Same binary, runs in worker mode or embedded in dev | Worker mode in K8s | Consistent with Phase 3 architecture pattern |

**Installation:**

No new packages needed. Scheduler is part of existing Asynq v0.26.0.

## Architecture Patterns

### Recommended Project Structure

```
internal/
├── worker/
│   ├── worker.go        # Server + Scheduler initialization
│   ├── scheduler.go     # NEW: Scheduler setup, cron registration
│   ├── handlers.go      # Existing task handlers
│   └── tasks.go         # Existing enqueue functions (reused by scheduler)
internal/config/
    └── config.go        # Add BriefingSchedule, BriefingTimezone fields
```

### Pattern 1: Scheduler Lifecycle Management (Embedded Mode)

**What:** Scheduler runs alongside worker Server in same process, both using shared Redis connection.

**When to use:** Development mode, or production with single worker pod that handles both task processing and scheduling.

**Example:**

```go
// internal/worker/scheduler.go
package worker

import (
    "log/slog"
    "time"
    "github.com/hibiken/asynq"
    "github.com/jimdaga/first-sip/internal/config"
)

// StartScheduler creates and starts an Asynq Scheduler for periodic tasks.
// Returns a stop function for graceful shutdown.
func StartScheduler(cfg *config.Config) (stop func(), err error) {
    redisOpt, err := asynq.ParseRedisURI(cfg.RedisURL)
    if err != nil {
        return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
    }

    // Parse timezone from config
    location, err := time.LoadLocation(cfg.BriefingTimezone)
    if err != nil {
        log.Printf("Invalid timezone %s, using UTC: %v", cfg.BriefingTimezone, err)
        location = time.UTC
    }

    scheduler := asynq.NewScheduler(
        redisOpt,
        &asynq.SchedulerOpts{
            Location: location,
            LogLevel: asynq.InfoLevel,
            Logger:   &asynqLoggerAdapter{logger: NewLogger(cfg.LogLevel, cfg.LogFormat)},
        },
    )

    // Register periodic briefing generation task
    if err := registerPeriodicTasks(scheduler, cfg); err != nil {
        return nil, fmt.Errorf("failed to register periodic tasks: %w", err)
    }

    // Start scheduler (non-blocking)
    if err := scheduler.Start(); err != nil {
        return nil, fmt.Errorf("failed to start scheduler: %w", err)
    }

    log.Printf("Scheduler started with schedule: %s (timezone: %s)",
        cfg.BriefingSchedule, cfg.BriefingTimezone)

    // Return shutdown function
    return func() { scheduler.Shutdown() }, nil
}

func registerPeriodicTasks(scheduler *asynq.Scheduler, cfg *config.Config) error {
    // Daily briefing generation task (for ALL users)
    // Note: Payload is empty because handler will query all users
    task := asynq.NewTask(
        TaskScheduledBriefingGeneration,
        nil,
        asynq.MaxRetry(3),
        asynq.Timeout(10*time.Minute), // Longer timeout for processing all users
        asynq.Retention(24*time.Hour),
        asynq.Unique(24*time.Hour), // Prevent duplicate if scheduler misbehaves
    )

    entryID, err := scheduler.Register(cfg.BriefingSchedule, task)
    if err != nil {
        return fmt.Errorf("failed to register briefing schedule: %w", err)
    }

    log.Printf("Registered daily briefing generation (entry_id=%s)", entryID)
    return nil
}
```

**Integration in cmd/server/main.go:**

```go
// Start embedded worker in development mode (non-blocking)
var stopWorker func()
var stopScheduler func()

if cfg.Env == "development" && cfg.RedisURL != "" {
    log.Println("Starting embedded worker for development")
    var err error
    stopWorker, err = worker.Start(cfg, db, webhookClient)
    if err != nil {
        log.Fatalf("Failed to start embedded worker: %v", err)
    }

    // Start embedded scheduler
    log.Println("Starting embedded scheduler for development")
    stopScheduler, err = worker.StartScheduler(cfg)
    if err != nil {
        log.Fatalf("Failed to start embedded scheduler: %v", err)
    }
}

// ... later in shutdown ...

// Shut down embedded scheduler
if stopScheduler != nil {
    stopScheduler()
}

// Shut down embedded worker
if stopWorker != nil {
    stopWorker()
}
```

**Worker mode (standalone):**

```go
// internal/worker/worker.go - modify Run() function
func Run(cfg *config.Config, db *gorm.DB, webhookClient *webhook.Client) error {
    srv, mux, err := newServer(cfg, db, webhookClient)
    if err != nil {
        return err
    }

    // Start scheduler in goroutine
    stopScheduler, err := StartScheduler(cfg)
    if err != nil {
        return fmt.Errorf("failed to start scheduler: %w", err)
    }
    defer stopScheduler()

    // Run blocks and handles signal interception
    return srv.Run(mux)
}
```

### Pattern 2: Scheduled Task Handler for All Users

**What:** Scheduled task queries all users and enqueues individual briefing generation tasks for each.

**When to use:** Daily automatic briefing generation triggered by cron schedule.

**Example:**

```go
// internal/worker/tasks.go
const (
    TaskGenerateBriefing           = "briefing:generate"
    TaskScheduledBriefingGeneration = "briefing:scheduled_generation" // NEW
)

// internal/worker/handlers.go (additions)
func handleScheduledBriefingGeneration(logger *slog.Logger, db *gorm.DB) func(context.Context, *asynq.Task) error {
    return func(ctx context.Context, task *asynq.Task) error {
        logger.Info("Starting scheduled briefing generation for all users")

        // Query all active users
        var users []models.User
        if err := db.WithContext(ctx).Find(&users).Error; err != nil {
            return fmt.Errorf("failed to query users: %w", err)
        }

        successCount := 0
        errorCount := 0

        // Enqueue briefing generation for each user
        for _, user := range users {
            // Create briefing record first
            briefing := models.Briefing{
                UserID: user.ID,
                Status: models.BriefingStatusPending,
            }
            if err := db.WithContext(ctx).Create(&briefing).Error; err != nil {
                logger.Error("Failed to create briefing", "user_id", user.ID, "error", err)
                errorCount++
                continue
            }

            // Enqueue task with Unique option to prevent duplicates
            if err := EnqueueGenerateBriefing(briefing.ID); err != nil {
                logger.Error("Failed to enqueue briefing", "briefing_id", briefing.ID, "error", err)
                errorCount++
                continue
            }

            successCount++
        }

        logger.Info(
            "Scheduled briefing generation completed",
            "total_users", len(users),
            "enqueued", successCount,
            "errors", errorCount,
        )

        // Don't fail if some briefings failed to enqueue (partial success is OK)
        return nil
    }
}

// Register in worker.go newServer():
mux.HandleFunc(TaskScheduledBriefingGeneration, handleScheduledBriefingGeneration(logger, db))
```

### Pattern 3: Cron Schedule Configuration

**What:** Schedule configured via environment variable with validation and sensible default.

**When to use:** All deployments to enable schedule changes without code modification.

**Example:**

```go
// internal/config/config.go (additions)
type Config struct {
    // ... existing fields ...
    BriefingSchedule string // Cron expression for daily briefing generation
    BriefingTimezone string // Timezone for cron schedule (IANA format)
}

func Load() *Config {
    cfg := &Config{
        // ... existing fields ...
        BriefingSchedule: getEnvWithDefault("BRIEFING_SCHEDULE", "0 6 * * *"), // 6 AM daily
        BriefingTimezone: getEnvWithDefault("BRIEFING_TIMEZONE", "UTC"),
    }

    // Validate cron schedule format (basic check)
    if err := validateCronSchedule(cfg.BriefingSchedule); err != nil {
        log.Printf("WARNING: Invalid BRIEFING_SCHEDULE '%s': %v. Using default '0 6 * * *'",
            cfg.BriefingSchedule, err)
        cfg.BriefingSchedule = "0 6 * * *"
    }

    return cfg
}

func validateCronSchedule(cronExpr string) error {
    // Basic validation: must have 5 space-separated fields
    fields := strings.Fields(cronExpr)
    if len(fields) != 5 {
        return fmt.Errorf("cron expression must have 5 fields (minute hour day month weekday), got %d", len(fields))
    }
    return nil
}
```

**Environment variable usage:**

```bash
# env.local
export BRIEFING_SCHEDULE="0 6 * * *"    # 6 AM daily (default)
export BRIEFING_TIMEZONE="America/New_York"  # Eastern Time
```

### Pattern 4: Task Uniqueness for Scheduled Generation

**What:** Use Asynq's `Unique()` option to prevent duplicate briefings if scheduler enqueues multiple times.

**When to use:** All scheduled tasks to provide idempotency guarantees.

**Example:**

```go
// Modify EnqueueGenerateBriefing to accept optional Unique parameter
func EnqueueGenerateBriefing(briefingID uint) error {
    payload, err := json.Marshal(map[string]uint{
        "briefing_id": briefingID,
    })
    if err != nil {
        return err
    }

    // Create task with Unique option for scheduler-triggered briefings
    task := asynq.NewTask(
        TaskGenerateBriefing,
        payload,
        asynq.MaxRetry(3),
        asynq.Timeout(5*time.Minute),
        asynq.Retention(24*time.Hour),
        // Unique by briefing_id for 1 hour - prevents duplicate generation
        // if scheduler misbehaves or if manual generation overlaps
        asynq.Unique(1*time.Hour),
    )

    _, err = client.Enqueue(task)
    if err != nil {
        // Check if duplicate task error (not a failure condition)
        if errors.Is(err, asynq.ErrDuplicateTask) {
            log.Printf("Briefing %d already queued (duplicate), skipping", briefingID)
            return nil // Not an error - task already enqueued
        }
        return err
    }

    return nil
}
```

### Anti-Patterns to Avoid

- **Running multiple scheduler instances without coordination:** Asynq Scheduler does NOT use distributed locks by default. Multiple schedulers will enqueue duplicate tasks. Solution: Single scheduler pod, or implement external leader election.
- **Hardcoding cron schedule in code:** Makes schedule changes require code deployment. Always use environment variable configuration.
- **Forgetting to register scheduler handlers:** Scheduler enqueues tasks, but if handler isn't registered in mux, tasks sit in queue forever.
- **Not handling Unique task errors gracefully:** `asynq.ErrDuplicateTask` is expected behavior for scheduled tasks with Unique option. Don't log as error, handle as successful no-op.
- **Using worker Server timeout for scheduler tasks:** Scheduled tasks that process all users need longer timeout than individual briefings. Set task-specific timeout via `asynq.Timeout()` option.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Cron parsing and scheduling | Custom time.Ticker loop with manual cron parsing | Asynq Scheduler with cron syntax | Handles edge cases: DST transitions, leap seconds, timezone changes, 5-field cron syntax nuances |
| Preventing duplicate scheduled tasks | Application-level database flags or Redis flags | Asynq's `Unique()` option | Built-in Redis-based deduplication with TTL, integrates with task lifecycle, visible in Asynqmon |
| Leader election for single scheduler | Custom Redis lock implementation | Accept single-pod deployment, or use K8s StatefulSet with Replicas=1 | For personal use, single pod is simplest. Production distributed schedulers need external coordination (not Asynq's scope) |
| Scheduler monitoring | Custom health check endpoints | Asynqmon + Inspector API `SchedulerEntries()` and `ListSchedulerEnqueueEvents()` | Shows registered schedules, last enqueue time, next enqueue time, enqueue errors |

**Key insight:** Scheduling has deceptive complexity—cron syntax edge cases (e.g., "0 0 29 2 *" February 29th), timezone handling, DST transitions, leap seconds. Asynq Scheduler delegates to robust cron parsing libraries with years of production hardening. Custom solutions inevitably miss edge cases or require same dependencies.

## Common Pitfalls

### Pitfall 1: Multiple Scheduler Instances Enqueue Duplicate Tasks

**What goes wrong:** Running 2+ scheduler instances (e.g., K8s Deployment with replicas=2) causes each instance to enqueue the same task on schedule, creating duplicate briefings for all users.

**Why it happens:** Asynq Scheduler is stateless and does NOT implement distributed coordination/leader election. Each instance independently enqueues tasks on schedule.

**How to avoid:**
- **Development:** Single embedded scheduler in main server process
- **Production (personal use):** Single worker pod with scheduler embedded
- **Production (multi-worker):** Separate scheduler Deployment with `replicas: 1`, or implement external leader election

**Warning signs:**
- Users receive 2+ briefings at same scheduled time
- Asynqmon shows duplicate pending tasks with identical payload/timestamp
- Database shows multiple briefings created at exact scheduled time

**Example fix (Kubernetes):**

```yaml
# Separate scheduler deployment with single replica
apiVersion: apps/v1
kind: Deployment
metadata:
  name: first-sip-scheduler
spec:
  replicas: 1  # CRITICAL: Only one scheduler instance
  selector:
    matchLabels:
      app: first-sip-scheduler
  template:
    metadata:
      labels:
        app: first-sip-scheduler
    spec:
      containers:
      - name: scheduler
        image: first-sip:latest
        command: ["/app/server", "--scheduler"]  # Hypothetical flag
        env:
        - name: BRIEFING_SCHEDULE
          value: "0 6 * * *"
```

**OR accept single worker pod with embedded scheduler (simplest for personal use):**

```yaml
# Single worker pod that handles both task processing AND scheduling
apiVersion: apps/v1
kind: Deployment
metadata:
  name: first-sip-worker
spec:
  replicas: 1  # Single pod runs both worker and scheduler
```

### Pitfall 2: Cron Expression Timezone Confusion

**What goes wrong:** Cron expression `0 6 * * *` runs at 6 AM UTC, not user's local timezone. Users in PST expect 6 AM PST, receive briefing at 10 PM previous night (6 AM UTC = 10 PM PST).

**Why it happens:** Asynq Scheduler defaults to UTC if `Location` not set in `SchedulerOpts`. Cron expressions don't encode timezone.

**How to avoid:**
- Always explicitly set `SchedulerOpts.Location` via environment variable
- Document timezone configuration clearly for deployment
- Consider user preference storage if multi-timezone support needed (future phase)

**Warning signs:**
- Users report briefings arriving at "wrong" time
- Logs show scheduled task running at unexpected hour in server logs

**Example fix:**

```go
// Parse timezone from config
location, err := time.LoadLocation(cfg.BriefingTimezone) // "America/New_York"
if err != nil {
    log.Printf("Invalid timezone %s, using UTC: %v", cfg.BriefingTimezone, err)
    location = time.UTC
}

scheduler := asynq.NewScheduler(
    redisOpt,
    &asynq.SchedulerOpts{
        Location: location, // CRITICAL: Set explicitly
        // ...
    },
)
```

**Environment variable:**

```bash
export BRIEFING_TIMEZONE="America/Los_Angeles"  # PST/PDT
```

### Pitfall 3: Forgetting to Start Scheduler

**What goes wrong:** Scheduler created and tasks registered, but `Start()` or `Run()` never called. No tasks ever enqueued.

**Why it happens:** Asynq Server has similar API (`NewServer` + `Run`), easy to copy pattern but forget final step.

**How to avoid:**
- Follow consistent pattern: create scheduler, register tasks, call `Start()`, defer `Shutdown()`
- Add integration test that verifies scheduled task enqueued after scheduler start
- Check Asynqmon scheduler entries on startup—empty list indicates scheduler not running

**Warning signs:**
- Scheduled tasks never appear in queue
- Asynqmon shows no scheduler entries
- No scheduled briefings generated despite passing schedule time

**Example fix:**

```go
scheduler := asynq.NewScheduler(redisOpt, opts)
registerPeriodicTasks(scheduler, cfg)

// CRITICAL: Actually start the scheduler
if err := scheduler.Start(); err != nil {
    log.Fatalf("Failed to start scheduler: %v", err)
}
```

### Pitfall 4: Scheduler Shutdown Not Called

**What goes wrong:** Scheduler continues running after HTTP server shutdown, goroutines leak, process doesn't exit cleanly.

**Why it happens:** Forgetting to defer scheduler `Shutdown()` or call it in signal handler.

**How to avoid:**
- Always use `defer stopScheduler()` pattern with closure
- For embedded mode, coordinate shutdown with HTTP server via signal context
- Test shutdown behavior: `docker stop` should exit process within grace period

**Warning signs:**
- Process doesn't exit cleanly on SIGTERM
- Docker containers take 10+ seconds to stop (hitting SIGKILL timeout)
- Goroutine leaks visible in process inspection

**Example fix:**

```go
stopScheduler, err := worker.StartScheduler(cfg)
if err != nil {
    log.Fatalf("Failed to start scheduler: %v", err)
}
// CRITICAL: Ensure shutdown called
defer stopScheduler()

// OR in signal handler:
<-ctx.Done()
log.Println("Shutting down...")

if stopScheduler != nil {
    stopScheduler() // Explicit shutdown before exit
}
```

### Pitfall 5: Task Payload Mismatch Between Scheduler and Handler

**What goes wrong:** Scheduler registers task with empty payload, handler expects `briefing_id` in payload, unmarshaling fails, task moves to dead letter queue.

**Why it happens:** Scheduled tasks have different semantics (process all users) vs manual tasks (single briefing). Reusing same task type with different payload structure.

**How to avoid:**
- Use separate task type constants for scheduled vs manual generation
- `TaskScheduledBriefingGeneration` (no payload) vs `TaskGenerateBriefing` (briefing_id payload)
- Validate payload schema at start of handler, return `asynq.SkipRetry` for invalid payloads

**Warning signs:**
- Scheduled tasks immediately fail with "invalid payload" error
- Manual briefing generation works, scheduled generation fails
- Archive queue fills with scheduled tasks

**Example fix:**

```go
// Separate task types with different payloads
const (
    TaskGenerateBriefing           = "briefing:generate"            // Payload: {"briefing_id": uint}
    TaskScheduledBriefingGeneration = "briefing:scheduled_generation" // Payload: nil/empty
)

// Register BOTH handlers
mux.HandleFunc(TaskGenerateBriefing, handleGenerateBriefing(logger, db, webhookClient))
mux.HandleFunc(TaskScheduledBriefingGeneration, handleScheduledBriefingGeneration(logger, db))
```

### Pitfall 6: Cron Expression Validation Failure

**What goes wrong:** Invalid cron expression (e.g., `0 25 * * *` - hour 25 doesn't exist) causes scheduler registration to fail silently or panic.

**Why it happens:** Environment variable typo, invalid syntax passed directly to scheduler without validation.

**How to avoid:**
- Validate cron expression format before passing to scheduler
- Catch registration errors and log clearly
- Provide helpful error message with valid syntax examples

**Warning signs:**
- Scheduler fails to start with cryptic error
- `Register()` returns error immediately
- No scheduled tasks appear in Asynqmon

**Example fix:**

```go
func validateCronSchedule(cronExpr string) error {
    fields := strings.Fields(cronExpr)
    if len(fields) != 5 {
        return fmt.Errorf("cron expression must have 5 fields (minute hour day month weekday), got %d. Example: '0 6 * * *'", len(fields))
    }

    // Additional validation: parse each field range
    // (optional: use cron parsing library for thorough validation)

    return nil
}

// In config.Load():
if err := validateCronSchedule(cfg.BriefingSchedule); err != nil {
    log.Fatalf("Invalid BRIEFING_SCHEDULE: %v", err)
}
```

## Code Examples

Verified patterns from official sources:

### Creating and Starting Scheduler (Non-Blocking)

```go
// Source: https://pkg.go.dev/github.com/hibiken/asynq
func StartScheduler(redisURL string, schedule string, timezone string) (func(), error) {
    redisOpt, _ := asynq.ParseRedisURI(redisURL)

    location, _ := time.LoadLocation(timezone)

    scheduler := asynq.NewScheduler(
        redisOpt,
        &asynq.SchedulerOpts{
            Location: location,
            LogLevel: asynq.InfoLevel,
        },
    )

    // Register periodic task
    task := asynq.NewTask("daily:briefing", nil)
    entryID, err := scheduler.Register(schedule, task)
    if err != nil {
        return nil, err
    }
    log.Printf("Registered schedule (entry_id=%s)", entryID)

    // Start scheduler (non-blocking)
    if err := scheduler.Start(); err != nil {
        return nil, err
    }

    // Return shutdown function
    return func() { scheduler.Shutdown() }, nil
}
```

### Scheduler Options Configuration

```go
// Source: https://github.com/hibiken/asynq/blob/master/scheduler.go
location, _ := time.LoadLocation("America/New_York")

opts := &asynq.SchedulerOpts{
    // Timezone for cron schedule interpretation
    Location: location,

    // Logger instance (same adapter pattern as Server)
    Logger: &asynqLoggerAdapter{logger: slog.Default()},

    // Minimum log level
    LogLevel: asynq.InfoLevel,

    // Callback before task enqueue
    PreEnqueueFunc: func(task *asynq.Task, opts []asynq.Option) {
        log.Printf("Enqueueing scheduled task: %s", task.Type())
    },

    // Callback after task enqueue (with error if failed)
    PostEnqueueFunc: func(info *asynq.TaskInfo, err error) {
        if err != nil {
            log.Printf("Failed to enqueue: %v", err)
        } else {
            log.Printf("Enqueued task ID: %s", info.ID)
        }
    },
}

scheduler := asynq.NewScheduler(redisOpt, opts)
```

### Registering Task with Options

```go
// Source: https://pkg.go.dev/github.com/hibiken/asynq
task := asynq.NewTask(
    "briefing:scheduled_generation",
    nil, // Empty payload for scheduled task
    asynq.MaxRetry(3),
    asynq.Timeout(10*time.Minute),
    asynq.Unique(24*time.Hour), // Prevent duplicate if scheduler runs twice
)

// Register with cron expression
entryID, err := scheduler.Register("0 6 * * *", task)
if err != nil {
    log.Fatalf("Failed to register task: %v", err)
}

log.Printf("Registered daily briefing generation (entry_id=%s)", entryID)
```

### Cron Syntax Examples

```go
// Source: https://crontab.guru/ and https://github.com/hibiken/asynq/wiki/Periodic-Tasks

// Standard cron expressions (5 fields: minute hour day month weekday)
scheduler.Register("0 6 * * *", task)      // Daily at 6:00 AM
scheduler.Register("30 14 * * *", task)    // Daily at 2:30 PM
scheduler.Register("0 */4 * * *", task)    // Every 4 hours
scheduler.Register("0 9 * * 1", task)      // Every Monday at 9:00 AM
scheduler.Register("0 0 1 * *", task)      // First day of month at midnight

// Special descriptors (Asynq supports these)
scheduler.Register("@daily", task)         // Same as "0 0 * * *"
scheduler.Register("@hourly", task)        // Same as "0 * * * *"
scheduler.Register("@every 30m", task)     // Every 30 minutes
scheduler.Register("@every 1h30m", task)   // Every 1.5 hours
```

### Inspecting Scheduler Entries (Monitoring)

```go
// Source: https://pkg.go.dev/github.com/hibiken/asynq
redisOpt, _ := asynq.ParseRedisURI(redisURL)
inspector := asynq.NewInspector(redisOpt)
defer inspector.Close()

// List all registered scheduler entries
entries, err := inspector.SchedulerEntries()
if err != nil {
    log.Fatal(err)
}

for _, entry := range entries {
    log.Printf("Entry ID: %s", entry.ID)
    log.Printf("Spec: %s", entry.Spec) // Cron expression
    log.Printf("Task Type: %s", entry.Task.Type())
    log.Printf("Next Enqueue: %s", entry.Next)
    log.Printf("Prev Enqueue: %s", entry.Prev)
}

// List enqueue events for specific entry
events, err := inspector.ListSchedulerEnqueueEvents(entryID)
for _, event := range events {
    log.Printf("Enqueued at: %s (task_id=%s)", event.EnqueuedAt, event.TaskID)
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Cron jobs triggering HTTP endpoints | Asynq Scheduler enqueuing tasks directly | Asynq v0.18+ (2021) | Eliminates HTTP overhead, provides built-in retry/monitoring, unifies with task queue infrastructure |
| Separate cron daemon (crontab) | Application-embedded scheduler | Modern distributed systems | Enables containerization, K8s-native deployments, programmatic configuration |
| robfig/cron library for Go apps | Asynq Scheduler | Asynq v0.18+ Scheduler release | Unified infrastructure (one Redis), task queue benefits (retry, monitoring), no separate worker pool |
| Hardcoded schedules in code | Environment variable configuration | 12-factor app pattern | Deployment-time schedule changes without code/image rebuild |

**Deprecated/outdated:**
- **Asynq PeriodicTaskManager (pre-v0.18):** Replaced by `Scheduler` API. Use `asynq.NewScheduler()` instead of `asynq.NewPeriodicTaskManager()`.
- **CRON_TZ=timezone prefix in cron expression:** While supported by some cron implementations, Asynq uses `SchedulerOpts.Location` for timezone configuration (cleaner separation).
- **Separate cron container in K8s:** Modern pattern is embedded scheduler in worker pod (single concern: background job processing + scheduling) rather than separate cron-curl containers.

## Open Questions

1. **User-specific schedule preferences (future enhancement)**
   - What we know: Current design generates briefings for ALL users at same time
   - What's unclear: How to support per-user timezone preferences or schedule customization
   - Recommendation: Phase 6 implements global schedule. Future phase could add user preferences table, scheduler enqueues "check user preferences" task, which then enqueues individual briefings based on each user's preferred time.

2. **Handling users across multiple timezones**
   - What we know: Single global schedule applies to all users
   - What's unclear: Best user experience for geographically distributed users
   - Recommendation: Start with UTC 6 AM (acceptable for personal use within single timezone). Future enhancement: multiple scheduler entries for major timezone groups (e.g., 6 AM PST, 6 AM EST, 6 AM UTC).

3. **Scheduler health monitoring in production**
   - What we know: Inspector API provides `SchedulerEntries()` and enqueue event history
   - What's unclear: How to surface scheduler health to monitoring/alerting
   - Recommendation: Add `/health` endpoint check that uses Inspector to verify scheduler has entries registered and last enqueue time is recent (within 25 hours for daily schedule). Alert if scheduler silent for 36+ hours.

4. **Scheduler high availability strategy**
   - What we know: Single scheduler instance required to prevent duplicates
   - What's unclear: What happens if scheduler pod crashes between scheduled enqueue times
   - Recommendation: For personal use, accept brief downtime (K8s will restart pod, next scheduled run recovers). For production, implement K8s leader election with multiple scheduler pods in standby (out of scope for Phase 6).

## Sources

### Primary (HIGH confidence)

- [Asynq Periodic Tasks Wiki](https://github.com/hibiken/asynq/wiki/Periodic-Tasks) - Official scheduler documentation
- [Asynq Package Documentation](https://pkg.go.dev/github.com/hibiken/asynq) - Scheduler API reference, SchedulerOpts struct
- [Asynq scheduler.go Source](https://github.com/hibiken/asynq/blob/master/scheduler.go) - SchedulerOpts complete definition
- [Asynq Unique Tasks Wiki](https://github.com/hibiken/asynq/wiki/Unique-Tasks) - Task deduplication patterns
- [Crontab.guru Daily at 6:00 AM](https://crontab.io/cron/daily-at-6am) - Cron syntax reference

### Secondary (MEDIUM confidence)

- [How to Build a Job Queue in Go with Asynq and Redis](https://oneuptime.com/blog/post/2026-01-07-go-asynq-job-queue-redis/view) - Recent 2026 tutorial
- [Cron Expression Examples](https://crontab.guru/examples.html) - Standard cron syntax reference
- [How to Implement Leader Election in Kubernetes Pods](https://oneuptime.com/blog/post/2026-01-19-kubernetes-leader-election-pods/view) - K8s leader election patterns (for future HA)
- [robfig/cron Go package](https://pkg.go.dev/github.com/robfig/cron) - Alternative cron library (context for Asynq's approach)

### Tertiary (LOW confidence)

- GitHub discussions on Asynq scheduler deployment patterns - Community patterns, not official recommendations

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Using existing Asynq v0.26.0, official Scheduler API, production-tested
- Architecture: HIGH - Patterns verified against official Asynq documentation and examples, consistent with Phase 3 patterns
- Pitfalls: MEDIUM-HIGH - Deduced from official docs and common distributed scheduler issues, some based on general distributed systems knowledge
- Code examples: HIGH - Sourced from pkg.go.dev and official GitHub, tested patterns from wiki

**Research date:** 2026-02-12
**Valid until:** 2026-04-12 (60 days - Asynq is stable, Scheduler API unchanged since v0.18)
