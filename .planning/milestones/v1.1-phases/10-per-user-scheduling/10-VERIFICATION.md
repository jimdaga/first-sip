---
phase: 10-per-user-scheduling
verified: 2026-02-26T19:52:52Z
status: passed
score: 7/7 must-haves verified
re_verification: false
gaps: []
human_verification:
  - test: "Enable a plugin for two users with different timezones and different cron schedules; wait for the scheduler to tick"
    expected: "Each user's plugin fires at the correct local time — Plugin A for User 1 (America/New_York) fires at the right New York time; Plugin B for User 2 (Asia/Tokyo) fires at the right Tokyo time"
    why_human: "Requires live DB with two distinct users, two distinct timezones set on users table, and a running Redis+Asynq environment to observe actual scheduler dispatch behavior"
---

# Phase 10: Per-User Scheduling Verification Report

**Phase Goal:** Replace the global cron scheduler with a per-minute Asynq task that reads per-user, per-plugin cron expressions from the database and dispatches plugin executions on schedule
**Verified:** 2026-02-26
**Status:** PASSED
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| #  | Truth | Status | Evidence |
|----|-------|--------|----------|
| 1  | `UserPluginConfig` has a `CronExpression` field for per-user per-plugin schedule configuration | VERIFIED | `internal/plugins/models.go` line 44: `CronExpression string \`gorm:"column:cron_expression"\`` (comment: "nullable — empty means no schedule"); migration `000006_add_schedule_to_user_plugin_configs.up.sql` adds `cron_expression VARCHAR(100)` column to `user_plugin_configs` |
| 2  | Timezone is sourced from the `users` table (not per-plugin), following Phase 14 migration | VERIFIED | `internal/plugins/models.go`: `UserPluginConfig` struct has NO `Timezone` field (migration 000009 dropped it); `internal/worker/scheduler.go` lines 91–95: `effectiveTimezone := cfg.User.Timezone; if effectiveTimezone == "" { effectiveTimezone = "UTC" }` — timezone read from `cfg.User.Timezone` (account level) |
| 3  | `handlePerMinuteScheduler` evaluates schedules via a single DB query (database-backed evaluation; no per-user Asynq cron registrations) | VERIFIED | `internal/worker/scheduler.go` lines 78–84: DB query `WHERE enabled = ? AND cron_expression IS NOT NULL AND cron_expression != ''` fetches all configs in one batch; `StartPerMinuteScheduler` (lines 35–69) registers exactly one Asynq cron entry (`* * * * *`) for the per-minute task — zero per-user cron entries |
| 4  | `StartPerMinuteScheduler` registers a `* * * * *` Asynq cron that fires every minute | VERIFIED | `internal/worker/scheduler.go` line 54: `entryID, err := scheduler.Register("* * * * *", task)` with `task` typed as `TaskPerMinuteScheduler`; `internal/worker/tasks.go` line 16: `TaskPerMinuteScheduler = "scheduler:per_minute"` |
| 5  | `StartPerMinuteScheduler` is called in both worker mode and embedded development mode | VERIFIED | `cmd/server/main.go` line 125: `stopScheduler, err := worker.StartPerMinuteScheduler(cfg)` (worker mode); line 159: `stopScheduler, err = worker.StartPerMinuteScheduler(cfg)` (embedded development mode) |
| 6  | Global cron scheduler and associated config fields are completely removed from the codebase | VERIFIED | Grep across entire codebase for `StartScheduler` (bare function name), `TaskScheduledBriefingGeneration`, `handleScheduledBriefingGeneration`, `BriefingSchedule`, `BriefingTimezone` returns zero source-code matches (only planning docs and UAT.md reference these identifiers historically) |
| 7  | Redis hash cache reduces DB load: last-run timestamps stored per user-plugin pair with cold-cache protection | VERIFIED | `internal/worker/scheduler.go` line 21: `const lastRunHashKey = "scheduler:last_run"`; line 220: `rdb.HGet(ctx, lastRunHashKey, fieldKey(userID, pluginID))` in `getLastRun`; line 234: `rdb.HSet(ctx, lastRunHashKey, fieldKey(userID, pluginID), t.Unix())` in `setLastRun`; lines 204–205 in `isDue`: `if lastRunAt.IsZero() { lastRunAt = time.Now().Add(-time.Minute) }` — cold-cache protection prevents mass-fire on startup |

**Score:** 7/7 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/worker/scheduler.go` | Per-minute scheduler registration and evaluation handler | VERIFIED | `StartPerMinuteScheduler` (line 27), `handlePerMinuteScheduler` (line 75), `isDue` (line 177), `getLastRun` / `setLastRun` (lines 219/233), `lastRunHashKey` constant (line 21) |
| `internal/worker/tasks.go` | `TaskPerMinuteScheduler` constant | VERIFIED | Line 16: `TaskPerMinuteScheduler = "scheduler:per_minute"` in `const` block alongside `TaskGenerateBriefing` and `TaskExecutePlugin` |
| `internal/worker/worker.go` | Handler registration for `TaskPerMinuteScheduler` in the Asynq mux | VERIFIED | Line 110: `mux.HandleFunc(TaskPerMinuteScheduler, handlePerMinuteScheduler(logger, db, rdb))` |
| `internal/plugins/models.go` | `UserPluginConfig.CronExpression` field (no `Timezone` field) | VERIFIED | Line 44: `CronExpression string` with nullable comment; zero `Timezone` field on `UserPluginConfig` struct (dropped by migration 000009) |
| `internal/database/migrations/000006_add_schedule_to_user_plugin_configs.up.sql` | `cron_expression` column addition | VERIFIED | `ALTER TABLE user_plugin_configs ADD COLUMN cron_expression VARCHAR(100)` plus partial index `idx_user_plugin_configs_scheduled` for fast scheduler queries |
| `internal/database/migrations/000009_remove_timezone_from_user_plugin_configs.up.sql` | `timezone` column removal | VERIFIED | `ALTER TABLE user_plugin_configs DROP COLUMN IF EXISTS timezone` — confirms per-plugin timezone is gone; timezone now on `users.timezone` |
| `cmd/server/main.go` | `StartPerMinuteScheduler` called in both modes | VERIFIED | Lines 125 (worker mode) and 159 (embedded development mode) both call `worker.StartPerMinuteScheduler(cfg)` |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `cmd/server/main.go` | `internal/worker/scheduler.go` | `worker.StartPerMinuteScheduler(cfg)` call | WIRED | Lines 125 and 159 invoke `StartPerMinuteScheduler`; stop function deferred for graceful shutdown |
| `internal/worker/scheduler.go` | `internal/worker/tasks.go` | `TaskPerMinuteScheduler` constant in `scheduler.Register` call | WIRED | `scheduler.Register("* * * * *", task)` at line 54; task uses `TaskPerMinuteScheduler` constant from `tasks.go` |
| `internal/worker/worker.go` | `internal/worker/scheduler.go` | `mux.HandleFunc(TaskPerMinuteScheduler, handlePerMinuteScheduler(...))` | WIRED | Line 110 wires the per-minute task type to `handlePerMinuteScheduler`; same package so no import needed |
| `internal/worker/scheduler.go` | `internal/plugins/models.go` | `plugins.UserPluginConfig` struct in DB query | WIRED | `var configs []plugins.UserPluginConfig` at line 77; `Preload("Plugin")` and `Preload("User")` fetch associations |
| `internal/worker/scheduler.go` | Redis | `rdb.HGet` / `rdb.HSet` in `getLastRun` / `setLastRun` | WIRED | `newSchedulerRedisClient` (line 247) creates dedicated Redis client; `getLastRun` line 220, `setLastRun` line 234 |

### Requirements Coverage

| Requirement | Description | Status | Notes |
|-------------|-------------|--------|-------|
| SCHED-01 | Per-user, per-plugin schedule configuration (cron expression stored in DB) | SATISFIED | `CronExpression` field on `UserPluginConfig`; migration 000006 adds column; `SaveSettingsHandler` persists value; timezone now account-level (users table) following Phase 14 migration 000009 |
| SCHED-02 | Database-backed schedule evaluation — NOT per-user Asynq cron entries | SATISFIED | Single DB query in `handlePerMinuteScheduler`; `isDue()` evaluates each config; confirmed zero per-user Asynq cron registrations in codebase |
| SCHED-03 | Per-minute Asynq scheduler task evaluates which user+plugin pairs are due | SATISFIED | `StartPerMinuteScheduler` registers `* * * * *` cron; `handlePerMinuteScheduler` is the handler; `TaskPerMinuteScheduler` constant in `tasks.go`; handler registered in `worker.go` mux |
| SCHED-05 | Global cron scheduler removed — replaced by per-user per-plugin DB-backed approach | SATISFIED | Zero codebase matches for `StartScheduler`, `TaskScheduledBriefingGeneration`, `handleScheduledBriefingGeneration`, `BriefingSchedule`, `BriefingTimezone` |
| SCHED-06 | Redis caching for last-run times (reduce DB load on per-minute evaluation) | SATISFIED | `lastRunHashKey` Redis hash; `HGet`/`HSet` in `getLastRun`/`setLastRun`; cold-cache protection in `isDue()` |

**Note:** SCHED-04 (timezone-aware cron matching) is verified in **Phase 14 VERIFICATION.md**, not Phase 10. Phase 14 migrated timezone from `UserPluginConfig` to `users` table (migration 000009). The per-timezone `isDue()` logic that reads `cfg.User.Timezone` is the Phase 14 implementation, so SCHED-04 correctly belongs to Phase 14's coverage.

### Anti-Patterns Found

| File | Pattern | Severity | Impact |
|------|---------|----------|--------|
| None detected | — | — | — |

No stale `BriefingSchedule` / `BriefingTimezone` config references found. No per-user Asynq scheduler registrations found. No `UserPluginConfig.Timezone` field references in source code (correctly removed by Phase 14).

### Human Verification Required

#### 1. Per-User Timezone Scheduler Dispatch

**Test:** Configure two users with different timezones (e.g. `America/New_York` and `Asia/Tokyo`) and two plugins with different cron schedules. Wait for the scheduler to tick past both schedule times.
**Expected:** Each user's plugin fires at the correct time in their respective timezone. The Redis `scheduler:last_run` hash key accumulates two fields (one per user-plugin pair) and `isDue()` correctly evaluates each based on the user's local time.
**Why human:** Requires live DB seeded with two distinct users, two distinct `users.timezone` values, a running Redis instance, and a running Asynq worker to observe actual dispatch behavior. The `isDue()` function's timezone logic (`CRON_TZ=<tz> <expr>`) cannot be verified from static analysis alone.

### Summary

Phase 10 delivered a production-ready per-minute scheduling system that replaced the previous global cron approach. The `StartPerMinuteScheduler` function registers a single Asynq cron entry (`* * * * *`) — one entry regardless of how many users or plugins exist. On each tick, `handlePerMinuteScheduler` runs one DB query to fetch all enabled configs with a non-null `cron_expression`, then evaluates each against its user's timezone using `isDue()`. Last-run timestamps are cached in a Redis hash (`scheduler:last_run`) with cold-cache protection to prevent mass-firing on startup.

`10-UAT.md` provides corroborating manual test evidence: 8/8 tests passed, including test 8 which directly confirmed that `BriefingSchedule` and `BriefingTimezone` were absent from `internal/config/config.go`. This VERIFICATION.md is the authoritative static code verification report; `10-UAT.md` is supplementary acceptance test evidence.

All 5 requirements (SCHED-01, SCHED-02, SCHED-03, SCHED-05, SCHED-06) are satisfied. SCHED-04 (timezone-aware matching) is covered in Phase 14 VERIFICATION.md.

---

_Verified: 2026-02-26_
_Verifier: Claude (gsd-executor)_
