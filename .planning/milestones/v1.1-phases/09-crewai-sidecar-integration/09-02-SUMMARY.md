---
phase: 09-crewai-sidecar-integration
plan: 02
subsystem: infra
tags: [python, fastapi, crewai, redis-streams, docker, asyncio]

# Dependency graph
requires:
  - phase: 09-crewai-sidecar-integration
    provides: Redis Streams infrastructure from plan 01 (plugin:requests, plugin:results streams, Go producer/consumer)
provides:
  - FastAPI sidecar service with health probes (/health/live, /health/ready)
  - Redis Streams consumer with two-phase pattern (pending recovery + new messages)
  - CrewAI executor with asyncio.timeout wrapper for workflow execution
  - Pydantic models for PluginRequest/PluginResult matching Go structs
  - Production Dockerfile with multi-stage build
affects: [09-03, plugin-development]

# Tech tracking
tech-stack:
  added: [crewai, fastapi, uvicorn, redis-py, pydantic]
  patterns: [two-phase stream consumer, asyncio timeout wrapper, dynamic module loading, separate health check client]

key-files:
  created:
    - sidecar/pyproject.toml
    - sidecar/models.py
    - sidecar/main.py
    - sidecar/worker.py
    - sidecar/executor.py
    - sidecar/Dockerfile
  modified: []

key-decisions:
  - "Use lifespan context manager instead of deprecated on_event for FastAPI startup/shutdown"
  - "Separate Redis client for health checks prevents blocking worker's XREADGROUP"
  - "ACK bad messages immediately (don't retry), don't ACK processing errors (retry via PEL)"
  - "Dynamic crew loading via importlib with create_crew(settings) factory convention"
  - "asyncio.timeout wrapper instead of CrewAI built-in timeout for thread-leak safety"
  - "Two workers in production (not many - CrewAI workflows are CPU-bound per execution)"

patterns-established:
  - "Two-phase consumer loop: Phase 1 recovers unACKed messages (XREADGROUP with '0'), Phase 2 reads new messages (XREADGROUP with '>')"
  - "Each plugin's crew/crew.py must export create_crew(settings: dict) -> Crew factory function"
  - "Health endpoints use separate Redis client to avoid exhausting worker connection pool"
  - "Worker publishes result to plugin:results stream BEFORE ACKing request (ensures durability)"

# Metrics
duration: 2min
completed: 2026-02-15
---

# Phase 09 Plan 02: CrewAI Sidecar Service Summary

**FastAPI sidecar with Redis Streams consumer, CrewAI executor with timeout protection, and production Dockerfile for isolated Python workflow runtime**

## Performance

- **Duration:** 2 min (146s)
- **Started:** 2026-02-15T02:52:48Z
- **Completed:** 2026-02-15T02:55:14Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments
- Complete FastAPI service with Kubernetes-compatible health probes (liveness and readiness)
- Two-phase Redis Streams consumer loop with automatic pending recovery and ACK-on-success pattern
- CrewAI executor with asyncio.timeout wrapper preventing thread leaks from long-running workflows
- Pydantic v2 models with exact schema alignment to Go PluginRequest/PluginResult structs
- Production-ready multi-stage Dockerfile using uv for fast dependency installation

## Task Commits

Each task was committed atomically:

1. **Task 1: Pydantic models, FastAPI app with health endpoints, and project config** - `bca04db` (feat)
2. **Task 2: Redis Streams worker loop, CrewAI executor with timeout, and Dockerfile** - `9fcdcd1` (feat)

## Files Created/Modified
- `sidecar/pyproject.toml` - Python project config with crewai, fastapi, uvicorn, redis dependencies
- `sidecar/models.py` - Pydantic PluginRequest/PluginResult models with stream constants (plugin:requests, plugin:results, crewai-workers)
- `sidecar/main.py` - FastAPI app with /health/live (always 200), /health/ready (Redis ping), lifespan startup/shutdown, separate health Redis client
- `sidecar/worker.py` - Two-phase consumer loop (pending recovery from PEL + new messages), ACK after result publication
- `sidecar/executor.py` - CrewExecutor with asyncio.timeout wrapper around crew.kickoff_async, dynamic crew loading via importlib
- `sidecar/Dockerfile` - Multi-stage build with uv installer, 2 uvicorn workers for production

## Decisions Made

**FastAPI lifespan pattern:**
- Used lifespan context manager instead of deprecated `@app.on_event("startup")` for FastAPI 0.115+ compatibility
- Provides proper async cleanup on shutdown (cancel worker task, close Redis connections)

**Separate health check client:**
- Created dedicated `app.state.health_redis` client for readiness probe
- Prevents health checks from blocking worker's XREADGROUP when connection pool is exhausted
- Critical for Kubernetes liveness/readiness probes under load

**ACK strategy for message reliability:**
- ACK immediately for bad messages (invalid JSON, validation errors) - don't retry poison messages
- ACK after successful result publication - ensures result durability before removing from PEL
- Don't ACK on processing errors - message stays in pending entry list for automatic retry

**Dynamic crew loading convention:**
- Each plugin's `crew/crew.py` must export `create_crew(settings: dict) -> Crew` factory function
- Enables runtime discovery of plugin crews without hardcoded imports
- Settings passed through from PluginRequest allows per-execution configuration

**Timeout implementation:**
- Use `asyncio.timeout()` wrapper instead of CrewAI's built-in timeout
- Prevents thread leaks from long-running workflows that don't respect internal cancellation
- Default 300s timeout configurable via CREW_TIMEOUT_SECONDS env var

**Worker concurrency:**
- 2 uvicorn workers in production (not many)
- CrewAI workflows are CPU-bound per execution - more workers just increase contention
- Each worker runs one consumer loop, scaling horizontally via multiple container replicas

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None - all tasks completed as specified with all verification checks passing.

## User Setup Required

None - no external service configuration required. Service reads configuration from environment variables (REDIS_URL, PLUGIN_DIR, CREW_TIMEOUT_SECONDS).

## Next Phase Readiness

**Ready for Phase 09 Plan 03 (Sample Plugin with CrewAI Crew):**
- Sidecar service complete and ready to execute workflows
- Stream constants and message schemas aligned between Go and Python
- Dynamic crew loading mechanism in place (expects crew/crew.py with create_crew factory)
- Health endpoints ready for Kubernetes deployment

**Validation needed:**
- End-to-end test with actual CrewAI crew (Plan 03 will provide first sample plugin)
- Verify timeout protection works with real long-running workflows
- Test pending recovery with unACKed messages in Redis

**Known gaps:**
- No actual plugin crews exist yet (Plan 03 will create first sample)
- Docker image not built/tested (will be validated in integration testing phase)

## Self-Check: PASSED

All claimed files and commits verified:
- Created files: pyproject.toml, models.py, main.py, worker.py, executor.py, Dockerfile (6/6 found)
- Task commits: bca04db, 9fcdcd1 (2/2 found)

---
*Phase: 09-crewai-sidecar-integration*
*Completed: 2026-02-15*
