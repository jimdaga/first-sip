# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-14)

**Core value:** A user's configured briefing plugins run on schedule and their latest results appear automatically on a tile-based dashboard — no manual action needed to receive fresh, personalized briefings every day.
**Current focus:** v1.1 Plugin Architecture

## Current Position

Phase: 12 — Dynamic Settings UI (COMPLETE)
Plan: 2/2 complete
Status: Complete
Last activity: 2026-02-23 — Plan 12-02 executed (settings UI templates, UAT fixes, shared AppNavbar)

## Performance Metrics

**Overall Velocity:**
- Total plans completed: 24
- Average duration: 7.2 min
- Total execution time: 2.7 hours

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
| 11 | 3 | 34 min | 11.3 min |
| 12 | 2/2 | 42 min | 21.0 min |

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
- [Phase 11-01]: DisplayOrder as nullable *int avoids zero-value ambiguity between "unordered" and "first position"
- [Phase 11-01]: TileSize defaults to "1x1" in syncPluginToDB (not in YAML or struct) — plugins without tile_size still get valid DB value
- [Phase 11-01]: Migration 000007 uses IF NOT EXISTS guards for idempotent re-runs during development
- [Phase 11-02]: DashboardHandler calls existing DashboardPage template signature to keep build green until Plan 03 updates template
- [Phase 11-02]: TileStatusHandler is working stub (200 OK) — Plan 03 provides Templ tile component to render
- [Phase 11-02]: UpdateTileOrderHandler skips malformed plugin_id values rather than failing the entire reorder request
- [Phase 11-03]: JS fixed-position tooltip over CSS ::after — glass-card overflow:hidden clips pseudo-elements
- [Phase 11-03]: Info badge in tile footer (bottom-left) not header — avoids close button overlap
- [Phase 11-03]: Browser timezone auto-detection via Intl API on dashboard load (fires once per session if user timezone is UTC)
- [Phase 12-01]: settingsvm package for ViewModel types breaks handler→templates→handler import cycle (same pattern as internal/tiles)
- [Phase 12-01]: schemaToFields converts *jsonschema.Schema to []FieldViewModel before passing to Templ — templates never import kaptinlin/jsonschema
- [Phase 12-01]: compiler.SetPreserveExtra(true) enabled for future x-extension field support (resolves Phase 12 research flag)
- [Phase 12-01]: Run Now uses saved DB settings not unsaved form state — prevents silent execution with unsaved config
- [Phase 12-01]: coerceFormValues handles absent boolean as false (Pitfall 3), array multi-select via rawForm[key] (Pitfall 4)
- [Phase 12-02]: AppNavbar shared component with activePage param — eliminates duplicated navbar HTML across dashboard and settings pages
- [Phase 12-02]: FieldTypeTagInput replaces FieldTypeCheckboxGroup for all array fields — better UX per user review; schema validation still enforces constraints at save time
- [Phase 12-02]: FieldTypeTimeSelect detected via HH:MM pattern substring in schema.Pattern — avoids hardcoding field names
- [Phase 12-02]: Consolidated three htmx.on('htmx:afterSwap') listeners into one — HTMX v2 replaces previous listeners for same event, only last one ran
- [Phase 12-02]: DisplayName via humanizePluginName (kebab-case → Title Case) — no DB change needed, purely presentational
- [Phase 12-02]: JS tooltip for .settings-tooltip-trigger matches tile-tooltip pattern — CSS ::after pseudo-elements clipped by overflow:hidden containers
- [Phase 12-02]: JSON array coercion in coerceFormValues — tag input hidden field sends JSON array string, parsed before passing to validator

### Pending Todos

[From .planning/todos/pending/ — ideas captured during sessions]

1. **Per-user briefing schedule configuration** (worker) — Addressed by Phase 10 (SCHED-01 through SCHED-06)
2. **Redesign system around plugin-based briefing architecture** (planning) — Addressed by v1.1 milestone (Phases 8-13)

### Blockers/Concerns

- CrewAI 2026 production patterns need validation (Phase 9 flagged for deeper research)
- kaptinlin/jsonschema extension field preservation (x-component) — RESOLVED in Phase 12-01: SetPreserveExtra(true) confirmed working

## Session Continuity

Last session: 2026-02-23 (Phase 12-02 complete — settings UI templates, 8 UAT fixes, shared AppNavbar)
Stopped at: Phase 12 complete (both plans done). Settings page fully functional.
Resume with: /gsd:execute-phase 13 (next phase in roadmap)

---
*Created: 2026-02-10*
*Last updated: 2026-02-23 after Phase 12-02 (settings UI templates, UAT fixes, AppNavbar shared component)*
