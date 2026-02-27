---
phase: 09-crewai-sidecar-integration
plan: 04
subsystem: infra
tags: [asynq, redis-streams, crewai, worker, plugin-execution]

# Dependency graph
requires:
  - phase: 09-crewai-sidecar-integration
    provides: "streams.Publisher, PluginRequest type, Redis Stream infrastructure (09-01/09-02)"
  - phase: 09-crewai-sidecar-integration
    provides: "PluginRun model with status constants (09-01)"
provides:
  - "Asynq task handler for plugin execution (plugin:execute) creating PluginRun + publishing to Redis Stream"
  - "TaskExecutePlugin constant and EnqueueExecutePlugin function for enqueueing plugin tasks"
  - "streams.Publisher initialized in main.go and passed through to worker"
  - "End-to-end Go path: Asynq task -> PluginRun creation -> Redis Stream publish"
affects:
  - phase-10-scheduling
  - plugin-trigger-wiring

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Publisher passed as optional dependency (nil = graceful degradation with SkipRetry)"
    - "PluginRun created before stream publish to ensure audit trail even on publish failure"
    - "10-minute task timeout for long-running CrewAI workflows"
    - "UUID-based plugin_run_id for external tracking separate from GORM ID"

key-files:
  created: []
  modified:
    - internal/worker/tasks.go
    - internal/worker/worker.go
    - cmd/server/main.go

key-decisions:
  - "Graceful degradation pattern: nil publisher -> PluginRun marked failed + SkipRetry (no infinite retry on config error)"
  - "PluginRun record created before publish attempt so audit trail exists regardless of stream availability"
  - "Publish failures are retryable (stream down) vs nil publisher is non-retryable (misconfiguration)"
  - "Publisher initialized in main.go with non-fatal error (warning log) so app works even without streams"

patterns-established:
  - "Optional dependency injection pattern: *Publisher passed to worker, nil check inside handler"
  - "Asynq task options for CrewAI: MaxRetry(2), Timeout(10min), Unique(30min)"

# Metrics
duration: 8min
completed: 2026-02-18
---

# Phase 9 Plan 04: Publisher Wiring Summary

**Asynq plugin:execute handler creating PluginRun records and publishing to Redis Stream via streams.Publisher wired from main.go through to worker**

## Performance

- **Duration:** ~8 min
- **Started:** 2026-02-19T02:51:26Z
- **Completed:** 2026-02-19T03:00:00Z
- **Tasks:** 1
- **Files modified:** 3

## Accomplishments

- Added `TaskExecutePlugin = "plugin:execute"` constant and `EnqueueExecutePlugin` function to tasks.go with 10-minute timeout and 30-minute dedup window
- Implemented `handleExecutePlugin` handler in worker.go that creates a PluginRun record, generates a UUID for external tracking, and publishes to the Redis Stream via `publisher.PublishPluginRequest`
- Initialized `streams.Publisher` in main.go and threaded it through `worker.Run` and `worker.Start` calls — completes the Go-to-CrewAI pipeline
- Existing `handleGenerateBriefing` and n8n webhook pathway preserved unchanged

## Task Commits

1. **Task 1: Add plugin execution task type, handler, and Publisher wiring** - `9259dbb` (feat)

**Plan metadata:** (docs commit follows)

## Files Created/Modified

- `internal/worker/tasks.go` - Added `TaskExecutePlugin` constant and `EnqueueExecutePlugin` function
- `internal/worker/worker.go` - Updated `Run`/`Start`/`newServer` signatures, added `handleExecutePlugin` handler
- `cmd/server/main.go` - Added Publisher initialization block and updated worker call sites

## Decisions Made

- Graceful degradation: if `publisher` is nil, `handleExecutePlugin` marks the PluginRun as failed and returns `asynq.SkipRetry` — misconfiguration should not cause infinite retries
- PluginRun record created before stream publish attempt so an audit trail exists regardless of publish outcome
- Publish failure (stream temporarily unavailable) returns a retryable error; nil publisher (misconfiguration) returns SkipRetry
- Publisher initialized with a non-fatal warning log so the app continues to serve v1.0 briefing features even if stream publisher fails to initialize

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- `make test` exits non-zero due to a pre-existing `go tool covdata` toolchain issue when `-coverprofile` flag is used. Direct `go test -v -race ./...` passes cleanly (TestHandler: PASS). This is not caused by our changes.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

The end-to-end path from Asynq task to Redis Stream is now wired:

```
EnqueueExecutePlugin()
  -> Asynq queue (plugin:execute)
    -> handleExecutePlugin()
      -> PluginRun{status: pending} created
        -> publisher.PublishPluginRequest()
          -> Redis Stream plugin:requests
            -> (existing Python CrewAI consumer picks up)
```

Phase 10 (scheduling) can now call `EnqueueExecutePlugin` to trigger plugin execution. The trigger point is the only remaining gap — Phase 09 gap closure is complete.

## Self-Check: PASSED

- internal/worker/tasks.go: FOUND
- internal/worker/worker.go: FOUND
- cmd/server/main.go: FOUND
- 09-04-SUMMARY.md: FOUND
- Commit 9259dbb: FOUND

---
*Phase: 09-crewai-sidecar-integration*
*Completed: 2026-02-18*
