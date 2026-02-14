# Project Research Summary

**Project:** First Sip v1.1 Plugin Architecture
**Domain:** Plugin-Based Personal Briefing System (Go Web App Extension)
**Researched:** 2026-02-13
**Confidence:** MEDIUM-HIGH

## Executive Summary

The v1.1 plugin architecture milestone transforms First Sip from a monolithic briefing generator into a modular, user-configurable briefing platform. Research reveals that successful plugin systems prioritize three principles: strong isolation boundaries (each plugin owns its data and schedule), metadata-driven UI generation (JSON Schema reduces code duplication), and graceful degradation (one plugin's failure doesn't break the system). The recommended approach combines Go's plugin framework with a Python CrewAI sidecar for AI workflows, leveraging existing Asynq infrastructure for per-user scheduling.

The critical architectural shift is from global cron-based briefing generation to per-minute schedule evaluation with database-backed user-plugin configurations. This enables per-user timezones, per-plugin schedules, and account tier enforcement. The CrewAI sidecar runs as a Kubernetes sidecar container (not a separate service) to avoid network latency and simplify deployment. Plugin metadata lives in YAML files, loaded at startup and cached in memory, with a clear migration path for schema evolution.

Key risks center on complexity management and scalability. The scheduler pattern must avoid O(users × plugins) Redis entries by using database queries with Redis caching for last-run times. The plugin metadata schema versioning system must be implemented immediately to prevent breaking changes when settings evolve. CrewAI process management requires proper signal handling to prevent orphaned Python processes on Go service restarts. These risks are mitigable with careful architecture from day one, rather than refactoring later at higher cost.

## Key Findings

### Recommended Stack

**New dependencies for plugin system (minimal additions):**

The existing Go/Gin/GORM/Asynq/Templ stack already provides most infrastructure needed. Only two new libraries required: `kaptinlin/jsonschema` (Google's official JSON Schema validator for Go, announced January 2026) for dynamic settings validation, and `goccy/go-yaml` for plugin metadata parsing (already in go.mod as indirect dependency). The CrewAI Python sidecar adds FastAPI + CrewAI dependencies but runs as a separate container.

**Core technologies:**
- **goccy/go-yaml v1.18.0+**: Plugin metadata parsing — already in go.mod, superior to unmaintained yaml.v3, passes 355+ YAML test suite cases
- **kaptinlin/jsonschema v0.6.11+**: Dynamic settings validation — Google's official JSON Schema package, preserves extension fields (x-component) for UI hints, full Draft 2020-12 compliance
- **Asynq v0.26.0 (existing)**: Per-user scheduling — extend with database-backed schedule evaluation, no additional scheduler needed (avoid go-co-op/gocron for now)
- **CrewAI (Python sidecar)**: AI workflow orchestration — multi-agent briefing generation, runs in sidecar container, HTTP communication with Go
- **CSS Grid (native)**: Tile-based dashboard — no JavaScript masonry libraries needed, auto-fit with minmax() for responsive layouts

**What NOT to add:**
- Additional task queue (Celery) — Asynq is single source of truth
- JavaScript form libraries — Templ generates server-side from JSON Schema
- Separate cron daemon — Asynq handles all scheduling
- Plugin marketplace infrastructure — pre-installed plugins only for v1.1

### Expected Features

**Must have (table stakes):**
- **Plugin enable/disable** — users expect granular control over what runs
- **Plugin directory** — discoverable list of available plugins with descriptions
- **Per-plugin settings** — each plugin has different configuration needs (API keys, location, preferences)
- **Per-user schedules** — timezone-aware, per-plugin cron expressions (not global 6 AM)
- **Tile-based dashboard** — scannable grid showing all enabled plugins' latest briefings
- **Dynamic settings forms** — auto-generated from JSON Schema (no manual form coding per plugin)
- **Plugin execution status** — clear visibility into last run, next run, errors

**Should have (competitive advantages):**
- **Code-based plugins** — version controlled workflows (differentiates from GUI-only tools like n8n)
- **CrewAI multi-agent workflows** — best-in-class AI orchestration (differentiates from single-LLM tools)
- **JSON Schema-driven UI** — schema serves as validation + UI generation + documentation (reduces maintenance)
- **Transparent scheduling** — users see exact cron expression, can customize per plugin
- **Account tier scaffolding** — limit plugin count by tier (free vs pro), enforce server-side

**Defer (v2+):**
- **Plugin marketplace** — external plugins require security review, code signing, sandbox
- **Multi-agent memory** — CrewAI memory backend for context across briefings (high complexity)
- **Plugin dependencies** — one plugin using another's output (coupling nightmare)
- **Real-time plugin execution** — on-demand triggers defeat batching benefits
- **Visual workflow builder** — CrewAI is code-first, GUI abstraction loses power

### Architecture Approach

The plugin architecture extends existing packages with minimal new surface area. Core additions: `/internal/plugins` for plugin lifecycle management, `/internal/scheduling` for per-user schedule evaluation, `/internal/settings` for plugin configuration UI, and `/plugins` directory (outside Go code) for plugin definitions. The CrewAI sidecar runs as a FastAPI service in the same Kubernetes pod as the worker, communicating via HTTP over localhost to avoid network latency.

**Major components:**
1. **Plugin Manager** (`/internal/plugins`) — Discovers plugins from `/plugins` directory at startup, parses YAML metadata, validates JSON schemas, maintains in-memory registry, provides HTTP client for CrewAI sidecar
2. **Scheduler** (`/internal/scheduling`) — Runs per-minute evaluation (not per-user cron), queries database for due user-plugin pairs, respects timezones, enqueues Asynq tasks, uses Redis cache for last-run times to reduce DB load
3. **Plugin Executor** (worker task handler) — Dequeues plugin execution tasks, calls CrewAI sidecar HTTP endpoint, creates Briefing records, handles failures with PluginRun tracking, updates next-run timestamps
4. **CrewAI Sidecar** (Python/FastAPI) — Loads plugin agents/tasks dynamically, executes workflows synchronously, returns JSON results, no separate task queue (Asynq is authoritative)
5. **Tile Dashboard** (Templ templates) — CSS Grid layout, pre-fetches latest briefing per plugin (avoids N+1), handles nil briefings gracefully, HTMX updates in-place
6. **Settings Service** (`/internal/settings`) — Generates forms from JSON Schema, validates with kaptinlin/jsonschema, coerces form strings to schema types, handles schema versioning

**Critical architectural decisions:**
- **Database-backed scheduling** (NOT per-user Asynq cron) — avoids O(users × plugins) Redis memory issue
- **Sidecar pattern** (NOT separate CrewAI service) — localhost communication, scales with workers, simplified deployment
- **Single task queue** (Asynq only, not Asynq + Celery) — reduces operational complexity, single monitoring dashboard
- **Schema versioning from day one** — prevents breaking changes when plugin settings evolve

### Critical Pitfalls

1. **Plugin metadata YAML vs runtime state mismatch** — Developer adds field to YAML schema, user's saved settings lack new field, template crashes. **Prevention:** Implement schema versioning immediately (version field in YAML + DB), write migration functions, templates handle missing fields with defaults, never remove schema fields (only deprecate).

2. **CrewAI Python process orphaned on Go restart** — Go crashes, Python sidecar continues running, new Go spawns another Python, duplicate tasks + memory leak. **Prevention:** Use shared process namespace in K8s, register signal handlers to kill subprocess, track PID and clean stale processes on startup, set Setpgid for process group cleanup.

3. **Per-user Asynq scheduler creates O(users × plugins) Redis entries** — 100 users × 10 plugins = 1000 scheduler entries, Redis OOM at scale. **Prevention:** Single per-minute cron task queries database for due schedules (not per-user Asynq cron), store schedule config in Postgres `user_plugin_configs` table with `next_run_at` index.

4. **Tile dashboard N+1 query for latest briefing per plugin** — Dashboard loads, queries each plugin's latest briefing separately = 8 plugins = 9 queries. **Prevention:** Single query with window function (`SELECT DISTINCT ON (plugin_id)`), pre-fetch all briefings in handler, pass map to template.

5. **HTMX form loses client state on validation error** — User fills 8 fields, submits, validation fails, HTMX replaces form, all input lost. **Prevention:** Bind ALL form fields on server (including invalid ones), re-render form with user's input pre-filled, display field-level errors inline.

## Implications for Roadmap

Based on research, suggested phase structure prioritizes foundation-first with early validation via working plugin:

### Phase 1: Plugin Framework Foundation
**Rationale:** Database schema and plugin metadata loading must come first — everything depends on these models. Validating the end-to-end flow early with a single working plugin reduces risk before building horizontal features.

**Delivers:**
- Database migrations (Plugin, UserPluginConfig, PluginRun, AccountTier models)
- Plugin metadata YAML parsing with schema versioning
- Plugin discovery and registration at startup
- Single example plugin (weather or news briefing) working end-to-end

**Addresses Features:**
- Plugin directory (basic list from database)
- Plugin metadata storage (foundation for settings)

**Avoids Pitfalls:**
- Schema versioning from day one (prevents mismatch issues)
- Plugin metadata validation prevents bad YAML

**Research Needed:** LOW — GORM migrations and YAML parsing well-documented

---

### Phase 2: CrewAI Sidecar Integration
**Rationale:** Establishing the Go-Python communication pattern early validates the most uncertain architectural component. Proves CrewAI can execute workflows and return results before building scheduling complexity.

**Delivers:**
- FastAPI sidecar with CrewAI executor
- Go HTTP client for sidecar communication
- Plugin executor service (Go → CrewAI → Briefing)
- Process management (signal handlers, PID tracking)
- Example plugin with real CrewAI agents/tasks

**Uses Stack:**
- CrewAI for multi-agent workflows
- HTTP (not gRPC) for simplicity

**Implements Architecture:**
- Sidecar pattern (same pod, localhost communication)
- Single task queue (Asynq enqueues, calls CrewAI HTTP endpoint synchronously)

**Avoids Pitfalls:**
- Python process lifecycle management with signal handlers
- No duplicate task queues (Asynq only)

**Research Needed:** MEDIUM — CrewAI production patterns less established, needs phase-specific research on agent configuration and error handling

---

### Phase 3: Per-User Scheduling
**Rationale:** With plugin execution proven (Phase 2), add scheduling logic. Database-backed approach avoids O(users) scalability trap. Must come before UI because UI displays schedule information.

**Delivers:**
- Per-minute scheduler task (evaluates due schedules)
- IsDue logic with timezone support
- Asynq task handler for plugin execution
- Migration: existing users to new UserPluginConfig records
- Redis caching for last-run times (performance optimization)

**Uses Stack:**
- Asynq scheduler (modified from global cron to per-minute evaluation)
- PostgreSQL for schedule storage (`next_run_at` indexed)

**Implements Architecture:**
- Scheduling package (schedule evaluation, cron matching)

**Avoids Pitfalls:**
- Database-backed schedules prevent O(users) Redis memory issue
- Timezone handling prevents "works for developer in UTC, breaks for users in PST"

**Research Needed:** LOW — Asynq patterns well-established, cron parsing standard

---

### Phase 4: Tile-Based Dashboard UI
**Rationale:** With backend execution and scheduling complete, build the primary user interface. Pre-fetching pattern prevents N+1 queries before they become a problem.

**Delivers:**
- Tile grid layout (CSS Grid with auto-fit/minmax)
- PluginTile Templ components
- Dashboard handler with optimized query (single query for latest briefings)
- Empty state handling (no plugins enabled, no briefings yet)
- HTMX in-place updates for tile status

**Uses Stack:**
- CSS Grid (native, no JavaScript masonry)
- Templ for components
- HTMX for tile updates

**Implements Architecture:**
- Tile Dashboard component

**Avoids Pitfalls:**
- N+1 queries via pre-fetch pattern
- Tile layout CSS constraints (line-clamp, min-height) for long names
- Nil briefing handling in templates

**Research Needed:** LOW — CSS Grid and HTMX patterns well-documented

---

### Phase 5: Dynamic Settings UI
**Rationale:** Settings come after dashboard because settings management is secondary workflow. JSON Schema validation and form generation are complex but well-bounded.

**Delivers:**
- Settings page handlers (GET/POST)
- Dynamic form generation from JSON Schema
- kaptinlin/jsonschema validation
- Form type coercion (string → integer/boolean)
- HTMX form submission with error handling (preserve user input)
- Plugin enable/disable actions

**Uses Stack:**
- kaptinlin/jsonschema for validation
- Templ for dynamic form rendering
- HTMX for form submission and swap

**Implements Architecture:**
- Settings Service package

**Avoids Pitfalls:**
- Form type coercion prevents "invalid type" errors
- HTMX form state preservation on validation errors
- Settings validation in service layer (not templates)

**Research Needed:** MEDIUM — JSON Schema extension fields (x-component) need testing, form generation pattern is custom

---

### Phase 6: Account Tier Scaffolding
**Rationale:** Tiers are infrastructure for future monetization. Scaffold enforcement now, add payment integration later. Comes last because it's pure constraint checking, not core functionality.

**Delivers:**
- AccountTier models and seeding (free, pro)
- Tier service with constraint checking
- Enforcement in plugin enable handler
- UI messaging for upgrade prompts
- User.AccountTierID relationship

**Uses Stack:**
- GORM for tier models
- Service layer for constraint logic

**Implements Architecture:**
- Tiers package (constraint enforcement)

**Avoids Pitfalls:**
- Tier checks in service layer only (not templates)
- Server-side enforcement (template is UX hint)

**Research Needed:** LOW — Standard constraint checking pattern

---

### Phase Ordering Rationale

**Foundation → Validation → Iteration:**
- **Phase 1-2** establish core architecture with working example (reduces risk)
- **Phase 3-4** add horizontal features (scheduling, UI) once execution proven
- **Phase 5-6** polish and constraints (settings, tiers) after core workflows stable

**Dependency chain:**
- Database schema (Phase 1) → Everything depends on models
- CrewAI integration (Phase 2) → Must prove before scheduling
- Scheduling (Phase 3) → Required before UI shows "next run"
- Dashboard (Phase 4) → Visual validation of scheduling
- Settings (Phase 5) → Requires working plugins to configure
- Tiers (Phase 6) → Constraint layer on top of working system

**Parallelization opportunities:**
- Phases 1-2 are sequential (foundational)
- Phase 3 can start after Phase 2 (scheduling independent of UI)
- Phases 4-5 can partially overlap (dashboard team, settings team)
- Phase 6 is fully independent (add anytime after Phase 1)

**Pitfall mitigation:**
- Schema versioning designed in Phase 1 (prevents mismatch)
- Process management in Phase 2 (prevents orphans)
- Database scheduling in Phase 3 (prevents O(users) Redis)
- Pre-fetch pattern in Phase 4 (prevents N+1)
- Type coercion in Phase 5 (prevents form errors)
- Service layer in Phase 6 (prevents logic duplication)

### Research Flags

**Phases likely needing deeper research during planning:**

- **Phase 2 (CrewAI Sidecar):** CrewAI 2026 production patterns — agent configuration, error handling, memory management, LangChain vs CrewAI tradeoffs. Training data through January 2025, CrewAI evolving rapidly. Use `/gsd:research-phase` before detailed planning.

- **Phase 5 (Settings UI):** JSON Schema extension fields — verify kaptinlin/jsonschema preserves x-component, x-placeholder for UI hints. HTMX form validation error rendering patterns. Custom integration, less library-specific guidance available.

**Phases with standard patterns (skip research-phase):**

- **Phase 1 (Plugin Framework):** GORM migrations, YAML parsing, model relationships — well-documented, stable patterns
- **Phase 3 (Scheduling):** Asynq task patterns, cron parsing, timezone handling — established best practices
- **Phase 4 (Tile Dashboard):** CSS Grid layouts, Templ component patterns, HTMX swaps — mature ecosystem
- **Phase 6 (Account Tiers):** Service layer constraint checking, GORM relationships — fundamental patterns

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | **MEDIUM-HIGH** | Core Go libraries verified (goccy/go-yaml in go.mod, kaptinlin/jsonschema official Google package). Asynq patterns proven in v1.0. CrewAI specifics based on training data, not 2026 production verification. |
| Features | **MEDIUM** | Plugin framework features based on WordPress/Zapier/n8n patterns (training data). Table stakes well-established. CrewAI differentiators logical but not market-validated. WebSearch unavailable for 2026 verification. |
| Architecture | **MEDIUM-HIGH** | Go plugin architecture follows standard patterns. Sidecar pattern well-established in K8s. Database-backed scheduling proven approach. CrewAI HTTP integration straightforward but library-specific internals medium confidence. |
| Pitfalls | **MEDIUM** | High confidence on established anti-patterns (N+1 queries, O(users) scheduler, subprocess management). Medium confidence on plugin-specific patterns (schema versioning, form coercion). CrewAI production patterns less proven. |

**Overall confidence:** **MEDIUM-HIGH**

Research provides strong foundation for roadmap planning. Core Go architecture patterns are well-understood. Main uncertainty is CrewAI integration specifics, which Phase 2 research will address. Stack choices are conservative (proven libraries), features align with competitor analysis, architecture avoids known scalability traps.

### Gaps to Address

**Needs validation during Phase 2 (CrewAI):**
- CrewAI agent configuration best practices for briefing use case — Training data through Jan 2025, library evolving. Research specific patterns for multi-agent workflows (researcher → writer → reviewer chains).
- FastAPI sidecar integration examples with CrewAI — Verify recommended patterns for long-running agent tasks, error propagation, timeout handling.
- CrewAI performance characteristics — Unknown execution time per workflow (2-5s estimate), memory requirements, concurrency limits. Load test during Phase 2.

**Needs validation during Phase 5 (Settings):**
- kaptinlin/jsonschema extension field handling — Documentation claims x-* fields preserved, but test round-trip to verify. Confirm extension fields survive validation.
- HTMX form validation UX patterns — Need standard pattern for displaying jsonschema validation errors (inline per field vs summary). Test user experience early.

**Needs validation during implementation (all phases):**
- Asynq dynamic queue patterns — Docs mention wildcard queue matching (`plugin:*:user:*`) but examples sparse. Verify behavior in Phase 3.
- Schema migration strategy — How to handle breaking changes to plugin settings schema after users have saved configs. Design migration functions early.
- Python virtualenv in Docker — Ensure CrewAI runs in venv, not system Python. Add to Phase 2 Docker setup.

**Defer to post-v1.1:**
- Account tier payment integration — v1.1 scaffolds enforcement only, no Stripe integration. Design payment flow in v1.2+.
- Plugin update mechanism — Version field exists but no automated update flow. Manual plugin updates via deploy acceptable for v1.1.

## Sources

### Primary (HIGH confidence)

**Stack Research:**
- [goccy/go-yaml GitHub](https://github.com/goccy/go-yaml) — YAML parsing library, verified as indirect dependency
- [Google Open Source Blog - JSON Schema package for Go](https://opensource.googleblog.com/2026/01/a-json-schema-package-for-go.html) — Official announcement of kaptinlin/jsonschema
- [Asynq GitHub](https://github.com/hibiken/asynq) — Task queue patterns, validated in v1.0
- [How to Build a Job Queue in Go with Asynq](https://oneuptime.com/blog/post/2026-01-07-go-asynq-job-queue-redis/view) — 2026 Asynq tutorial
- [CSS Grid Layout: Complete Guide 2026](https://devtoolbox.dedyn.io/blog/css-grid-complete-guide) — CSS Grid patterns

**Architecture Research:**
- Existing First Sip codebase (`cmd/server/main.go`, `internal/worker/`, `internal/models/`) — Current patterns analyzed
- `/Users/jim/git/jimdaga/first-sip/TODO.md` — Project redesign context

### Secondary (MEDIUM confidence)

**Features Research:**
- WordPress plugin ecosystem (training data) — Plugin architecture patterns
- Zapier integration model (training data) — SaaS plugin patterns
- n8n self-hosted workflows (training data) — Workflow automation patterns
- Grafana plugin architecture (training data) — Dashboard plugin patterns

**Pitfalls Research:**
- Asynq best practices (training data + recent tutorials) — Scheduler anti-patterns
- GORM documentation (training data) — N+1 query prevention
- HTMX documentation (training data) — Form handling patterns
- Go subprocess management patterns (training data) — Process lifecycle

### Tertiary (LOW confidence, needs validation)

**CrewAI Integration:**
- CrewAI documentation (training data through Jan 2025) — Multi-agent workflows, task orchestration
- CrewAI production patterns — Based on general Python-Go integration patterns, not CrewAI-specific production experience
- FastAPI + CrewAI examples — Inferred from FastAPI patterns + CrewAI docs, not verified production integrations

**JSON Schema Forms:**
- JSON Schema Form libraries (React JSON Schema Form, training data) — Dynamic form generation concepts
- kaptinlin/jsonschema extension field behavior — Documented but not tested in First Sip context

---
*Research completed: 2026-02-13*
*Ready for roadmap: yes*
