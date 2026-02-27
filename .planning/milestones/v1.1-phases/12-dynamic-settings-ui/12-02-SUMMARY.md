---
phase: 12-dynamic-settings-ui
plan: 02
subsystem: ui
tags: [templ, htmx, glass-morphism, settings, accordion, tag-input, timezone-select, time-picker]

requires:
  - phase: 12-01
    provides: PluginSettingsViewModel, handlers, routes, JSON Schema → FieldViewModel conversion

provides:
  - Full settings page Templ templates with accordion layout and dynamic form rendering
  - AppNavbar shared component used by both dashboard and settings pages
  - Tag input UI component for open-ended array fields (topics)
  - Timezone select with curated IANA timezone list
  - Time picker select with 30-minute slots for HH:MM pattern fields
  - FieldTypeTagInput and FieldTypeTimeSelect field type constants
  - DisplayName (humanized) on PluginSettingsViewModel

affects:
  - 12-03 (if any — future settings improvements)
  - dashboard (navbar now shared)

tech-stack:
  added: []
  patterns:
    - JS-positioned tooltip pattern (escapes overflow:hidden) — same as tile tooltip
    - Tag input pattern: hidden JSON array input + JS badge rendering
    - Consolidated HTMX afterSwap handler (single htmx.on to avoid HTMX v2 listener replacement)
    - AppNavbar shared templ component for consistent nav across all authenticated pages
    - FieldType dispatch pattern extended with FieldTypeTagInput and FieldTypeTimeSelect

key-files:
  created:
    - internal/templates/settings.templ (AppNavbar, SettingsPage, PluginAccordionRow, PluginSettingsForm, PluginStatusSection, renderTimezoneSelect, renderTimeSelectField, renderTagInputField)
  modified:
    - internal/settingsvm/settingsvm.go (added FieldTypeTagInput, FieldTypeTimeSelect, DisplayName field)
    - internal/settings/viewmodel.go (humanizePluginName, fieldTypeFromSchema time/tag detection, JSON array coercion in coerceFormValues, DisplayName population)
    - internal/templates/settings_helpers.go (added timeSlots() helper)
    - internal/templates/dashboard.templ (replaced inline navbar with @AppNavbar("dashboard"))
    - static/css/liquid-glass.css (navbar-links, settings-tooltip, tag input styles)

key-decisions:
  - "AppNavbar shared component with activePage param for consistent nav — eliminates duplicated navbar HTML across pages"
  - "FieldTypeTagInput replaces FieldTypeCheckboxGroup for all array fields — better UX per user feedback; schema validation still enforces constraints at save time"
  - "FieldTypeTimeSelect detected via HH:MM pattern substring in schema.Pattern field — avoids hardcoding field names"
  - "Consolidated three htmx.on('htmx:afterSwap') listeners into one — HTMX v2 replaces previous listeners for same event, only last one ran"
  - "DisplayName via humanizePluginName (kebab-case → Title Case) — no DB change needed, purely presentational"
  - "JS tooltip for .settings-tooltip-trigger matches tile-tooltip pattern — CSS ::after pseudo-elements clipped by overflow:hidden containers"
  - "Tag input hidden field holds JSON array string; coerceFormValues updated to parse JSON array from single form value for array-type fields"

requirements-completed: [SET-01, SET-02, SET-03, SET-05, SET-06]

duration: 35min
completed: 2026-02-23
---

# Phase 12 Plan 02: Dynamic Settings UI — Templates Summary

**Complete settings page UI with accordion layout, tag input for topics, timezone/time selectors, shared AppNavbar, humanized plugin names, and fixed HTMX save behavior**

## Performance

- **Duration:** ~35 min
- **Started:** 2026-02-23T13:45:00Z
- **Completed:** 2026-02-23T14:20:43Z
- **Tasks:** 2 (1 build + 1 UAT fix round)
- **Files modified:** 8

## Accomplishments

- Full settings page with glass-morphism accordion rows for all plugins
- 8 UAT issues identified by user review all resolved in one commit
- Shared `AppNavbar` component extracted — dashboard and settings pages now use identical nav
- Tag input component with JS badge rendering replaces checkbox group for array fields
- Timezone field is a curated `<select>` with 40+ IANA timezones; preferred_time is a 30-minute slot time picker
- HTMX `htmx.on('htmx:afterSwap')` handler consolidation fixed accordion collapse-on-save bug
- JS-positioned tooltip for field descriptions (escapes overflow:hidden, matches tile tooltip pattern)
- Plugin display name humanized: "daily-news-digest" → "Daily News Digest"

## Task Commits

1. **Task 1: Build settings Templ templates** - `7612891` (feat)
2. **Task 2 (UAT fixes): Resolve 8 review issues** - `6ee2db9` (fix)

## Files Created/Modified

- `/Users/jim/git/jimdaga/first-sip/internal/templates/settings.templ` — Full settings page: AppNavbar, SettingsPage, PluginAccordionRow, PluginSettingsForm, PluginStatusSection, timezone select, time select, tag input
- `/Users/jim/git/jimdaga/first-sip/internal/settingsvm/settingsvm.go` — Added FieldTypeTagInput, FieldTypeTimeSelect, DisplayName field on PluginSettingsViewModel
- `/Users/jim/git/jimdaga/first-sip/internal/settings/viewmodel.go` — humanizePluginName, fieldTypeFromSchema pattern detection, JSON array coercion, DisplayName population
- `/Users/jim/git/jimdaga/first-sip/internal/templates/settings_helpers.go` — timeSlots() helper for 30-min time slot select
- `/Users/jim/git/jimdaga/first-sip/internal/templates/dashboard.templ` — Replaced inline navbar with @AppNavbar("dashboard")
- `/Users/jim/git/jimdaga/first-sip/static/css/liquid-glass.css` — .navbar-links, .settings-tooltip, tag input styles (.settings-tag-*)

## Decisions Made

- **AppNavbar shared component** — Extracts nav into one place. `activePage` param drives active link highlight.
- **FieldTypeTagInput replaces FieldTypeCheckboxGroup** — All array fields use tag input per UAT feedback. Predefined enum values (topics schema) still enforced by JSON Schema validation at save time; UX change only.
- **FieldTypeTimeSelect via pattern detection** — Schema's `pattern` field containing `:[0-5][0-9]$` (HH:MM suffix) triggers time picker, avoiding hardcoded field-name checks.
- **Consolidated HTMX afterSwap handler** — HTMX v2 replaces previous listeners for the same event name when using `htmx.on()`. Three separate `htmx.on('htmx:afterSwap', ...)` calls meant only the last one ran, breaking save-then-expand behavior. Merged into one handler.
- **DisplayName as computed field** — `humanizePluginName()` converts kebab-case to Title Case at ViewModel build time. No DB schema change needed.
- **JSON array coercion** — `coerceFormValues` now checks if the single form value for an array field starts with `[` and parses it as JSON. This handles both tag input (JSON array in hidden field) and legacy multi-checkbox (multiple form values).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] HTMX v2 afterSwap listener replacement**
- **Found during:** Task 2 (UAT review)
- **Issue:** Three separate `htmx.on('htmx:afterSwap')` calls — HTMX v2 only keeps the last listener per event, so only the `_expandedRows` restore ran. "Saved ✓" button never re-expanded the row.
- **Fix:** Consolidated all three handlers into a single `htmx.on('htmx:afterSwap')` with `(a)/(b)/(c)/(d)` sections
- **Files modified:** internal/templates/settings.templ
- **Verification:** Build compiles, script logic covers all three original cases
- **Committed in:** 6ee2db9

**2. [Rule 2 - Missing Critical] JSON array coercion for tag input**
- **Found during:** Task 2 (adding tag input)
- **Issue:** `coerceFormValues` read `rawVals[0]` as the full JSON array string for array fields, then passed `[]string{"[\"tech\"]"}` to the validator instead of `[]string{"tech"}`
- **Fix:** Added JSON detection: if `rawVals[0]` starts with `[`, unmarshal as `[]string` before passing to validator
- **Files modified:** internal/settings/viewmodel.go
- **Verification:** Build compiles; coercion now handles both tag-input JSON format and legacy multi-checkbox format
- **Committed in:** 6ee2db9

---

**Total deviations:** 2 auto-fixed (1 bug, 1 missing critical)
**Impact on plan:** Both auto-fixes essential for correct behavior. No scope creep.

## Issues Encountered

None beyond the 8 UAT issues that were the explicit scope of this continuation task.

## Next Phase Readiness

- Phase 12 complete — settings page fully functional with all UAT issues resolved
- Dashboard and settings share consistent AppNavbar component
- Settings backend (Phase 12-01) + UI (Phase 12-02) together deliver complete per-user plugin configuration
- Ready for Phase 13 or any next phase in the roadmap

---
*Phase: 12-dynamic-settings-ui*
*Completed: 2026-02-23*
