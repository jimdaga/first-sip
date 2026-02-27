---
phase: 10-per-user-scheduling
plan: 01
subsystem: worker
tags: [asynq, cron, redis, scheduler, gorm, migrations, timezone]

# Dependency graph
requires:
  - phase: 09-crewai-sidecar
    provides: EnqueueExecutePlugin, streams.Publisher, PluginRun model, worker infrastructure
  - phase: 08-plugin-architecture
    provides: UserPluginConfig model, Plugin model, plugins package
provides:
  - Migration 000006 adding cron_expression and timezone columns to user_plugin_configs
  - Partial index idx_user_plugin_configs_scheduled for scheduler query performance
  - ValidateCronExpression function using robfig/cron/v3 parser
  - StartPerMinuteScheduler replaces StartScheduler (Asynq scheduler firing every minute)
  - handlePerMinuteScheduler queries DB and enqueues plugin:execute for due user-plugin pairs
  - isDue with CRON_TZ timezone-aware cron evaluation and cold-cache protection
  - Redis HSET/HGET last-run cache (key: scheduler:last_run, field: userID:pluginID)
  - TaskPerMinuteScheduler constant ("scheduler:per_minute")
affects: [10-per-user-scheduling, 12-settings-ui]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Per-minute Asynq scheduler: fires once/minute, handler queries DB and dispatches due pairs"
    - "CRON_TZ prefix for timezone-aware cron parsing via robfig/cron/v3"
    - "Redis hash for last-run cache: HSET/HGET scheduler:last_run with userID:pluginID fields"
    - "Cold-cache protection: zero lastRunAt treated as one minute ago to prevent mass-fire"
    - "Partial DB index on (enabled, cron_expression) WHERE deleted_at IS NULL AND enabled AND cron_expression IS NOT NULL"

key-files:
  created:
    - internal/database/migrations/000006_add_schedule_to_user_plugin_configs.up.sql
    - internal/database/migrations/000006_add_schedule_to_user_plugin_configs.down.sql
  modified:
    - internal/plugins/models.go
    - internal/worker/scheduler.go
    - internal/worker/tasks.go
    - internal/worker/worker.go
    - cmd/server/main.go

key-decisions:
  - "Per-minute Asynq scheduler fires TaskPerMinuteScheduler; handler does DB query + dispatch (not O(users*plugins) Redis entries)"
  - "CRON_TZ=<timezone> prefix on cron expression for robfig/cron/v3 timezone-aware parsing"
  - "Cold-cache protection: zero lastRunAt set to time.Now().Add(-time.Minute) prevents mass-fire on startup"
  - "Redis hash (HSET/HGET) for last-run cache avoids DB round-trips per scheduler tick"
  - "TaskScheduledBriefingGeneration removed; handleScheduledBriefingGeneration removed (superseded by per-minute approach)"

patterns-established:
  - "isDue(cronExpr, timezone, lastRunAt): returns (bool, error) — empty expr = false, invalid TZ = UTC fallback"
  - "fieldKey(userID, pluginID): returns 'userID:pluginID' string for Redis hash fields"
  - "ValidateCronExpression(expr): uses shared cronParser, returns typed error for invalid expressions"

# Metrics
duration: 3min
completed: 2026-02-19
---

# Phase 10 Plan 01: Per-User Scheduling Engine Summary

**Database-backed per-minute scheduler replacing global cron: migration 000006 adds cron_expression+timezone to user_plugin_configs, isDue evaluates CRON_TZ-prefixed cron expressions per user timezone with Redis last-run cache preventing cold-start mass-fire**

## Performance

- **Duration:** 3 min
- **Started:** 2026-02-19T01:11:58Z
- **Completed:** 2026-02-19T01:15:18Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments
- Created migration 000006 with cron_expression (nullable) + timezone (NOT NULL DEFAULT 'UTC') columns and partial index for scheduler query performance
- Replaced StartScheduler + TaskScheduledBriefingGeneration with StartPerMinuteScheduler + TaskPerMinuteScheduler (fires every minute)
- Implemented isDue with CRON_TZ timezone-aware cron parsing, cold-cache protection, and UTC fallback for invalid timezones
- Built Redis HSET/HGET last-run cache (scheduler:last_run hash) to avoid redundant DB queries across ticks
- Added ValidateCronExpression to plugins/models.go for use at write time (Phase 12 settings UI)

## Task Commits

Each task was committed atomically:

1. **Task 1: Database migration and model update for schedule fields** - `e9161c3` (feat)
2. **Task 2: Per-minute scheduler with timezone-aware cron evaluation and Redis cache** - `31ea758` (feat)

**Plan metadata:** _(docs commit follows)_

## Files Created/Modified
- `internal/database/migrations/000006_add_schedule_to_user_plugin_configs.up.sql` - ALTER TABLE adds cron_expression, timezone columns and partial index
- `internal/database/migrations/000006_add_schedule_to_user_plugin_configs.down.sql` - DROP INDEX + DROP COLUMN IF EXISTS rollback
- `internal/plugins/models.go` - Added CronExpression, Timezone fields to UserPluginConfig; added ValidateCronExpression function
- `internal/worker/scheduler.go` - Complete replacement: StartPerMinuteScheduler, handlePerMinuteScheduler, isDue, Redis cache helpers
- `internal/worker/tasks.go` - Added TaskPerMinuteScheduler; removed TaskScheduledBriefingGeneration
- `internal/worker/worker.go` - Removed handleScheduledBriefingGeneration; registered TaskPerMinuteScheduler handler with dedicated Redis client
- `cmd/server/main.go` - Replaced StartScheduler calls with StartPerMinuteScheduler (Rule 3 auto-fix)

## Decisions Made
- `CRON_TZ=<timezone>` prefix on cron expressions lets robfig/cron/v3 evaluate schedules in the user's local timezone without a custom parser
- `lastRunAt.IsZero()` → set to `time.Now().Add(-time.Minute)` prevents cold-start mass-fire (a zero lastRunAt would make every job appear due)
- Redis hash key `scheduler:last_run` with field `userID:pluginID` is O(1) per lookup and survives scheduler restarts
- Dedicated Redis client for scheduler cache (`newSchedulerRedisClient`) keeps concerns separate from Asynq's internal connection

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Updated main.go and worker.go after removing StartScheduler**
- **Found during:** Task 2 (per-minute scheduler implementation)
- **Issue:** Deleting `StartScheduler` and `TaskScheduledBriefingGeneration` caused `cmd/server/main.go` to fail compilation (undefined: worker.StartScheduler) and `worker.go` to have a dead handler registration
- **Fix:** Updated both call sites in main.go to `worker.StartPerMinuteScheduler(cfg)`; removed `handleScheduledBriefingGeneration` from worker.go; registered `TaskPerMinuteScheduler` handler with a dedicated Redis client in `newServer`
- **Files modified:** cmd/server/main.go, internal/worker/worker.go
- **Verification:** `go build ./...` succeeds with no errors
- **Committed in:** 31ea758 (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (Rule 3 - blocking compile error)
**Impact on plan:** Required to keep codebase compiling. Plan 02 was expected to handle main.go wiring, but removing StartScheduler without updating main.go was immediately blocking. The wire-up done here is minimal (direct replacement of call sites only).

## Issues Encountered
None beyond the expected blocking compile error from removing StartScheduler.

## User Setup Required
None - no external service configuration required. Migration 000006 will run automatically on next `make db-up` / server start with migrations.

## Next Phase Readiness
- Plan 10-02: Wire StartPerMinuteScheduler into the new embedded-worker startup path with per-user scheduling context
- Plan 10-02 can assume: TaskPerMinuteScheduler handler is registered in worker.go; Redis last-run cache is operational; migration 000006 is ready to apply
- No blockers for Plan 10-02

## Self-Check: PASSED

All artifacts verified:
- FOUND: 000006 up.sql, 000006 down.sql
- FOUND: SUMMARY.md
- FOUND: commits e9161c3, 31ea758
- FOUND: StartPerMinuteScheduler, TaskPerMinuteScheduler, CronExpression, ValidateCronExpression, isDue, scheduler:last_run
- REMOVED: TaskScheduledBriefingGeneration (correct)

---
*Phase: 10-per-user-scheduling*
*Completed: 2026-02-19*
