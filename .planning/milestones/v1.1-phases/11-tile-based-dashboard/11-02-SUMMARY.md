---
phase: 11-tile-based-dashboard
plan: "02"
subsystem: api
tags: [go, gin, gorm, postgres, htmx, dashboard, plugins, cron]

# Dependency graph
requires:
  - phase: 11-01
    provides: Plugin model with icon/tile_size, UserPluginConfig with display_order, migration 000007

provides:
  - internal/dashboard package with DashboardHandler, TileStatusHandler, UpdateTileOrderHandler
  - TileViewModel struct with BriefingContent and LastSuccessfulContent for expand-in-place
  - getDashboardTiles: three-query batch (configs + DISTINCT ON latest runs + DISTINCT ON latest successful runs) — no N+1
  - GetSingleTile for per-tile HTMX polling
  - computeNextRun using robfig/cron/v3 with IANA timezone support
  - timeAwareGreeting (morning/afternoon/evening) based on user timezone
  - formatTimingTooltip with formatRelativeTime helper
  - Routes wired: GET /dashboard, GET /api/tiles/:pluginID, POST /api/tiles/order

affects:
  - 11-03 (tile grid Templ components — will update DashboardHandler render call and TileStatusHandler stub)
  - all future phases using dashboard tile data

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Three-query batch pattern: configs + DISTINCT ON latest + DISTINCT ON latest-successful avoids N+1 for failed-run fallback"
    - "Map-based O(1) lookup for latest run data assembly (latestRunMap, latestSuccessMap)"
    - "Package-local render helper mirrors main.go render — no shared dependency needed"
    - "TileStatusHandler as working stub with TODO(11-03) comment — keeps build green between plans"
    - "getAuthUser extracts email from gin context + DB lookup (matches RequireAuth middleware contract)"

key-files:
  created:
    - internal/dashboard/viewmodel.go
    - internal/dashboard/handlers.go
  modified:
    - cmd/server/main.go

key-decisions:
  - "DashboardHandler calls existing DashboardPage(name, email, latestBriefing) signature to keep build green until Plan 03 updates template"
  - "TileStatusHandler is a working stub (returns 200 OK) — Plan 03 provides Templ tile component to render"
  - "UpdateTileOrderHandler skips malformed plugin_id values rather than failing the whole request"

patterns-established:
  - "getDashboardTiles three-query batch: configs query, DISTINCT ON latest runs, DISTINCT ON latest successful runs"
  - "Nil-safe LatestRunAt pointer from run.CreatedAt (copied to avoid taking address of loop variable)"

# Metrics
duration: 2min
completed: 2026-02-22
---

# Phase 11 Plan 02: Dashboard Backend Summary

**Dashboard package with three-query batch tile assembly (DISTINCT ON latest runs + latest successful runs), time-aware greeting, cron-based next-run computation, and three API routes replacing the inline main.go dashboard lambda**

## Performance

- **Duration:** 2 min
- **Started:** 2026-02-22T02:42:05Z
- **Completed:** 2026-02-22T02:44:34Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments

- Created `internal/dashboard` package with viewmodel.go and handlers.go
- getDashboardTiles executes exactly three queries (configs, DISTINCT ON latest runs, DISTINCT ON latest successful runs) with map-based O(1) assembly — zero N+1
- TileViewModel carries BriefingContent and LastSuccessfulContent for both in-place expand and failed-run error overlay
- DashboardHandler, TileStatusHandler (stub), and UpdateTileOrderHandler wired to routes in main.go
- Old inline dashboard lambda removed from main.go
- go build ./... and all tests pass

## Task Commits

Each task was committed atomically:

1. **Task 1: Create dashboard view model and query layer** - `51673cc` (feat)
2. **Task 2: Create dashboard handlers and wire routes in main.go** - `5c29939` (feat)

**Plan metadata:** (docs commit — see below)

## Files Created/Modified

- `internal/dashboard/viewmodel.go` - TileViewModel struct, getDashboardTiles (3 queries, no N+1), GetSingleTile, computeNextRun, timeAwareGreeting, extractSummary/Content, formatTimingTooltip/formatRelativeTime
- `internal/dashboard/handlers.go` - DashboardHandler, TileStatusHandler, UpdateTileOrderHandler, getAuthUser, render helper
- `cmd/server/main.go` - dashboard import added, inline lambda replaced with dashboard.DashboardHandler(db), three new routes wired

## Decisions Made

- DashboardHandler calls existing `DashboardPage(name, email, latestBriefing)` template signature to keep the build green. Plan 03 will update both template and handler call simultaneously when tile components are ready.
- TileStatusHandler is a working stub (returns 200 OK, calls GetSingleTile for correctness). Plan 03 will render the actual TileCard Templ component.
- UpdateTileOrderHandler skips malformed plugin_id form values (continues) rather than aborting the entire reorder request.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Dashboard backend is complete. Three routes are wired and compile cleanly.
- Plan 03 needs to: (1) create TileCard Templ component, (2) update DashboardPage template signature to accept greeting + []TileViewModel, (3) update DashboardHandler to call new template, (4) update TileStatusHandler to render TileCard.
- No blockers.

---
*Phase: 11-tile-based-dashboard*
*Completed: 2026-02-22*

## Self-Check: PASSED

- internal/dashboard/viewmodel.go exists (437 lines, min 80)
- internal/dashboard/handlers.go exists (136 lines, min 60)
- 11-02-SUMMARY.md exists
- Commits 51673cc and 5c29939 confirmed in git log
