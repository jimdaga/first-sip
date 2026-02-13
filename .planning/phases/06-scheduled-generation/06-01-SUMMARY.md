---
phase: 06-scheduled-generation
plan: 01
subsystem: infra
tags: [asynq, scheduler, cron, background-jobs, automation]

# Dependency graph
requires:
  - phase: 03-background-job-infrastructure
    provides: Asynq worker infrastructure with Redis backend
  - phase: 04-briefing-generation-mock
    provides: Briefing generation task and handlers
provides:
  - Asynq Scheduler for automatic daily briefing generation
  - Configurable cron schedule via BRIEFING_SCHEDULE environment variable
  - Timezone-aware scheduling via BRIEFING_TIMEZONE environment variable
  - Scheduled task handler that generates briefings for all users
  - Duplicate prevention for overlapping manual and scheduled generation
affects: [07-production-deployment, future-user-preferences]

# Tech tracking
tech-stack:
  added: []
  patterns: [scheduler-lifecycle, embedded-scheduler-dev-mode, task-uniqueness]

key-files:
  created:
    - internal/worker/scheduler.go
  modified:
    - internal/config/config.go
    - internal/worker/tasks.go
    - internal/worker/worker.go
    - cmd/server/main.go

key-decisions:
  - "Default schedule 6 AM UTC (0 6 * * *) - configurable via BRIEFING_SCHEDULE environment variable"
  - "Timezone support via BRIEFING_TIMEZONE (default UTC) - enables deployment-time configuration"
  - "Embedded scheduler in development mode - runs alongside embedded worker in single process"
  - "Standalone scheduler in worker mode - scheduler lifecycle managed in main.go with deferred shutdown"
  - "Unique(1h) option on EnqueueGenerateBriefing - prevents duplicate briefings when manual and scheduled generation overlap"
  - "Separate task type for scheduled generation (briefing:scheduled_generation) - queries all users and enqueues individual tasks"
  - "Graceful ErrDuplicateTask handling - log and return nil (not an error condition)"

patterns-established:
  - "Scheduler lifecycle pattern: StartScheduler returns stop function for coordinated shutdown"
  - "Multi-mode scheduler deployment: embedded in dev, standalone in worker mode"
  - "Task uniqueness for idempotency: Unique option with TTL on scheduled tasks"

# Metrics
duration: 14 min
completed: 2026-02-13
---

# Phase 6 Plan 1: Scheduled Briefing Generation Summary

**Asynq Scheduler infrastructure with automatic daily briefing generation at configurable time (default 6 AM UTC) for all users**

## Performance

- **Duration:** 14 min
- **Started:** 2026-02-13T03:14:52Z
- **Completed:** 2026-02-13T03:29:44Z
- **Tasks:** 3 (2 implementation + 1 verification checkpoint)
- **Files modified:** 5

## Accomplishments

- Asynq Scheduler integrated with existing worker infrastructure using shared Redis backend
- Configurable daily briefing schedule via BRIEFING_SCHEDULE environment variable (default "0 6 * * *")
- Timezone-aware scheduling via BRIEFING_TIMEZONE environment variable (default "UTC")
- Scheduled task handler creates briefing records for all users and enqueues individual generation tasks
- Duplicate prevention using Unique(1h) option prevents overlapping manual and scheduled generation
- Coordinated scheduler lifecycle in both development (embedded) and worker (standalone) modes
- Graceful shutdown integration with HTTP server and worker shutdown sequence

## Task Commits

Each task was committed atomically:

1. **Task 1: Create scheduler infrastructure and config** - `c749ca1` (feat)
   - Created internal/worker/scheduler.go with StartScheduler function
   - Added BriefingSchedule and BriefingTimezone to config.Config
   - Added TaskScheduledBriefingGeneration constant
   - Implemented handleScheduledBriefingGeneration handler
   - Added Unique option to EnqueueGenerateBriefing with ErrDuplicateTask handling

2. **Task 2: Wire scheduler into main.go lifecycle** - `6a6c947` (feat)
   - Integrated scheduler startup in worker mode before worker.Run()
   - Added embedded scheduler in development mode after embedded worker
   - Coordinated shutdown sequence: HTTP server → scheduler → worker

3. **Task 3: Verify complete scheduler integration** - Human verification checkpoint (no code changes)

**Plan metadata:** (to be committed with STATE.md)

## Files Created/Modified

- `internal/worker/scheduler.go` - StartScheduler function with timezone support, task registration, lifecycle management
- `internal/config/config.go` - Added BriefingSchedule and BriefingTimezone fields with defaults
- `internal/worker/tasks.go` - Added TaskScheduledBriefingGeneration constant, Unique option, ErrDuplicateTask handling
- `internal/worker/worker.go` - Added handleScheduledBriefingGeneration handler, registered in mux
- `cmd/server/main.go` - Scheduler lifecycle integration for both development and worker modes

## Decisions Made

- **Default schedule 6 AM UTC**: Configurable via BRIEFING_SCHEDULE environment variable for deployment-time flexibility without code changes
- **Timezone support**: BRIEFING_TIMEZONE environment variable enables proper time interpretation across deployments (default UTC)
- **Embedded scheduler in dev mode**: Runs alongside embedded worker in single process for simplified development workflow
- **Separate scheduled task type**: briefing:scheduled_generation (no payload) vs briefing:generate (briefing_id payload) - cleaner separation of concerns
- **Unique(1h) on EnqueueGenerateBriefing**: Prevents duplicate briefings when manual and scheduled generation overlap, with graceful ErrDuplicateTask handling
- **Scheduler shutdown ordering**: Shutdown scheduler before worker to prevent new task enqueuing during worker shutdown

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None. Implementation followed research patterns directly. Verification revealed expected behavior: some scheduled tasks created briefing records that remained pending when worker wasn't running (normal for development testing with intermittent worker). These were manually cleaned up by marking as failed, which is correct operational behavior.

## User Setup Required

None - scheduler uses existing Redis infrastructure from Phase 3. Configuration is optional (defaults work for UTC deployments).

**Optional environment variables:**
- `BRIEFING_SCHEDULE` - Cron expression (default: "0 6 * * *" = 6 AM daily)
- `BRIEFING_TIMEZONE` - IANA timezone (default: "UTC")

Example for PST deployment:
```bash
export BRIEFING_SCHEDULE="0 6 * * *"
export BRIEFING_TIMEZONE="America/Los_Angeles"
```

## Verification Results

All 6 verification steps passed:

1. ✓ Scheduler starts in development mode with correct logging
2. ✓ Custom BRIEFING_SCHEDULE and BRIEFING_TIMEZONE configuration works
3. ✓ Scheduled generation creates briefing records for all users and enqueues tasks
4. ✓ Duplicate prevention works (Unique option prevents re-enqueuing)
5. ✓ Worker mode scheduler integration works
6. ✓ Graceful shutdown coordination works

**Testing notes:**
- Briefing IDs 7-15 were created by scheduled task runs during verification
- Some briefings remained pending when worker wasn't running (expected in dev testing)
- Stale pending briefings manually marked as failed (correct operational cleanup)
- Duplicate prevention validated through natural overlap during testing

## Next Phase Readiness

Phase 6 Plan 1 complete. Ready for Phase 6 Plan 2 (if additional scheduled generation features needed) or Phase 7 (Production Deployment).

**Scheduler infrastructure ready for:**
- Production deployment with single worker pod
- Additional scheduled tasks (future phases)
- Per-user schedule preferences (future enhancement)

**Known considerations for production:**
- Single scheduler instance required to prevent duplicate task enqueuing (satisfied by single worker pod deployment)
- Timezone configuration must be set for non-UTC deployments
- Monitor scheduler health via Asynqmon scheduler entries

---
*Phase: 06-scheduled-generation*
*Completed: 2026-02-13*

## Self-Check: PASSED

All key files verified to exist on disk:
- ✓ internal/worker/scheduler.go

All task commits verified in git history:
- ✓ c749ca1 (Task 1)
- ✓ 6a6c947 (Task 2)
