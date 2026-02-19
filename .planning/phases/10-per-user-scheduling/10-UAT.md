---
status: complete
phase: 10-per-user-scheduling
source: 10-01-SUMMARY.md, 10-02-SUMMARY.md
started: 2026-02-19T15:30:00Z
updated: 2026-02-19T15:45:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Code compiles and tests pass
expected: `go build ./...` succeeds with no errors and `go test -race ./...` passes all tests.
result: pass

### 2. Migration 000006 exists with schedule columns
expected: File `internal/database/migrations/000006_add_schedule_to_user_plugin_configs.up.sql` exists and adds `cron_expression` (nullable text) and `timezone` (NOT NULL DEFAULT 'UTC') columns to `user_plugin_configs`, plus a partial index for scheduler performance.
result: pass

### 3. UserPluginConfig model has schedule fields
expected: `internal/plugins/models.go` has `CronExpression` (nullable/pointer string) and `Timezone` (string) fields on the `UserPluginConfig` struct. `ValidateCronExpression` function exists for cron syntax validation.
result: pass

### 4. Per-minute scheduler replaces global cron
expected: `internal/worker/scheduler.go` contains `StartPerMinuteScheduler` and `handlePerMinuteScheduler`. No references to the old `StartScheduler`, `TaskScheduledBriefingGeneration`, or `handleScheduledBriefingGeneration` exist anywhere in the codebase.
result: pass

### 5. Timezone-aware cron evaluation
expected: `isDue` function in `scheduler.go` prepends `CRON_TZ=<timezone>` to cron expressions before evaluating, falls back to UTC for invalid timezones, and treats zero `lastRunAt` as one minute ago (cold-cache protection).
result: pass

### 6. Redis last-run cache
expected: Scheduler uses Redis hash key `scheduler:last_run` with fields `userID:pluginID` for HSET/HGET operations to cache last-run times, avoiding DB queries on every tick.
result: pass

### 7. Critical queue for scheduler task
expected: `TaskPerMinuteScheduler` task is enqueued with `asynq.Queue("critical")`. Asynq server config has a Queues map with critical priority higher than default (e.g., critical:6, default:3).
result: pass

### 8. Global scheduler config removed
expected: `internal/config/config.go` has no `BriefingSchedule` or `BriefingTimezone` fields. The env vars `BRIEFING_SCHEDULE` and `BRIEFING_TIMEZONE` are no longer read.
result: pass

## Summary

total: 8
passed: 8
issues: 0
pending: 0
skipped: 0

## Gaps

[none]
