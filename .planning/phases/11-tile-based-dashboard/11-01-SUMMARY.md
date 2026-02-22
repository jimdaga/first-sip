---
phase: 11-tile-based-dashboard
plan: "01"
subsystem: database
tags: [plugins, gorm, migration, postgres, tile-dashboard]

# Dependency graph
requires:
  - phase: 10-per-user-scheduling
    provides: UserPluginConfig with cron_expression/timezone columns and scheduler wiring

provides:
  - Plugin model with icon and tile_size columns backed by migration 000007
  - UserPluginConfig with display_order column for drag-and-drop tile ordering
  - PluginMetadata struct accepts icon and tile_size YAML fields
  - syncPluginToDB persists icon and tile_size from YAML to database
  - daily-news-digest plugin declares icon (newspaper emoji) and tile_size (2x1)
  - Migration 000007 with partial index on user_id/display_order for dashboard queries

affects:
  - 11-02 (tile grid layout rendering)
  - 11-03 (tile HTMX interaction and ordering)
  - all future phases using Plugin or UserPluginConfig models

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Default TileSize to 1x1 in syncPluginToDB when YAML field is empty (defensive default)"
    - "Nullable *int DisplayOrder pattern for optional ordering (nil = unordered)"
    - "IF NOT EXISTS guards on migrations for idempotent column additions"

key-files:
  created:
    - internal/database/migrations/000007_add_tile_fields.up.sql
    - internal/database/migrations/000007_add_tile_fields.down.sql
  modified:
    - internal/plugins/metadata.go
    - internal/plugins/models.go
    - internal/plugins/loader.go
    - plugins/daily-news-digest/plugin.yaml

key-decisions:
  - "DisplayOrder as nullable *int (not int with zero-value) avoids ambiguity between unordered and first position"
  - "TileSize defaults to 1x1 in syncPluginToDB (not in YAML or struct) so plugins without tile_size still get a valid DB value"
  - "Migration uses IF NOT EXISTS guards for idempotent re-runs during development"

patterns-established:
  - "Go struct fields updated BEFORE plugin.yaml (KnownFields(true) rejects unknown YAML keys)"
  - "Partial index on user_plugin_configs uses deleted_at IS NULL AND enabled = true for dashboard query performance"

# Metrics
duration: 2min
completed: 2026-02-22
---

# Phase 11 Plan 01: Tile Dashboard Schema Foundation Summary

**GORM plugin models extended with icon, tile_size, and display_order backed by PostgreSQL migration 000007 with partial index for ordered dashboard queries**

## Performance

- **Duration:** 2 min
- **Started:** 2026-02-22T02:38:06Z
- **Completed:** 2026-02-22T02:39:53Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments

- Added Icon and TileSize fields to PluginMetadata, Plugin model, and syncPluginToDB with defensive 1x1 default
- Added nullable DisplayOrder *int to UserPluginConfig for drag-and-drop tile ordering
- Created migration 000007 adding icon/tile_size to plugins and display_order to user_plugin_configs with partial index
- Updated daily-news-digest plugin.yaml with newspaper emoji icon and 2x1 tile size

## Task Commits

Each task was committed atomically:

1. **Task 1: Add icon/tile_size/display_order to plugin models** - `1ccfe1a` (feat)
2. **Task 2: Create migration 000007 for tile dashboard columns** - `86b1caf` (chore)

**Plan metadata:** (docs commit — see below)

## Files Created/Modified

- `internal/plugins/metadata.go` - Added Icon and TileSize fields with yaml tags to PluginMetadata struct
- `internal/plugins/models.go` - Added Icon, TileSize to Plugin model; DisplayOrder *int to UserPluginConfig
- `internal/plugins/loader.go` - Updated syncPluginToDB to persist icon and tile_size (both create and update paths)
- `plugins/daily-news-digest/plugin.yaml` - Added icon: "📰" and tile_size: "2x1"
- `internal/database/migrations/000007_add_tile_fields.up.sql` - ALTER TABLE for icon, tile_size, display_order + partial index
- `internal/database/migrations/000007_add_tile_fields.down.sql` - DROP COLUMN rollback for all three columns

## Decisions Made

- DisplayOrder as nullable *int (not int with zero-value) avoids ambiguity between "unordered" and "first position"
- TileSize defaults to "1x1" in syncPluginToDB (not in YAML or struct) so plugins without tile_size still get a valid DB value
- Migration uses IF NOT EXISTS guards for idempotent re-runs during development

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- `make test` exits non-zero due to pre-existing `go: no such tool "covdata"` environment issue (unrelated to this plan). The actual test (`TestHandler`) passed. `go build ./...` and `go test ./internal/health/...` both succeed cleanly.

## User Setup Required

None - no external service configuration required. Migration 000007 will run automatically on next application startup via GORM AutoMigrate/golang-migrate.

## Next Phase Readiness

- Schema foundation is complete. Migration 000007 is ready to run.
- Plugin models have icon, tile_size, and display_order fields for the tile grid renderer (Phase 11-02).
- daily-news-digest plugin declares tile metadata that will render correctly after migration runs.
- No blockers.

---
*Phase: 11-tile-based-dashboard*
*Completed: 2026-02-22*

## Self-Check: PASSED

- All 7 files exist on disk
- Commits 1ccfe1a and 86b1caf confirmed in git log
