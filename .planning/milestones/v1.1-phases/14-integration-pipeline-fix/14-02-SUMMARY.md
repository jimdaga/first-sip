---
phase: 14-integration-pipeline-fix
plan: 02
subsystem: database, scheduling, settings, ui
tags: [go, gorm, postgresql, htmx, templ, asynq, scheduler, timezone, migration]

requires:
  - phase: 14-01
    provides: Sidecar JSON wrapping, sections parsing, retry button — same phase, previous plan

provides:
  - Migration 000009 dropping timezone column from user_plugin_configs
  - Scheduler using User.Timezone (account-level) instead of per-plugin timezone
  - Account settings page at /settings/account with timezone picker
  - SaveTimezoneHandler persisting user-level timezone via POST /api/user/settings/timezone
  - Dashboard and settings SQL queries sourcing timezone from users table via JOIN

affects:
  - scheduler
  - dashboard
  - settings-ui
  - account-management

tech-stack:
  added: []
  patterns:
    - "Account-level timezone: single user.timezone field drives all scheduling, dashboard next-run display, and settings status — no per-plugin timezone"
    - "Users JOIN in SQL: dashboard and settings queries JOIN users u ON u.id = upc.user_id to source timezone"

key-files:
  created:
    - internal/database/migrations/000009_remove_timezone_from_user_plugin_configs.up.sql
    - internal/database/migrations/000009_remove_timezone_from_user_plugin_configs.down.sql
  modified:
    - internal/plugins/models.go
    - internal/worker/scheduler.go
    - internal/settings/handlers.go
    - internal/settings/viewmodel.go
    - internal/settingsvm/settingsvm.go
    - internal/dashboard/viewmodel.go
    - internal/templates/settings.templ
    - internal/templates/settings_templ.go
    - cmd/server/main.go

key-decisions:
  - "Timezone removed from UserPluginConfig — scheduler now reads cfg.User.Timezone with UTC fallback, simplifying the model"
  - "Account settings page at /settings/account with renderTimezoneSelect reuse — same component as plugin settings"
  - "Dashboard and settings SQL JOIN users table for timezone — single source of truth, no per-plugin column needed"

patterns-established:
  - "Users JOIN pattern: LEFT JOIN users u ON u.id = ? (userID) for settings queries without upc row; JOIN users u ON u.id = upc.user_id for dashboard queries with guaranteed upc row"

requirements-completed:
  - SCHED-04

duration: 18min
completed: 2026-02-26
---

# Phase 14 Plan 02: Timezone Migration and Account Settings Summary

**Migration drops upc.timezone column, scheduler uses User.Timezone, account settings page at /settings/account with timezone picker sourced from users table throughout**

## Performance

- **Duration:** ~18 min
- **Started:** 2026-02-26T14:40:44Z
- **Completed:** 2026-02-26T14:58:44Z
- **Tasks:** 2
- **Files modified:** 9 (plus 2 migrations created)

## Accomplishments
- Migration 000009 drops `timezone` column from `user_plugin_configs` table
- Scheduler now reads `cfg.User.Timezone` with UTC fallback — eliminating stale comment and per-plugin timezone logic
- Dashboard and settings SQL queries JOIN users table and source timezone from `users.timezone`
- New account settings page at `/settings/account` with timezone picker (reuses `renderTimezoneSelect` component)
- `AccountSettingsPageHandler` and `SaveTimezoneHandler` added to settings package
- Per-plugin timezone picker removed from `PluginSettingsForm`
- Settings sidebar now shows "Account" sub-link alongside "Plugin Settings"
- Settings hub tile for "User Preferences" is now a live link to `/settings/account`

## Task Commits

1. **Task 1: Migration, model update, and scheduler timezone fix** - `4a0aed2` (feat)
2. **Task 2: Remove per-plugin timezone from settings/dashboard and add account settings page** - `6d52f5a` (feat)

**Plan metadata:** TBD (docs: complete plan)

## Files Created/Modified
- `internal/database/migrations/000009_remove_timezone_from_user_plugin_configs.up.sql` - DROP COLUMN IF EXISTS timezone
- `internal/database/migrations/000009_remove_timezone_from_user_plugin_configs.down.sql` - ADD COLUMN IF NOT EXISTS timezone
- `internal/plugins/models.go` - Removed Timezone field from UserPluginConfig struct
- `internal/worker/scheduler.go` - cfg.User.Timezone replaces cfg.Timezone; stale comment removed
- `internal/settings/handlers.go` - Removed per-plugin timezone from Save/Toggle handlers; added AccountSettingsPageHandler and SaveTimezoneHandler
- `internal/settings/viewmodel.go` - SQL queries JOIN users for timezone; Timezone removed from vm assignment
- `internal/settingsvm/settingsvm.go` - Removed Timezone field from PluginSettingsViewModel
- `internal/dashboard/viewmodel.go` - SQL queries JOIN users for timezone in both getDashboardTiles and GetSingleTile
- `internal/templates/settings.templ` - Added AccountSettingsPage; removed per-plugin timezone picker; updated sidebar and hub tile
- `cmd/server/main.go` - Registered /settings/account and /api/user/settings/timezone routes

## Decisions Made
- Timezone removed from UserPluginConfig — single source of truth at user level
- Account settings page reuses existing `renderTimezoneSelect` component — no new component needed
- Dashboard SQL uses `JOIN users u ON u.id = upc.user_id` (inner join, guaranteed upc row); settings SQL uses `LEFT JOIN users u ON u.id = ?` (userID param, since LEFT JOIN on upc may produce no row)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Task 1 build fails until Task 2 handler changes applied**
- **Found during:** Task 1 verification
- **Issue:** Removing Timezone from UserPluginConfig struct caused compilation errors in settings/handlers.go which still referenced it
- **Fix:** Proceeded to complete Task 2 handler changes before final verification; build verification done once after both tasks
- **Files modified:** internal/settings/handlers.go (Task 2)
- **Verification:** `go build ./...` passes after Task 2
- **Committed in:** 6d52f5a (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking dependency ordering)
**Impact on plan:** Task 1 model change creates a compile-time dependency on Task 2 handler changes. Minimal scope impact — all changes as planned.

## Issues Encountered
None beyond the expected compile-time dependency between Task 1 (model) and Task 2 (handlers).

## User Setup Required
None - no external service configuration required. Migration 000009 will run automatically on next server start.

## Next Phase Readiness
- Timezone pipeline fully cleaned up: single user-level timezone flows from account settings → scheduler → dashboard display → settings status
- Phase 14 complete — integration pipeline is fixed (Phase 14-01: output format; Phase 14-02: timezone)
- Ready for whatever comes next in the roadmap

---
*Phase: 14-integration-pipeline-fix*
*Completed: 2026-02-26*
