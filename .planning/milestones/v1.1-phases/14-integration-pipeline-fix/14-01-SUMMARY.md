---
phase: 14-integration-pipeline-fix
plan: 01
subsystem: api
tags: [crewai, redis-streams, jsonb, python, templ, htmx]

# Dependency graph
requires:
  - phase: 09-redis-streams
    provides: sidecar executor.py and Go stream handler that this plan fixes
  - phase: 11-tile-based-dashboard
    provides: TileCard component that received the retry button and error message update

provides:
  - Sidecar wraps CrewAI output as {"summary", "sections[]} valid JSON before Redis Stream publish
  - Go extractContent renders sections array as HTML for tile display
  - Stream handler json.Valid guard prevents invalid JSONB storage
  - Retry button on failed tiles triggers re-execution via existing RunNowHandler
  - Dead code removed: TileSkeleton component and ValidateUserSettings function

affects:
  - 14-02-timezone-migration
  - future plugin implementations that rely on the output format contract

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Sidecar wraps CrewAI raw Markdown output in {"summary", "sections"} JSON before publishing
    - Go extractContent renders sections array as HTML using strings.Builder with html/template escaping
    - json.Valid safety guard in stream handler prevents JSONB storage of malformed payloads

key-files:
  created: []
  modified:
    - sidecar/executor.py
    - internal/dashboard/viewmodel.go
    - internal/streams/handler.go
    - internal/templates/dashboard.templ
    - internal/templates/dashboard_templ.go
  deleted:
    - internal/plugins/validator.go

key-decisions:
  - "Sidecar _wrap_output uses re.split on ^##\\s+ headings to build sections array — simple, lossless, no extra dependency"
  - "Empty/None raw_output falls back to placeholder JSON instead of failing — reserve status=failed for genuine exceptions"
  - "html/template.HTMLEscapeString used in viewmodel.go for section title escaping (not templ import — avoids new dependency in non-template code)"
  - "min() builtin used in stream handler output_preview (Go 1.21+ builtin, project is Go 1.26)"
  - "Retry button reuses existing POST /api/settings/:pluginID/run-now — no new handler needed"
  - "Old malformed runs return empty content (no crash) via JSON parse failure returning empty string"

patterns-established:
  - "Pattern: Sidecar output contract — always publish {summary, sections[]} JSON, never raw Markdown"
  - "Pattern: extractContent renders sections as HTML for @templ.Raw() tile display"

requirements-completed:
  - CREW-05

# Metrics
duration: 2min
completed: 2026-02-26
---

# Phase 14 Plan 01: Integration Pipeline Fix (Output Format) Summary

**Sidecar wraps CrewAI Markdown output as {"summary", "sections[]} JSON fixing PostgreSQL JSONB rejection; Go tile rendering updated to display sections; retry button added to failed tiles**

## Performance

- **Duration:** 2 min
- **Started:** 2026-02-26T14:35:00Z
- **Completed:** 2026-02-26T14:37:00Z
- **Tasks:** 2
- **Files modified:** 5 (+ 1 deleted)

## Accomplishments

- Fixed the core CREW-05 pipeline break: sidecar now publishes valid JSON to Redis Stream instead of raw Markdown that PostgreSQL rejected as invalid JSONB
- Updated Go dashboard viewmodel to parse sections array and render structured HTML for tile display
- Added json.Valid safety guard in stream handler to catch any future malformed payloads before JSONB storage
- Added Retry button to failed tile expanded state with generic user-facing error message
- Removed TileSkeleton (never called) and ValidateUserSettings (never called) dead code

## Task Commits

Each task was committed atomically:

1. **Task 1: Fix sidecar output wrapping and Go-side JSON parsing** - `60c5c07` (feat)
2. **Task 2: Add Retry button to failed tiles and remove dead code** - `60652ec` (feat)

## Files Created/Modified

- `sidecar/executor.py` - Added _wrap_output, _extract_summary, _build_sections methods; execute() now wraps raw CrewAI output before publishing
- `internal/dashboard/viewmodel.go` - Added OutputSection struct; updated PluginRunOutput with Sections field; updated extractContent to render sections as HTML
- `internal/streams/handler.go` - Added json.Valid guard with slog.Warn before JSONB storage
- `internal/templates/dashboard.templ` - Updated error message to generic text; added Retry button; removed TileSkeleton component
- `internal/templates/dashboard_templ.go` - Regenerated via make templ-generate
- `internal/plugins/validator.go` - Deleted (dead code, zero callers)

## Decisions Made

- Used `html/template.HTMLEscapeString` for section title escaping in viewmodel.go rather than importing the templ package into non-template code
- Empty/None sidecar output uses placeholder JSON fallback instead of failing — genuine failures (exceptions, timeouts) still use status="failed"
- `min()` builtin used for output_preview truncation in stream handler warning (Go 1.26 project)
- Retry button reuses existing `/api/settings/:pluginID/run-now` route — no new handler required

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required. Changes take effect on next sidecar deployment.

## Next Phase Readiness

- Output format contract is now enforced end-to-end: sidecar → Redis Stream → JSONB → dashboard tile
- Phase 14-02 can proceed with the timezone migration (remove per-plugin timezone, add user-level settings page)
- Any existing PluginRun records with raw Markdown output will show as empty content (no crash) — users can press Retry to regenerate

---
*Phase: 14-integration-pipeline-fix*
*Completed: 2026-02-26*

## Self-Check: PASSED

- FOUND: sidecar/executor.py
- FOUND: internal/dashboard/viewmodel.go
- FOUND: internal/streams/handler.go
- FOUND: internal/templates/dashboard.templ
- CONFIRMED DELETED: internal/plugins/validator.go
- FOUND: 14-01-SUMMARY.md
- FOUND: commit 60c5c07 (task 1)
- FOUND: commit 60652ec (task 2)
