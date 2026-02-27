---
phase: 12-dynamic-settings-ui
plan: 01
subsystem: ui
tags: [go, gin, htmx, jsonschema, gorm, settings, viewmodel, forms]

# Dependency graph
requires:
  - phase: 11-tile-based-dashboard
    provides: UserPluginConfig model, plugin DB schema, dashboard handler patterns
  - phase: 10-per-user-scheduling
    provides: ValidateCronExpression, CronExpression/Timezone columns on UserPluginConfig
  - phase: 09-crewai-plugin
    provides: worker.EnqueueExecutePlugin for Run Now functionality

provides:
  - settings package with schema-to-form conversion (schemaToFields, coerceFormValues)
  - per-field validation using kaptinlin/jsonschema DetailedErrors()
  - 5 HTTP handlers: GET /settings, toggle, save, validate-field, run-now
  - settingsvm package (ViewModel types, mirrors tiles pattern to avoid import cycle)
  - minimal settings.templ stub enabling full server build

affects:
  - 12-02-PLAN: provides all handler and ViewModel types that Plan 02 templates will render

# Tech tracking
tech-stack:
  added: []
  patterns:
    - settingsvm package pattern: ViewModel types in separate package to break handler→templates→handler import cycle (mirrors internal/tiles)
    - Schema-driven form generation: schemaToFields converts *jsonschema.Schema to []FieldViewModel before passing to Templ (no jsonschema import in templates)
    - Type coercion before schema validation: coerceFormValues handles HTML string→Go type conversion
    - Per-field error extraction via result.DetailedErrors() with JSON pointer paths
    - computeNextRun duplicated in settings to avoid import cycle with dashboard

key-files:
  created:
    - internal/settingsvm/settingsvm.go
    - internal/settings/viewmodel.go
    - internal/settings/handlers.go
    - internal/templates/settings.templ
    - internal/templates/settings_templ.go
  modified:
    - cmd/server/main.go

key-decisions:
  - "settingsvm package for ViewModel types breaks handler→templates→handler import cycle (same pattern as internal/tiles)"
  - "schemaToFields converts *jsonschema.Schema to []FieldViewModel before passing to Templ — templates never import kaptinlin/jsonschema"
  - "compiler.SetPreserveExtra(true) enabled for future x-extension field support without current schema requiring it"
  - "coerceFormValues handles HTML string coercion: integer→Atoi, boolean→absent=false (Pitfall 3), array→rawForm[key]"
  - "computeNextRun duplicated in settings package (not moved to shared package) — avoids import cycle, follows existing dashboard pattern"
  - "Run Now uses saved DB settings, not unsaved form state — documented in handler comment"

patterns-established:
  - "settingsvm package: all ViewModel types shared between handler and template layers live in dedicated package"
  - "schemaToFields: sorted keys for deterministic form order, submittedValues > savedSettings > default priority"
  - "validateAndGetFieldErrors: uses result.DetailedErrors() not result.Errors (keyword-keyed, not field-keyed)"

requirements-completed: [SET-01, SET-02, SET-03, SET-04, SET-06]

# Metrics
duration: 7min
completed: 2026-02-23
---

# Phase 12 Plan 01: Settings Backend Summary

**JSON Schema-driven settings backend with schema-to-form conversion, per-field validation, and 5 HTMX handler endpoints wired into main.go**

## Performance

- **Duration:** 7 min
- **Started:** 2026-02-23T13:33:47Z
- **Completed:** 2026-02-23T13:40:29Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments
- Settings backend package with schema-driven form generation (schemaToFields converts JSON Schema properties to []FieldViewModel with sorted keys, type detection, and value priority chain)
- Type coercion layer (coerceFormValues) handles HTML form string→Go type conversion for integer, boolean, and array fields before schema validation
- Per-field error extraction using kaptinlin/jsonschema's DetailedErrors() returning JSON pointer paths
- All 5 handler endpoints implemented: GET /settings (page), POST toggle, POST save (with validation+coercion), POST validate-field (blur), POST run-now (enqueue)
- Import cycle resolved via settingsvm package (mirrors internal/tiles pattern used for dashboard)

## Task Commits

Each task was committed atomically:

1. **Task 1: Create settings viewmodel with schema-to-form conversion** - `b3a5004` (feat)
2. **Task 2: Create settings handlers and wire routes in main.go** - `9600a15` (feat)

**Plan metadata:** (forthcoming)

## Files Created/Modified
- `internal/settingsvm/settingsvm.go` - FieldViewModel, PluginSettingsViewModel, PluginStatusViewModel types (in separate package to avoid import cycle)
- `internal/settings/viewmodel.go` - schemaToFields, coerceFormValues, validateAndGetFieldErrors, validateSingleField, loadPluginSchema, BuildPluginSettingsViewModels, getPluginStatus
- `internal/settings/handlers.go` - 5 Gin handlers: SettingsPageHandler, TogglePluginHandler, SaveSettingsHandler, ValidateFieldHandler, RunNowHandler
- `internal/templates/settings.templ` - Minimal stub for SettingsPage and PluginAccordionRow (Plan 02 replaces)
- `internal/templates/settings_templ.go` - Generated from settings.templ
- `cmd/server/main.go` - Added settings import and 5 route registrations in protected group

## Decisions Made
- **settingsvm package:** Handler imports templates, templates would import settings for ViewModel types → cycle. Solution: ViewModel types in separate `settingsvm` package that both import. This exactly mirrors `internal/tiles` which was created for the same reason with the dashboard/templates cycle.
- **SetPreserveExtra(true):** Enabled on all schema compilations to future-proof x-extension field support. Current daily-news-digest schema has no x- fields but this avoids a breaking change later.
- **Run Now uses saved DB settings:** The plan's open question — "should Run Now use unsaved form state?" — resolved as: always reads UserPluginConfig.Settings from DB. Documented in handler comment.
- **computeNextRun duplicated:** Kept in internal/settings rather than moved to internal/plugins. Moving would require touching more files; duplication follows existing dashboard pattern.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Created internal/settingsvm package to resolve import cycle**
- **Found during:** Task 2 (Create settings handlers)
- **Issue:** `settings/handlers.go` imports `templates` package; `templates/settings.templ` imports `settings` for ViewModel types → import cycle prevented compilation
- **Fix:** Moved ViewModel types (FieldViewModel, PluginSettingsViewModel, PluginStatusViewModel, etc.) to new `internal/settingsvm` package. `templates` imports `settingsvm`. `settings` package re-exports as type aliases. Exactly mirrors `internal/tiles` pattern established in Phase 11.
- **Files modified:** internal/settingsvm/settingsvm.go (new), internal/settings/viewmodel.go (re-export aliases), internal/templates/settings.templ (import settingsvm)
- **Verification:** `go build ./internal/settings/... && go build ./cmd/server/...` both succeed
- **Committed in:** 9600a15 (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Required for compilation. Pattern follows existing project conventions exactly. No scope creep.

## Issues Encountered
- Import cycle between settings handlers and templates — resolved via settingsvm package (Rule 3 auto-fix, standard project pattern)

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- All backend types and handlers are in place for Plan 02 to implement the full Templ accordion UI
- templates.SettingsPage() and templates.PluginAccordionRow() currently render stub HTML
- Plan 02 replaces stub templates with full accordion UI (glass-card accordion rows, toggle switches, form rendering, status section)
- settingsvm.PluginSettingsViewModel has all fields Plan 02 needs: Fields, HasSchema, HasRequiredFields, Status, SaveSuccess, CronExpression, Timezone

## Self-Check: PASSED

- internal/settingsvm/settingsvm.go: FOUND
- internal/settings/viewmodel.go: FOUND
- internal/settings/handlers.go: FOUND
- internal/templates/settings.templ: FOUND
- internal/templates/settings_templ.go: FOUND
- Commit b3a5004 (Task 1): FOUND
- Commit 9600a15 (Task 2): FOUND
- Route wiring in main.go (settings.SettingsPageHandler): FOUND

---
*Phase: 12-dynamic-settings-ui*
*Completed: 2026-02-23*
