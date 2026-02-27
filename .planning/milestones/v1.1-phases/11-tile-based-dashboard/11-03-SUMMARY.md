---
phase: 11-tile-based-dashboard
plan: "03"
subsystem: frontend
tags: [templ, htmx, css, sortablejs, dashboard, tiles, javascript, timezone]

# Dependency graph
requires:
  - phase: 11-01
    provides: Plugin model with icon/tile_size, UserPluginConfig with display_order, migration 000007
  - phase: 11-02
    provides: DashboardHandler, TileStatusHandler, UpdateTileOrderHandler, getDashboardTiles query, TileViewModel
provides:
  - DashboardPage templ component with tile grid, time-aware greeting, and date header
  - TileCard component with embedded collapsed/expanded content (no server round-trip on expand)
  - TileGrid with SortableJS drag-and-drop persisting via POST /api/tiles/order
  - TileOnboarding empty state with link to settings
  - TileSkeleton loading placeholder
  - JS-based fixed-position tooltip (escapes overflow:hidden parent)
  - Browser timezone auto-detection via Intl API with POST /api/user/timezone
  - UpdateTimezoneHandler endpoint for auto-detecting browser timezone
  - Tile footer with info badge at bottom-left
  - Mobile responsive single-column layout at 640px breakpoint
affects: [12-settings-ui]

# Tech tracking
tech-stack:
  added:
    - "SortableJS 1.15.7 via CDN"
  patterns:
    - "Embed full BriefingContent in hidden div, reveal via CSS class toggle (no server request on expand)"
    - "JS fixed-position tooltip appended to body to escape overflow:hidden parents"
    - "Browser timezone auto-detection: Intl.DateTimeFormat().resolvedOptions().timeZone sent once per session"
    - "SortableJS re-initialized on HTMX swap via htmx.onLoad callback"
    - "HTMX polls every 30s only on tiles with pending/processing status"

key-files:
  created: []
  modified:
    - internal/templates/dashboard.templ
    - internal/templates/dashboard_templ.go
    - internal/templates/layout.templ
    - internal/templates/layout_templ.go
    - static/css/liquid-glass.css
    - internal/dashboard/handlers.go
    - cmd/server/main.go

key-decisions:
  - "JS tooltip over CSS ::after — glass-card overflow:hidden clips CSS pseudo-elements; fixed-position JS tooltip escapes any parent"
  - "Info badge in tile footer (bottom-left) not header — avoids overlap with close button (top-right)"
  - "Browser timezone auto-detection on dashboard load — only fires if user timezone is still UTC default"
  - "Expanded tile gets min-height: 300px for visual distinction even with minimal content"
  - "overflow:hidden removed from .tile-card base, applied only to .tile-expanded state"

patterns-established:
  - "Tooltip pattern: create fixed div in body, position on mouseover via getBoundingClientRect"
  - "Timezone detection: sessionStorage gate prevents repeated POSTs within same browser session"
  - "Tile footer: .tile-footer hidden when tile-expanded via CSS rule"

# Metrics
duration: ~30min (includes UAT debugging session)
completed: 2026-02-22
---

# Phase 11 Plan 03: Tile Dashboard Frontend Summary

**Complete tile-based dashboard frontend: CSS Grid with glass card tiles, expand-in-place with pre-loaded content, SortableJS drag-and-drop, HTMX live polling, JS tooltip, browser timezone auto-detection, and all tile states (collapsed, expanded, skeleton, onboarding, error)**

## Performance

- **Duration:** ~30 min (includes interactive UAT)
- **Started:** 2026-02-22
- **Completed:** 2026-02-22
- **Tasks:** 3 (2 auto + 1 human-verify checkpoint)
- **Files modified:** 7

## Accomplishments
- Rewrote dashboard.templ with DashboardPage, TileGrid, TileCard, TileSkeleton, TileOnboarding components
- Tile expand/collapse is purely client-side — full BriefingContent embedded in hidden div, revealed via CSS class toggle
- SortableJS drag-and-drop with HTMX form submission for order persistence
- HTMX polls every 30s only on pending/processing tiles (no unnecessary polling on completed tiles)
- Added tile grid CSS, skeleton loading animation, tooltip styles, expand animation, mobile responsive breakpoint
- Fixed tooltip clipping (JS fixed-position approach escapes overflow:hidden parents)
- Moved info badge to tile footer (bottom-left) to avoid close button overlap
- Added browser timezone auto-detection with POST /api/user/timezone endpoint
- Human verification completed — all tile states, expand/collapse, tooltip, and timezone working

## Task Commits

1. **Task 1: Add tile grid CSS and SortableJS CDN** — `219d640` (feat)
2. **Task 2: Rewrite dashboard.templ with tile grid components** — `2f997c8` (feat)
3. **Task 3: UAT fixes from human verification** — `57c2407` (fix)

## Files Modified
- `internal/templates/dashboard.templ` — Full rewrite with tile grid components + UAT fixes
- `internal/templates/dashboard_templ.go` — Regenerated from .templ
- `internal/templates/layout.templ` — SortableJS CDN script tag
- `internal/templates/layout_templ.go` — Regenerated from .templ
- `static/css/liquid-glass.css` — Tile grid, skeleton, tooltip, expand, footer, mobile responsive styles
- `internal/dashboard/handlers.go` — UpdateTimezoneHandler for browser timezone detection
- `cmd/server/main.go` — POST /api/user/timezone route

## Decisions Made
- JS fixed-position tooltip instead of CSS ::after (overflow:hidden on glass-card clips pseudo-elements)
- Info badge in tile footer not header (avoids close button overlap)
- Browser timezone auto-detection fires once per session via sessionStorage gate
- Expanded tile min-height: 300px ensures visual distinction

## Issues Encountered
- Tooltip clipped by glass-card overflow:hidden — fixed with JS-based fixed-position tooltip
- Tile expand showed no visual change with minimal content — fixed with min-height
- Info badge overlapped close button in header — moved to footer
- User timezone defaulting to UTC caused wrong greeting — added auto-detection

## User Setup Required
None — timezone detection is automatic on dashboard visit.

## Self-Check: PASSED

All artifacts verified:
- FOUND: DashboardPage, TileGrid, TileCard, TileSkeleton, TileOnboarding in dashboard.templ
- FOUND: tile-grid, tile-footer, tile-tooltip, tile-expanded CSS in liquid-glass.css
- FOUND: SortableJS CDN in layout.templ
- FOUND: UpdateTimezoneHandler in handlers.go
- FOUND: /api/user/timezone route in main.go
- VERIFIED: Human UAT completed with fixes applied

---
*Phase: 11-tile-based-dashboard*
*Completed: 2026-02-22*
