# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-14)

**Core value:** A user's configured briefing plugins run on schedule and their latest results appear automatically on a tile-based dashboard — no manual action needed to receive fresh, personalized briefings every day.
**Current focus:** v1.1 Plugin Architecture

## Current Position

Phase: 10 — Per-User Scheduling
Plan: 2/? complete
Status: In Progress
Last activity: 2026-02-19 — Plan 10-02 executed (scheduler wiring complete, config cleanup)

## Performance Metrics

**Overall Velocity:**
- Total plans completed: 18
- Average duration: 7.7 min
- Total execution time: 2.4 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 01 | 2 | 21 min | 10.5 min |
| 02 | 2 | 51 min | 25.5 min |
| 03 | 2 | 13 min | 6.5 min |
| 04 | 2 | 13 min | 6.5 min |
| 05 | 1 | 2 min | 2.4 min |
| 06 | 1 | 14 min | 14.0 min |
| 07 | 1 | 9 min | 9.0 min |
| 08 | 3 | 5 min | 1.7 min |
| 09 | 4 | 15 min | 3.8 min |
| 10 | 2 | 5 min | 2.5 min |

## Accumulated Context

### Decisions

All v1.0 decisions logged in PROJECT.md Key Decisions table.

**v1.1 decisions:**
- CrewAI over n8n for AI workflows (code-first, large community, bundleable per plugin)
- Redis Streams for Go → CrewAI communication (no new infra, async decoupling, independent scaling)
- Database-backed per-user scheduling (NOT per-user Asynq cron entries — avoids O(users × plugins) Redis)
- kaptinlin/jsonschema for dynamic settings validation (Google's official JSON Schema for Go)
- Schema versioning from day one (prevents metadata/state mismatch)
- Strict YAML validation with KnownFields(true) for plugin metadata (catches typos early)
- Non-fatal plugin discovery - invalid plugins logged and skipped, not blocking
- Default schema_version to "v1" when missing for backward compatibility
- JSONB storage for capabilities, configs, settings, and run data (flexibility + query capability)
- Composite unique index on user_id + plugin_id for UserPluginConfig (prevent duplicates)
- PluginRunID as separate UUID field for external tracking while maintaining GORM ID for joins
- Non-fatal plugin initialization allows graceful degradation (app serves v1.0 if plugins fail)
- PluginDir config field with PLUGIN_DIR env var support for environment-based override
- [Phase 09]: Redis Streams for Go-CrewAI communication with consumer groups and XACK pattern
- [Phase 09]: Non-fatal result consumer errors for graceful degradation (v1.0 still works if streams fail)
- [Phase 09-02]: FastAPI lifespan context manager for startup/shutdown (replaces deprecated on_event)
- [Phase 09-02]: Separate Redis client for health checks prevents blocking worker XREADGROUP
- [Phase 09-02]: Two-phase consumer loop recovers unACKed messages before reading new ones
- [Phase 09-02]: Dynamic crew loading via importlib with create_crew(settings) factory convention
- [Phase 09-02]: asyncio.timeout wrapper around CrewAI workflows prevents thread leaks
- [Phase 09-03]: @CrewBase decorator with YAML-based agent/task configuration
- [Phase 09-03]: Sequential task pipeline with context dependencies (research → write → review)
- [Phase 09-03]: Docker Compose mounts plugins read-only (no rebuild for crew changes)
- [Phase 09-03]: K8s HPA scales sidecar 1-5 replicas based on CPU (CrewAI is CPU-bound)
- [Phase 09-04]: Graceful degradation pattern — nil publisher marks PluginRun failed with SkipRetry (not infinite retry on misconfiguration)
- [Phase 09-04]: PluginRun record created before stream publish attempt (audit trail exists regardless of publish outcome)
- [Phase 09-04]: Publish failures retryable (stream down); nil publisher non-retryable (misconfiguration)
- [Phase 09-04]: Publisher initialized in main.go with non-fatal warning so app serves v1.0 even if streams fail to initialize
- [Phase 10-01]: Per-minute Asynq scheduler fires TaskPerMinuteScheduler; handler queries DB and dispatches due user-plugin pairs (not O(users*plugins) Redis entries)
- [Phase 10-01]: CRON_TZ=<timezone> prefix on cron expressions for robfig/cron/v3 timezone-aware parsing
- [Phase 10-01]: Cold-cache protection — zero lastRunAt treated as one minute ago prevents mass-fire on startup
- [Phase 10-01]: Redis hash HSET/HGET for last-run cache (scheduler:last_run key, userID:pluginID fields) avoids DB round-trips per tick
- [Phase 10-02]: asynq.Queue("critical") on TaskPerMinuteScheduler prevents scheduler starvation by long-running plugin:execute tasks
- [Phase 10-02]: BriefingSchedule/BriefingTimezone env vars retired — scheduling fully database-backed per-user (cron_expression + timezone columns)

### Pending Todos

[From .planning/todos/pending/ — ideas captured during sessions]

1. **Per-user briefing schedule configuration** (worker) — Addressed by Phase 10 (SCHED-01 through SCHED-06)
2. **Redesign system around plugin-based briefing architecture** (planning) — Addressed by v1.1 milestone (Phases 8-13)

### Blockers/Concerns

- CrewAI 2026 production patterns need validation (Phase 9 flagged for deeper research)
- kaptinlin/jsonschema extension field preservation (x-component) needs testing (Phase 12 flagged)

## Session Continuity

Last session: 2026-02-19 (Phase 10 Plan 02 execution — scheduler wiring and config cleanup)
Stopped at: Completed 10-02-PLAN.md (critical queue wiring, BriefingSchedule/BriefingTimezone removal)
Resume with: /gsd:execute-phase 10 (continue phase 10 plan 03)

---
*Created: 2026-02-10*
*Last updated: 2026-02-19 after Phase 10 Plan 02 execution*
