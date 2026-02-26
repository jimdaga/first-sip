# Phase 15: Verification & Documentation Closure - Research

**Researched:** 2026-02-26
**Domain:** Documentation audit, static code verification, requirements tracking
**Confidence:** HIGH

## Summary

Phase 15 is a pure documentation and verification phase — no new code is written. The goal is to close two classes of open gaps from the v1.1 milestone audit: (1) Phases 10 and 11 both completed their code plans but never received a VERIFICATION.md, leaving 11 requirements (SCHED-01/02/03/05/06 + TILE-01/02/03/04/05/06) as orphaned verifications; (2) REQUIREMENTS.md has 18 checkboxes that are unchecked despite the underlying code being complete and verified in other phase VERIFICATION.md files.

Both gaps are documentation-only. The code has been audited against the codebase during this research session and confirmed to be present and correct. The primary planning challenge is structuring two VERIFICATION.md files (one for Phase 10, one for Phase 11) that follow the project's established Observable Truths pattern, plus a targeted REQUIREMENTS.md update.

**Primary recommendation:** Plan two tasks — one VERIFICATION.md per phase — plus a final REQUIREMENTS.md patch task. No code changes, no migration needed, no risk to build green state.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| SCHED-01 | Per-user, per-plugin schedule configuration (cron expression + timezone) | Confirmed in codebase: `CronExpression` field on `UserPluginConfig`; `timezone` sourced from `users` table via JOIN; migration 000006 adds the column; `ValidateCronExpression` validates at write time |
| SCHED-02 | Database-backed schedule evaluation (NOT per-user Asynq cron entries) | Confirmed: `handlePerMinuteScheduler` queries DB for enabled configs, evaluates each with `isDue`; no per-user Asynq cron registrations exist |
| SCHED-03 | Per-minute scheduler task evaluates which user+plugin pairs are due | Confirmed: `StartPerMinuteScheduler` registers `* * * * *` Asynq cron; `handlePerMinuteScheduler` is the handler; `TaskPerMinuteScheduler` constant exists in `tasks.go` |
| SCHED-05 | Remove global cron scheduler — replaced by per-user per-plugin schedules | Confirmed: zero grep matches for `StartScheduler` (bare), `TaskScheduledBriefingGeneration`, `handleScheduledBriefingGeneration`, `BriefingSchedule`, `BriefingTimezone` across entire codebase; 10-UAT.md test 8 verified this |
| SCHED-06 | Redis caching for last-run times (reduce DB load on per-minute evaluation) | Confirmed: `lastRunHashKey = "scheduler:last_run"` Redis hash; `getLastRun` uses `HGet`; `setLastRun` uses `HSet`; cold-cache protection: zero lastRunAt treated as one minute ago |
| TILE-01 | CSS Grid tile layout replacing current dashboard (auto-fit/minmax responsive) | Confirmed: `.tile-grid` CSS class uses `grid-template-columns: repeat(auto-fit, minmax(280px, 1fr))` in `liquid-glass.css` line 928 |
| TILE-02 | Each enabled plugin renders as a tile showing: plugin name, latest briefing summary, status | Confirmed: `TileCard` in `dashboard.templ` renders `tile-name`, `tile-summary-text` from `BriefingSummary`, `tile-status-icon` for pending/error states |
| TILE-03 | Tile status displays last run time and next scheduled run | Confirmed: `TimingTooltip` computed by `formatTimingTooltip()` in `viewmodel.go`; shown via `tile-info-badge` with `data-tooltip`; format: "Last run: X ago · Next: Y" |
| TILE-04 | Pre-fetch latest briefing per plugin in single query (window function, avoid N+1) | Confirmed: `getDashboardTiles` runs exactly 3 queries using PostgreSQL `DISTINCT ON (plugin_id)` for both latest runs and latest successful runs; map-based O(1) assembly |
| TILE-05 | Empty states: no plugins enabled, plugin enabled but no briefings yet | Confirmed: `TileOnboarding` for no-plugins-enabled state; tile collapsed/expanded content shows "Your first briefing is scheduled for..." or "will run soon" when no `BriefingSummary` |
| TILE-06 | HTMX in-place updates for tile status changes | Confirmed: `TileCard` conditionally adds `hx-get="/api/tiles/{pluginID}"` + `hx-trigger="every 30s"` + `hx-swap="outerHTML settle:0.6s"` only when `LatestRunStatus == "pending" or "processing"` |
</phase_requirements>

## Standard Stack

### Core
| Tool | Version | Purpose | Why Standard |
|------|---------|---------|--------------|
| Existing VERIFICATION.md pattern | v1 project standard | Document observable truths against codebase | Already established in Phases 12, 13, 14 — planner must follow exactly |
| REQUIREMENTS.md checkbox syntax | GitHub Flavored Markdown | `[x]` vs `[ ]` | Existing format, update in-place |

### Supporting
| Tool | Version | Purpose | When to Use |
|------|---------|---------|-------------|
| `grep`/codebase inspection | N/A | Evidence collection for VERIFICATION.md | Every truth claim needs a file path + line number citation |
| 10-UAT.md | v1 project artifact | Existing acceptance test results for Phase 10 | Contains 8 passing tests that can be promoted to VERIFICATION.md truths |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Static code verification | Running live tests | Static verification is the project standard; live tests require infra. Static is faster and correct for this phase. |

## Architecture Patterns

### VERIFICATION.md Structure (project standard)

Every VERIFICATION.md in this project uses this format, taken from Phases 12, 13, and 14:

```
---
phase: XX-phase-name
verified: ISO-8601-timestamp
status: passed
score: N/N must-haves verified
re_verification: false|null
gaps: []
human_verification:
  - test: "..."
    expected: "..."
    why_human: "..."
---

# Phase N: Phase Name Verification Report

**Phase Goal:** [one sentence]
**Verified:** [date]
**Status:** PASSED
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | [claim] | VERIFIED | [file:line evidence] |

**Score:** N/N truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| [path] | [what it provides] | VERIFIED | [substantive details] |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|

### Requirements Coverage

| Requirement | Description | Status | Notes |
|-------------|-------------|--------|-------|

### Anti-Patterns Found

[table or "No anti-patterns found"]

### Human Verification Required

[list of tests requiring live infra, or "None — all truths verifiable statically"]

### Summary

[prose summary]
```

### REQUIREMENTS.md Checkbox Update Pattern

The update is mechanical: change `- [ ]` to `- [x]` for the 18 satisfied requirements. The traceability table at the bottom also needs the `Status` column updated from `Pending` to `Satisfied` and the coverage count updated from 18 to 36.

Current state in REQUIREMENTS.md:
- Lines showing `- [ ]` for: CREW-05, SCHED-01/02/03/04/05/06, TILE-01/02/03/04/05/06, TIER-01/02/03/04/05 (18 total)
- Coverage line reads: `Satisfied: 18 (Phases 8, 9, 12 verified + checked)`

Target state:
- All 36 requirements show `- [x]`
- Coverage line reads: `Satisfied: 36 (all phases complete)`
- Traceability table: update "Pending" rows to "Satisfied"

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Evidence for truths | Manual memory recall | Direct file reads + line citations | Phase 14 VERIFICATION.md pattern cites `file.go lines N-M` for every truth claim |
| VERIFICATION.md format | Custom format | Copy the Phase 13 or 14 VERIFICATION.md structure exactly | Consistency matters — planner will follow Phase 13 template precisely |

**Key insight:** The verifier role's value is accurate file/line citations. Do not paraphrase; quote file paths and line numbers from actual codebase inspection.

## Common Pitfalls

### Pitfall 1: SCHED-04 Confusion
**What goes wrong:** Phase 15's requirement list is SCHED-01/02/03/05/06 — SCHED-04 (timezone-aware matching) was handled by Phase 14. Writing SCHED-04 into the Phase 10 VERIFICATION.md would be wrong.
**Why it happens:** SCHED-04 appears in the roadmap under Phase 10 originally, but was closed by Phase 14.
**How to avoid:** Phase 10 VERIFICATION.md covers only: SCHED-01, SCHED-02, SCHED-03, SCHED-05, SCHED-06. SCHED-04 stays in Phase 14 VERIFICATION.md.
**Warning signs:** If the Requirements Coverage table in Phase 10 VERIFICATION.md includes SCHED-04, it is wrong.

### Pitfall 2: Stale Column Reference in Verification
**What goes wrong:** Phase 10's migration 000006 added a `timezone` column to `user_plugin_configs`. Phase 14's migration 000009 dropped that column. A Phase 10 VERIFICATION.md truth claim about `UserPluginConfig.Timezone` would reference removed code.
**Why it happens:** Phase 10 implemented per-plugin timezone; Phase 14 removed it and moved timezone to users table.
**How to avoid:** Phase 10 VERIFICATION.md must cite `CronExpression` on `UserPluginConfig` and note that `Timezone` was later migrated to `users` table by Phase 14. Do NOT cite `Timezone` as a current `UserPluginConfig` field — it no longer exists.
**Warning signs:** grep for `Timezone` in `plugins/models.go` should return zero results.

### Pitfall 3: Referencing Old UAT.md as Replacement for VERIFICATION.md
**What goes wrong:** Phase 10 has a `10-UAT.md` with 8 passing tests. This is NOT the same as a VERIFICATION.md.
**Why it happens:** UAT.md was a manual testing record; VERIFICATION.md is a static code verification report.
**How to avoid:** The Phase 10 VERIFICATION.md must be a fresh document using the Observable Truths format — it can reference the same truths that UAT.md tested, but must include file/line evidence not present in UAT.md.

### Pitfall 4: Missing Evidence Depth
**What goes wrong:** A truth is listed as "VERIFIED" with only "it was implemented in Plan 10-01" as evidence.
**Why it happens:** It's tempting to cite SUMMARY.md as the source.
**How to avoid:** Evidence must cite actual source files and line numbers. E.g., "VERIFIED — `internal/worker/scheduler.go` line 21: `const lastRunHashKey = "scheduler:last_run"`; line 220: `rdb.HGet(ctx, lastRunHashKey, ...)`"

### Pitfall 5: REQUIREMENTS.md Traceability Table Inconsistency
**What goes wrong:** Checkboxes are updated but the traceability table still shows "Pending" for some rows, or the coverage count is not updated.
**Why it happens:** Two separate updates needed in the same file.
**How to avoid:** The planner must include both the checkbox section AND the traceability table AND the coverage counts as separate explicit update steps.

## Code Examples

### Phase 10 — Evidence Summary for Each Requirement

The following file+line evidence was gathered during research and should be cited in VERIFICATION.md:

**SCHED-01** (per-user per-plugin cron config):
- `internal/plugins/models.go` line 44: `CronExpression string` on `UserPluginConfig`
- `internal/database/migrations/000006_add_schedule_to_user_plugin_configs.up.sql`: adds `cron_expression VARCHAR(100)` column
- `internal/settings/handlers.go` — `SaveSettingsHandler` persists cron expression to DB via `UserPluginConfig.CronExpression`
- Timezone now sourced from `users.timezone` (via JOIN in `dashboard/viewmodel.go` and `settings/viewmodel.go`) — not per-plugin

**SCHED-02** (database-backed evaluation):
- `internal/worker/scheduler.go` lines 77–84: DB query `WHERE enabled = ? AND cron_expression IS NOT NULL` to fetch configs
- No per-user Asynq cron entries anywhere in codebase (confirmed: zero grep matches for `asynq.NewScheduler` with per-user logic)
- `handlePerMinuteScheduler` is the single evaluation point

**SCHED-03** (per-minute scheduler task):
- `internal/worker/scheduler.go` line 54: `scheduler.Register("* * * * *", task)` — fires every minute
- `internal/worker/tasks.go`: `TaskPerMinuteScheduler` constant
- `internal/worker/worker.go`: handler registered in mux
- `cmd/server/main.go`: `StartPerMinuteScheduler(cfg)` called in both server and embedded-worker modes

**SCHED-05** (global cron removed):
- Zero grep matches for `StartScheduler` (bare function), `TaskScheduledBriefingGeneration`, `handleScheduledBriefingGeneration`
- `internal/config/config.go`: no `BriefingSchedule` or `BriefingTimezone` fields
- `10-UAT.md` test 8: confirmed passed ("FOUND: `internal/config/config.go` without BriefingSchedule, BriefingTimezone")

**SCHED-06** (Redis last-run cache):
- `internal/worker/scheduler.go` line 21: `const lastRunHashKey = "scheduler:last_run"`
- Line 220: `rdb.HGet(ctx, lastRunHashKey, fieldKey(userID, pluginID))`
- Line 234: `rdb.HSet(ctx, lastRunHashKey, fieldKey(userID, pluginID), t.Unix())`
- `isDue()` lines 204–205: cold-cache protection (`lastRunAt.IsZero()` → set to one minute ago)

### Phase 11 — Evidence Summary for Each Requirement

**TILE-01** (CSS Grid tile layout):
- `static/css/liquid-glass.css` line 926: `.tile-grid { ... }`
- Line 928: `grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));`
- Line 1204: responsive breakpoint at 640px changes to `minmax(260px, 1fr)`
- `internal/templates/dashboard.templ` line 110–123: `TileGrid` component with `class="tile-grid sortable"`

**TILE-02** (plugin name + briefing summary + status per tile):
- `dashboard.templ` line 144: `{ tile.DisplayName }` in tile header
- Line 170: `tile.BriefingSummary` rendered in collapsed content
- Lines 153–163: `tile-status-icon` shows spinner for pending/processing, error SVG for `HasError`
- `TileViewModel` in `internal/tiles/viewmodel.go` carries all three data points

**TILE-03** (last run time + next scheduled run in tile):
- `internal/dashboard/viewmodel.go` line 394–406: `formatTimingTooltip()` builds "Last run: X · Next: Y" string
- `dashboard.templ` line 146: `data-tooltip={ tile.TimingTooltip }` on `tile-info-badge`
- `computeNextRun()` in `viewmodel.go` line 308 uses user's IANA timezone from `users` table JOIN

**TILE-04** (pre-fetch without N+1):
- `getDashboardTiles()` in `viewmodel.go` lines 70–196: exactly 3 queries
- Query 2 (line 101): `SELECT DISTINCT ON (plugin_id) ...` for latest run
- Query 3 (line 125): `SELECT DISTINCT ON (plugin_id) ...` for latest successful run
- `latestRunMap` and `latestSuccessMap` provide O(1) assembly (no per-tile DB calls)

**TILE-05** (empty states):
- `dashboard.templ` lines 23–27: `if !hasPlugins { @TileOnboarding() }` — no-plugins-enabled state
- `TileOnboarding()` lines 222–227: "Enable your first plugin to get started" with link to settings
- Collapsed content (line 172–176): "Your first briefing is scheduled for {time}" or "will run soon" when no BriefingSummary
- Expanded content (lines 211–215): same waiting states

**TILE-06** (HTMX in-place updates):
- `dashboard.templ` lines 136–140: conditional HTMX polling attributes
- `hx-get={ fmt.Sprintf("/api/tiles/%d", tile.PluginID) }` — only on pending/processing tiles
- `hx-trigger="every 30s"` — 30-second polling interval
- `hx-swap="outerHTML settle:0.6s"` — replaces entire tile in-place
- `TileStatusHandler` in `internal/dashboard/handlers.go` renders `TileCard` for polling response

### REQUIREMENTS.md Update Scope

Requirements currently showing `- [ ]` that should become `- [x]`:
- CREW-05 (Phase 14 verified)
- SCHED-01, SCHED-02, SCHED-03, SCHED-04, SCHED-05, SCHED-06 (Phase 10+14 verified)
- TILE-01, TILE-02, TILE-03, TILE-04, TILE-05, TILE-06 (Phase 11 verified)
- TIER-01, TIER-02, TIER-03, TIER-04, TIER-05 (Phase 13 verified)

Total: 18 checkboxes to check. The coverage count must also update:
- Current: `Satisfied: 18 (Phases 8, 9, 12 verified + checked)`
- Target: `Satisfied: 36 (all 36 requirements satisfied — v1.1 complete)`

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Global cron scheduler (BriefingSchedule env var) | Per-minute Asynq scheduler querying DB | Phase 10 | SCHED-05 is satisfied |
| Per-plugin timezone in UserPluginConfig | Account-level timezone in users table | Phase 14 | SCHED-04 satisfied; Phase 10 VERIFICATION.md must not cite UserPluginConfig.Timezone |
| Old monolithic dashboard | Tile grid with CSS Grid + DISTINCT ON queries | Phase 11 | TILE-01 through TILE-06 satisfied |

**Deprecated/outdated in context of Phase 10 VERIFICATION.md:**
- `UserPluginConfig.Timezone` column: dropped by migration 000009 (Phase 14). Do not reference as current.
- `BriefingSchedule` / `BriefingTimezone` config fields: deleted in Phase 10-02.

## Open Questions

1. **Human verification scope for Phase 10 and 11 VERIFICATION.md files**
   - What we know: Phases 12, 13, and 14 each include human verification items for things requiring a live server
   - What's unclear: Should Phase 10 VERIFICATION.md list human tests (e.g., "schedule fires at correct local time"), or can all truths be verified statically given that Phase 14 VERIFICATION.md already covers timezone behavior?
   - Recommendation: Include at minimum one human verification item for Phase 10 (live scheduler timing) and one for Phase 11 (drag-and-drop ordering persists). These are standard for "requires live DB" scenarios in this project.

2. **Phase 10 UAT.md vs VERIFICATION.md relationship**
   - What we know: `10-UAT.md` exists with 8 passing tests; `11-tile-based-dashboard` had no UAT.md
   - What's unclear: Should the Phase 10 VERIFICATION.md cite UAT.md test results, or be fully independent?
   - Recommendation: VERIFICATION.md is independent (cites source files), but may note UAT.md as corroborating evidence. Do not replace file/line evidence with UAT.md citations.

## Sources

### Primary (HIGH confidence)
- Direct codebase inspection:
  - `/internal/worker/scheduler.go` — SCHED-01/02/03/05/06 evidence
  - `/internal/plugins/models.go` — UserPluginConfig struct (CronExpression, no Timezone)
  - `/internal/database/migrations/000006_add_schedule_to_user_plugin_configs.up.sql` — migration evidence
  - `/internal/templates/dashboard.templ` — TILE-01/02/03/05/06 evidence
  - `/static/css/liquid-glass.css` lines 926-928 — CSS Grid TILE-01 evidence
  - `/internal/dashboard/viewmodel.go` — TILE-03/04/06 evidence (DISTINCT ON, formatTimingTooltip)
  - `/internal/tiles/viewmodel.go` — TileViewModel struct
- Existing phase artifacts:
  - `.planning/phases/10-per-user-scheduling/10-UAT.md` — 8/8 passing tests
  - `.planning/phases/10-per-user-scheduling/10-01-SUMMARY.md` — key decisions
  - `.planning/phases/10-per-user-scheduling/10-02-SUMMARY.md` — key decisions
  - `.planning/phases/11-tile-based-dashboard/11-01/02/03-SUMMARY.md` — key decisions
  - `.planning/phases/12-dynamic-settings-ui/12-VERIFICATION.md` — format template
  - `.planning/phases/13-account-tier-scaffolding/13-VERIFICATION.md` — format template (most detailed)
  - `.planning/phases/14-integration-pipeline-fix/14-VERIFICATION.md` — format template + SCHED-04 already verified

### Secondary (MEDIUM confidence)
- ROADMAP.md Phase 10/11 success criteria — defines what the VERIFICATION.md must confirm
- REQUIREMENTS.md traceability table — defines which IDs belong to which phase

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — project's own verification pattern is fully documented in Phases 12-14
- Architecture: HIGH — VERIFICATION.md format is established, no ambiguity
- Pitfalls: HIGH — researched from actual codebase state (migrations, struct fields, grep results)
- Code evidence: HIGH — all citations verified against live files during research

**Research date:** 2026-02-26
**Valid until:** Indefinite — this is a pure documentation phase; evidence does not go stale

---

## Planning Notes for Planner

Phase 15 should be planned as **2-3 tasks**:

**Option A — 3 tasks:**
1. Write Phase 10 VERIFICATION.md (SCHED-01/02/03/05/06)
2. Write Phase 11 VERIFICATION.md (TILE-01/02/03/04/05/06)
3. Update REQUIREMENTS.md (18 checkbox checks + traceability table + coverage counts)

**Option B — 2 tasks:**
1. Write Phase 10 + Phase 11 VERIFICATION.md files (both in one task)
2. Update REQUIREMENTS.md

Recommendation: **Option A (3 tasks)** — separates concerns, each task has clear atomic success criteria, and the VERIFICATION.md files are substantive enough to warrant individual commits.

Each task should end with a `make templ-generate` check if no templates changed (to confirm build stays green), or at minimum a `go build ./...` verification step.

**No code changes required.** The build should remain green throughout.
