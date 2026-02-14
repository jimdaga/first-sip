# Requirements: First Sip v1.1 — Plugin Architecture

**Defined:** 2026-02-14
**Core Value:** A user's configured briefing plugins run on schedule and their latest results appear automatically on a tile-based dashboard — no manual action needed to receive fresh, personalized briefings every day.

## v1.1 Requirements

Requirements for plugin architecture milestone. Each maps to roadmap phases.

### Plugin Framework

- [ ] **PLUG-01**: Plugin metadata defined in YAML files (name, description, owner, version, capabilities, default config)
- [ ] **PLUG-02**: Plugin registry discovers and loads plugins from `/plugins` directory at startup
- [ ] **PLUG-03**: Plugin, UserPluginConfig, and PluginRun database models with GORM migrations
- [ ] **PLUG-04**: Schema versioning field in plugin metadata — templates handle missing fields with defaults
- [ ] **PLUG-05**: JSON Schema settings definition per plugin (schedule, frequency, plugin-specific inputs)
- [ ] **PLUG-06**: At least one working example plugin (daily news digest) end-to-end

### CrewAI Integration

- [ ] **CREW-01**: FastAPI Python sidecar service with health check endpoint
- [ ] **CREW-02**: Redis Streams for Go → CrewAI communication (publish work, consume results)
- [ ] **CREW-03**: CrewAI multi-agent workflow execution (researcher → writer → reviewer pattern)
- [ ] **CREW-04**: Plugin executor reads from Redis Stream, runs CrewAI workflow, publishes result
- [ ] **CREW-05**: Go worker consumes results from response stream and creates Briefing records
- [ ] **CREW-06**: Timeout handling for long-running AI workflows
- [ ] **CREW-07**: CrewAI pods scale independently from Go workers

### Per-User Scheduling

- [ ] **SCHED-01**: Per-user, per-plugin schedule configuration (cron expression + timezone)
- [ ] **SCHED-02**: Database-backed schedule evaluation (NOT per-user Asynq cron entries)
- [ ] **SCHED-03**: Per-minute scheduler task evaluates which user+plugin pairs are due
- [ ] **SCHED-04**: Timezone-aware schedule matching (user's local time, not server UTC)
- [ ] **SCHED-05**: Remove global cron scheduler — replaced by per-user per-plugin schedules
- [ ] **SCHED-06**: Redis caching for last-run times (reduce DB load on per-minute evaluation)

### Tile-Based Dashboard

- [ ] **TILE-01**: CSS Grid tile layout replacing current dashboard (auto-fit/minmax responsive)
- [ ] **TILE-02**: Each enabled plugin renders as a tile showing: plugin name, latest briefing summary, status
- [ ] **TILE-03**: Tile status displays last run time and next scheduled run
- [ ] **TILE-04**: Pre-fetch latest briefing per plugin in single query (window function, avoid N+1)
- [ ] **TILE-05**: Empty states: no plugins enabled, plugin enabled but no briefings yet
- [ ] **TILE-06**: HTMX in-place updates for tile status changes

### Dynamic Settings UI

- [ ] **SET-01**: Settings page listing all available plugins with enable/disable toggle
- [ ] **SET-02**: Dynamic form generation from plugin's JSON Schema settings definition
- [ ] **SET-03**: kaptinlin/jsonschema validation with inline error display
- [ ] **SET-04**: Form type coercion (HTML string inputs → JSON Schema types: integer, boolean)
- [ ] **SET-05**: Form state preservation on validation errors (re-render with user's input)
- [ ] **SET-06**: Plugin status info on settings page (last run, next run, error count)

### Account Tier Scaffolding

- [ ] **TIER-01**: AccountTier model with free/pro tiers seeded in database
- [ ] **TIER-02**: User.AccountTierID relationship (default: free)
- [ ] **TIER-03**: Tier service with constraint checking (max enabled plugins, max frequency)
- [ ] **TIER-04**: Enforcement in plugin enable handler — reject if tier limit reached
- [ ] **TIER-05**: UI messaging for tier limits (upgrade prompt when limit approached/reached)

## Out of Scope

| Feature | Reason |
|---------|--------|
| Payment/Stripe integration | Tier scaffolding only — billing deferred to v1.2+ |
| Plugin marketplace / third-party plugins | Internal plugins only; marketplace needs security review, code signing |
| Plugin dependencies (one plugin uses another's output) | Coupling nightmare, validate independent plugins first |
| Visual workflow builder | CrewAI is code-first, GUI abstraction loses power |
| Real-time on-demand plugin execution | Defeats batching purpose, scheduled only |
| Plugin detail pages / drill-down views | Tiles with summary only for v1.1 |
| CrewAI memory backend (context across briefings) | High complexity, defer to v1.2+ |
| Multiple plugins beyond daily news digest | Prove architecture with one, add more in v1.2+ |
| Email/push notification of briefings | Not needed for personal use phase |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| PLUG-01 | TBD | Pending |
| PLUG-02 | TBD | Pending |
| PLUG-03 | TBD | Pending |
| PLUG-04 | TBD | Pending |
| PLUG-05 | TBD | Pending |
| PLUG-06 | TBD | Pending |
| CREW-01 | TBD | Pending |
| CREW-02 | TBD | Pending |
| CREW-03 | TBD | Pending |
| CREW-04 | TBD | Pending |
| CREW-05 | TBD | Pending |
| CREW-06 | TBD | Pending |
| CREW-07 | TBD | Pending |
| SCHED-01 | TBD | Pending |
| SCHED-02 | TBD | Pending |
| SCHED-03 | TBD | Pending |
| SCHED-04 | TBD | Pending |
| SCHED-05 | TBD | Pending |
| SCHED-06 | TBD | Pending |
| TILE-01 | TBD | Pending |
| TILE-02 | TBD | Pending |
| TILE-03 | TBD | Pending |
| TILE-04 | TBD | Pending |
| TILE-05 | TBD | Pending |
| TILE-06 | TBD | Pending |
| SET-01 | TBD | Pending |
| SET-02 | TBD | Pending |
| SET-03 | TBD | Pending |
| SET-04 | TBD | Pending |
| SET-05 | TBD | Pending |
| SET-06 | TBD | Pending |
| TIER-01 | TBD | Pending |
| TIER-02 | TBD | Pending |
| TIER-03 | TBD | Pending |
| TIER-04 | TBD | Pending |
| TIER-05 | TBD | Pending |

**Coverage:**
- v1.1 requirements: 36 total
- Mapped to phases: 0 (pending roadmap creation)
- Unmapped: 36

---
*Requirements defined: 2026-02-14*
*Last updated: 2026-02-14*
