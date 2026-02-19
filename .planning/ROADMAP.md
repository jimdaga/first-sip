# Roadmap: First Sip — Daily Briefing

## Milestones

- ✅ **v1.0 MVP** — Phases 1-7 (shipped 2026-02-13)
- 🚧 **v1.1 Plugin Architecture** — Phases 8-13 (in progress)

## Phases

<details>
<summary>✅ v1.0 MVP (Phases 1-7) — SHIPPED 2026-02-13</summary>

### Phase 1: Authentication
**Goal**: Users can log in with Google OAuth
**Plans**: 2 plans (completed 2026-02-10)

### Phase 2: Database & Models
**Goal**: User and briefing data persists in PostgreSQL
**Plans**: 2 plans (completed 2026-02-11)

### Phase 3: Background Job Infrastructure
**Goal**: Async task processing with Asynq and Redis
**Plans**: 2 plans (completed 2026-02-12)

### Phase 4: Briefing Generation (Mock)
**Goal**: Users can request and receive mock briefings
**Plans**: 2 plans (completed 2026-02-12)

### Phase 5: Briefing Display
**Goal**: Users see briefings in liquid glass UI
**Plans**: 1 plan (completed 2026-02-12)

### Phase 6: Scheduled Generation
**Goal**: Daily briefings generate automatically
**Plans**: 1 plan (completed 2026-02-13)

### Phase 7: Briefing History
**Goal**: Users can browse past briefings
**Plans**: 1 plan (completed 2026-02-13)

</details>

---

### 🚧 v1.1 Plugin Architecture (In Progress)

**Milestone Goal:** Transform First Sip from monolithic briefing generator into a modular, user-configurable plugin platform where each briefing type runs on its own schedule with its own CrewAI workflow.

**Phase Numbering:** Continues from v1.0 (last phase was 7).

**Research Flags:**
- **Phase 9 (CrewAI):** Research complete (2026-02-14)
- **Phase 12 (Settings):** Needs research on JSON Schema extension field preservation

---

#### Phase 8: Plugin Framework Foundation ✅
**Goal**: Plugin metadata system with database-backed registry and end-to-end working example plugin
**Depends on**: v1.0 infrastructure
**Requirements**: PLUG-01, PLUG-02, PLUG-03, PLUG-04, PLUG-05, PLUG-06
**Completed**: 2026-02-14

Plans:
- [x] 08-01-PLAN.md — Plugin metadata struct, YAML loader, directory discovery, and in-memory registry
- [x] 08-02-PLAN.md — GORM models (Plugin, UserPluginConfig, PluginRun), JSON Schema validator, SQL migrations
- [x] 08-03-PLAN.md — Daily news digest example plugin, startup wiring, and seed data

---

#### Phase 9: CrewAI Sidecar Integration ✅
**Goal**: Go-to-CrewAI communication pipeline working end-to-end with real multi-agent workflow execution
**Depends on**: Phase 8
**Requirements**: CREW-01, CREW-02, CREW-03, CREW-04, CREW-05, CREW-06, CREW-07
**Completed**: 2026-02-18

Plans:
- [x] 09-01-PLAN.md — Redis Streams Go infrastructure (publisher, consumer, result handler)
- [x] 09-02-PLAN.md — FastAPI Python sidecar (health endpoints, worker loop, CrewAI executor with timeout)
- [x] 09-03-PLAN.md — CrewAI workflow for daily-news-digest, docker-compose sidecar, K8s deployment
- [x] 09-04-PLAN.md — Gap closure: wire Publisher into worker task handler with PluginRun record creation

---

#### Phase 10: Per-User Scheduling
**Goal**: Database-backed per-user per-plugin scheduling replaces global cron with timezone support
**Depends on**: Phase 9
**Requirements**: SCHED-01, SCHED-02, SCHED-03, SCHED-04, SCHED-05, SCHED-06
**Success Criteria** (what must be TRUE):
  1. Each user can configure unique schedule per plugin (cron expression + timezone stored in UserPluginConfig)
  2. Per-minute scheduler task evaluates database for due user-plugin pairs (NOT per-user Asynq cron entries)
  3. Schedule matching respects user's local timezone (user in PST sees 6 AM PST, not 6 AM UTC)
  4. Global cron scheduler removed — all briefing generation driven by per-user per-plugin schedules
  5. Redis caches last-run times to reduce database load during per-minute evaluation
**Plans**: 2 plans

Plans:
- [ ] 10-01-PLAN.md — Migration, model update, per-minute scheduler engine with timezone-aware cron evaluation and Redis cache
- [ ] 10-02-PLAN.md — Wire scheduler into main.go, remove global scheduler, add critical queue priority

---

#### Phase 11: Tile-Based Dashboard
**Goal**: CSS Grid tile layout displays all enabled plugins with latest briefing and status
**Depends on**: Phase 10
**Requirements**: TILE-01, TILE-02, TILE-03, TILE-04, TILE-05, TILE-06
**Success Criteria** (what must be TRUE):
  1. Dashboard displays tiles in CSS Grid layout (auto-fit/minmax responsive, no JavaScript masonry)
  2. Each enabled plugin renders as tile showing plugin name, latest briefing summary, and status badge
  3. Tile displays last run time and next scheduled run in user's timezone
  4. Dashboard pre-fetches latest briefing per plugin in single query (window function, avoids N+1)
  5. Empty states display gracefully (no plugins enabled shows prompt, plugin enabled but no briefings shows waiting state)
  6. HTMX updates tile status in-place when briefing generation completes
**Plans**: TBD

Plans:
- [ ] 11-01: TBD
- [ ] 11-02: TBD

---

#### Phase 12: Dynamic Settings UI
**Goal**: Settings page with plugin management, auto-generated forms from JSON Schema, and validation
**Depends on**: Phase 11
**Requirements**: SET-01, SET-02, SET-03, SET-04, SET-05, SET-06
**Success Criteria** (what must be TRUE):
  1. Settings page lists all available plugins with enable/disable toggle per plugin
  2. Forms generate dynamically from each plugin's JSON Schema settings definition (no manual form coding)
  3. kaptinlin/jsonschema validates settings with inline field-level error display
  4. HTML form string inputs coerce to JSON Schema types (integer, boolean) without validation errors
  5. Form preserves user's input on validation errors (re-render with submitted values, not defaults)
  6. Plugin status info displays on settings page (last run, next run, recent error count)
**Research Flag**: Needs research on JSON Schema extension field preservation (x-component, x-placeholder)
**Plans**: TBD

Plans:
- [ ] 12-01: TBD
- [ ] 12-02: TBD

---

#### Phase 13: Account Tier Scaffolding
**Goal**: Tier-based constraint enforcement for plugin count and frequency limits (scaffolding only, no payment)
**Depends on**: Phase 12
**Requirements**: TIER-01, TIER-02, TIER-03, TIER-04, TIER-05
**Success Criteria** (what must be TRUE):
  1. AccountTier model exists with free and pro tiers seeded in database
  2. User model has AccountTierID relationship (defaults to free tier on registration)
  3. Tier service checks constraints (max enabled plugins, max frequency) server-side
  4. Plugin enable handler rejects enable request when user reaches tier limit
  5. UI displays upgrade messaging when user approaches or reaches tier limit
**Plans**: TBD

Plans:
- [ ] 13-01: TBD
- [ ] 13-02: TBD

---

## Progress

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. Authentication | v1.0 | 2/2 | Complete | 2026-02-10 |
| 2. Database & Models | v1.0 | 2/2 | Complete | 2026-02-11 |
| 3. Background Job Infrastructure | v1.0 | 2/2 | Complete | 2026-02-12 |
| 4. Briefing Generation (Mock) | v1.0 | 2/2 | Complete | 2026-02-12 |
| 5. Briefing Display | v1.0 | 1/1 | Complete | 2026-02-12 |
| 6. Scheduled Generation | v1.0 | 1/1 | Complete | 2026-02-13 |
| 7. Briefing History | v1.0 | 1/1 | Complete | 2026-02-13 |
| 8. Plugin Framework Foundation | v1.1 | 3/3 | Complete | 2026-02-14 |
| 9. CrewAI Sidecar Integration | v1.1 | 4/4 | Complete | 2026-02-18 |
| 10. Per-User Scheduling | v1.1 | 0/2 | Not started | - |
| 11. Tile-Based Dashboard | v1.1 | 0/2 | Not started | - |
| 12. Dynamic Settings UI | v1.1 | 0/2 | Not started | - |
| 13. Account Tier Scaffolding | v1.1 | 0/2 | Not started | - |

---
*Created: 2026-02-10*
*Last updated: 2026-02-19 after Phase 10 planning complete*
