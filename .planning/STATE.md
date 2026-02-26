# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-14)

**Core value:** A user's configured briefing plugins run on schedule and their latest results appear automatically on a tile-based dashboard — no manual action needed to receive fresh, personalized briefings every day.
**Current focus:** v1.1 Plugin Architecture

## Current Position

Phase: 15 — Verification & Documentation Closure (IN PROGRESS)
Plan: 1/2 complete
Status: In Progress
Last activity: 2026-02-26 — Plan 15-01 executed (Phase 10 VERIFICATION.md for SCHED-01/02/03/05/06, Phase 11 VERIFICATION.md for TILE-01/02/03/04/05/06)

## Performance Metrics

**Overall Velocity:**
- Total plans completed: 25
- Average duration: 7.1 min
- Total execution time: ~2.7 hours

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
| 13 | 2/2 | ~62 min | 31.0 min |
| 14 | 2/2 | ~20 min | 10.0 min |
| 15 | 1/2 | ~3 min | 3.0 min |

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
- [Phase 13-01]: AccountTierID as *uint (nullable pointer) — NULL means pre-migration user, falls back to free tier in TierService automatically
- [Phase 13-01]: MaxEnabledPlugins = -1 sentinel for unlimited (pro tier) — avoids extra boolean column
- [Phase 13-01]: SeedAccountTiers called in ALL environments — tiers are production data, not dev fixtures
- [Phase 13-01]: TierService as standalone internal/tiers package — importable by any handler without circular imports
- [Phase 13-01]: CanUseFrequency uses two consecutive Next() calls to compute actual schedule interval — handles irregular cron expressions correctly
- [Phase 13-02]: SettingsPage signature changed from []PluginSettingsViewModel to SettingsPageViewModel — wraps plugins + TierInfo in single struct for clean template access
- [Phase 13-02]: IsFreeUser bool field on PluginSettingsViewModel carries tier context to accordion row for cron hint without threading full TierInfo through every level
- [Phase 13-02]: proNotifyHandler as inline closure in main.go — no DB table needed for scaffolding, slog.Info sufficient for MVP notification tracking
- [Phase 13-02]: Counter accent color at limit uses --accent token (warm orange) not red — limits are value-focused upgrade prompts, not error states
- [Phase 14-01]: Sidecar _wrap_output uses re.split on ^##\s+ headings to build sections array — simple, lossless, no extra dependency
- [Phase 14-01]: Empty/None raw_output uses placeholder JSON fallback instead of failing — genuine failures (exceptions, timeouts) use status=failed
- [Phase 14-01]: html/template.HTMLEscapeString used in viewmodel.go for section title escaping (not templ import — avoids new dependency in non-template code)
- [Phase 14-01]: Retry button reuses existing POST /api/settings/:pluginID/run-now — no new handler needed
- [Phase 14-01]: Old malformed runs return empty content (no crash) via JSON parse failure returning empty string
- [Phase 14-02]: Timezone removed from UserPluginConfig — scheduler reads cfg.User.Timezone (account-level) with UTC fallback, single source of truth
- [Phase 14-02]: Account settings page at /settings/account with renderTimezoneSelect reuse — same component as plugin settings
- [Phase 14-02]: Dashboard and settings SQL JOIN users table for timezone — users.timezone replaces upc.timezone
- [Phase 15-01]: SCHED-04 excluded from Phase 10 VERIFICATION.md — belongs to Phase 14 (timezone migrated from UserPluginConfig to users table by migration 000009)
- [Phase 15-01]: 7 truths for Phase 10 and 8 truths for Phase 11 (vs 5/6 minimums) — extra truths needed to cover migration context and TileViewModel package architecture

### Pending Todos

[From .planning/todos/pending/ — ideas captured during sessions]

1. **Per-user briefing schedule configuration** (worker) — Addressed by Phase 10 (SCHED-01 through SCHED-06)
2. **Redesign system around plugin-based briefing architecture** (planning) — Addressed by v1.1 milestone (Phases 8-13)

### Blockers/Concerns

- CrewAI 2026 production patterns need validation (Phase 9 flagged for deeper research)
- kaptinlin/jsonschema extension field preservation (x-component) — RESOLVED in Phase 12-01: SetPreserveExtra(true) confirmed working

## Session Continuity

Last session: 2026-02-26 (Phase 15-01 complete — Phase 10 VERIFICATION.md for SCHED-01/02/03/05/06, Phase 11 VERIFICATION.md for TILE-01/02/03/04/05/06)
Stopped at: Phase 15 Plan 1/2 complete. Verification closure for scheduling and tile requirements done.
Resume with: Phase 15 Plan 02 — update REQUIREMENTS.md (18 checkbox checks + traceability table + coverage count)

---
*Created: 2026-02-10*
*Last updated: 2026-02-26 after Phase 15-01 (Phase 10 VERIFICATION.md for SCHED-01/02/03/05/06, Phase 11 VERIFICATION.md for TILE-01/02/03/04/05/06)*
