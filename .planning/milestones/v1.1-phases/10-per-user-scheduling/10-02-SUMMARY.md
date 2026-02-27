---
phase: 10-per-user-scheduling
plan: 02
subsystem: worker
tags: [asynq, scheduler, config, queue-priorities, cleanup]

# Dependency graph
requires:
  - phase: 10-01
    provides: StartPerMinuteScheduler, handlePerMinuteScheduler, TaskPerMinuteScheduler, newSchedulerRedisClient

provides:
  - asynq.Queue("critical") on TaskPerMinuteScheduler task so it lands on the critical queue
  - Asynq server Queues map (critical:6, default:3) ensuring scheduler never delayed by plugin tasks
  - Removal of BriefingSchedule and BriefingTimezone from Config struct and Load()

affects: [10-per-user-scheduling]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Asynq queue priorities: Queues map with critical:6, default:3; scheduler task uses asynq.Queue(\"critical\")"
    - "Per-user schedules stored in DB (cron_expression, timezone) — no env-var config needed"

key-files:
  created: []
  modified:
    - internal/worker/scheduler.go
    - internal/config/config.go

key-decisions:
  - "asynq.Queue(\"critical\") on TaskPerMinuteScheduler ensures scheduler tick cannot be starved by long-running plugin:execute tasks on the default queue"
  - "BriefingSchedule and BriefingTimezone env vars retired — per-user schedules are fully database-backed (migration 000006)"

# Metrics
duration: 2min
completed: 2026-02-19
---

# Phase 10 Plan 02: Scheduler Wiring and Config Cleanup Summary

**Added asynq.Queue("critical") to per-minute scheduler task and removed deprecated BriefingSchedule/BriefingTimezone config fields — global cron scheduler is fully retired in favor of database-backed per-user scheduling**

## Performance

- **Duration:** ~2 min
- **Started:** 2026-02-19T15:11:36Z
- **Completed:** 2026-02-19T15:12:51Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments

- Added `asynq.Queue("critical")` to the `TaskPerMinuteScheduler` task in `StartPerMinuteScheduler` so the per-minute tick lands on the critical queue and is processed before default-queue `plugin:execute` tasks
- Removed `BriefingSchedule` and `BriefingTimezone` fields from `Config` struct and their corresponding `getEnvWithDefault` assignments in `Load()` — global cron config is fully retired
- Verified all success criteria: zero grep matches for BriefingSchedule, BriefingTimezone, StartScheduler (bare), TaskScheduledBriefingGeneration, handleScheduledBriefingGeneration; critical queue confirmed in both server config and task options

## Task Commits

Each task was committed atomically:

1. **Task 1: Wire per-minute scheduler and add critical queue to Asynq server** - `7515eba` (feat)
   - Added `asynq.Queue("critical")` option to `TaskPerMinuteScheduler` task in `scheduler.go`
   - Note: main.go wiring, Asynq server queue config, mux handler registration, and old handler removal were all completed by Plan 10-01 as a Rule 3 auto-fix

2. **Task 2: Remove deprecated global scheduler config fields** - `e202147` (chore)
   - Removed `BriefingSchedule string` and `BriefingTimezone string` from `Config` struct
   - Removed `BriefingSchedule` and `BriefingTimezone` assignments from `Load()`

**Plan metadata:** _(docs commit follows)_

## Files Created/Modified

- `internal/worker/scheduler.go` — Added `asynq.Queue("critical")` to `TaskPerMinuteScheduler` task options in `StartPerMinuteScheduler`
- `internal/config/config.go` — Removed `BriefingSchedule` and `BriefingTimezone` fields from struct and `Load()` function

## Decisions Made

- `asynq.Queue("critical")` on the per-minute scheduler task ensures it is never delayed by long-running `plugin:execute` tasks — the critical queue (priority 6) is always drained before the default queue (priority 3)
- Retiring `BRIEFING_SCHEDULE` / `BRIEFING_TIMEZONE` env vars removes dead configuration and signals clearly that scheduling is now fully database-backed per-user

## Deviations from Plan

### Tasks Partially Pre-Completed by Plan 10-01

**Task 1: Wire per-minute scheduler and add critical queue to Asynq server**

The majority of Task 1 was completed by Plan 10-01 as a Rule 3 auto-fix (blocking compile error):

| Sub-task | Status | Where done |
|---|---|---|
| Add Queues map to Asynq server config | Done in 10-01 (31ea758) | worker.go lines 91-94 |
| Register TaskPerMinuteScheduler handler in mux | Done in 10-01 (31ea758) | worker.go line 110 |
| Call StartPerMinuteScheduler in worker mode (main.go) | Done in 10-01 (31ea758) | main.go line 111 |
| Call StartPerMinuteScheduler in embedded dev mode (main.go) | Done in 10-01 (31ea758) | main.go line 145 |
| Remove handleScheduledBriefingGeneration | Done in 10-01 (31ea758) | worker.go (removed) |
| Add asynq.Queue("critical") to task options | Done in 10-02 (7515eba) | scheduler.go line 48 |

The only remaining sub-task was adding `asynq.Queue("critical")` to the task options in `StartPerMinuteScheduler`, which was completed in this plan.

**Sub-tasks NOT done by 10-01 (plan notes said "schedulerCache Redis client created and passed through call chain"):**
The plan originally called for passing `schedulerCache *redis.Client` as a parameter through `Run`/`Start`/`newServer`. Instead, 10-01 created `newSchedulerRedisClient` and called it directly inside `newServer` — equivalent result, different approach. No parameter threading was needed.

---

**Total deviations:** 1 pre-completion by 10-01 (Rule 3 — blocking compile error)
**Impact:** Task 1 scope reduced to a single 1-line addition (`asynq.Queue("critical")`). Task 2 executed as planned.

## Issues Encountered

None. Both builds passed cleanly; `go test -race ./...` passed.

## User Setup Required

None. The removal of `BriefingSchedule` and `BriefingTimezone` may produce warnings if `.env` files still set `BRIEFING_SCHEDULE` or `BRIEFING_TIMEZONE` — those env vars will simply be ignored going forward.

## Next Phase Readiness

- Plan 10-03 and beyond: Per-user scheduling end-to-end is fully wired
- The full path: `StartPerMinuteScheduler` fires every minute → `handlePerMinuteScheduler` queries DB → `isDue` evaluates user cron expressions → `EnqueueExecutePlugin` dispatches due pairs on the default queue → scheduler task itself runs on critical queue to prevent starvation
- No blockers

## Self-Check: PASSED

All artifacts verified:

- FOUND: `internal/worker/scheduler.go` with `asynq.Queue("critical")` at line 48
- FOUND: `internal/config/config.go` without BriefingSchedule, BriefingTimezone
- FOUND: commits 7515eba (feat scheduler Queue critical), e202147 (chore config cleanup)
- VERIFIED: Zero grep matches for BriefingSchedule, BriefingTimezone, StartScheduler (bare), TaskScheduledBriefingGeneration, handleScheduledBriefingGeneration
- VERIFIED: critical queue in both Asynq server config (worker.go:92) and task options (scheduler.go:48)
- VERIFIED: StartPerMinuteScheduler called in both branches of main.go (lines 111 and 145)
- VERIFIED: `go build ./...` and `go test -race ./...` both pass

---
*Phase: 10-per-user-scheduling*
*Completed: 2026-02-19*
