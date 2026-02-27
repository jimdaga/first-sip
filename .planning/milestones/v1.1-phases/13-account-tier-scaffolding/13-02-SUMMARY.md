---
phase: 13-account-tier-scaffolding
plan: 02
subsystem: ui
tags: [tier-enforcement, htmx, templ, glass-morphism, settings, pro-page]

# Dependency graph
requires:
  - phase: 13-01
    provides: AccountTier model, TierService with CanEnablePlugin/CanUseFrequency/GetUserTier
  - phase: 12-02
    provides: SettingsPage template structure, settingsvm.PluginSettingsViewModel, accordion row pattern
provides:
  - TierInfo struct and SettingsPageViewModel wrapper in internal/settingsvm
  - Server-side enforcement in TogglePluginHandler (block 4th plugin for free users)
  - Server-side enforcement in SaveSettingsHandler (block sub-24h cron for free users)
  - Plugin counter UI showing N/max plugins enabled with accent color at limit
  - Disabled toggle with tooltip for non-enabled plugins at tier limit
  - Pro hint below cron input for free tier users
  - OOB TierPluginCounter component for HTMX toggle swap
  - /pro coming soon page with email capture form
  - proNotifyHandler logging email via slog (no DB)
affects: [phase-14, any handler touching settings, any template reading SettingsPageViewModel]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - SettingsPageViewModel wraps slice + TierInfo — same separation pattern as Phase 12-01 (viewmodel isolation)
    - OOB HTMX swap for counter sync — TierPluginCounter rendered with hx-swap-oob=true alongside accordion row
    - IsDisabledByTier field propagation — viewmodel builder sets flag, template reads it without knowing tier logic
    - proNotifyHandler as inline func in main.go — no DB for scaffolding, slog.Info logs email

key-files:
  created:
    - internal/templates/pro.templ
    - internal/templates/pro_templ.go
    - internal/templates/plugin_detail.templ
    - internal/templates/plugin_detail_templ.go
    - internal/templates/sidebar.go
  modified:
    - internal/settingsvm/settingsvm.go
    - internal/settings/viewmodel.go
    - internal/settings/handlers.go
    - cmd/server/main.go
    - internal/briefings/handlers.go
    - internal/dashboard/handlers.go
    - internal/dashboard/viewmodel.go
    - internal/templates/settings.templ
    - internal/templates/settings_templ.go
    - internal/templates/settings_helpers.go
    - internal/templates/dashboard.templ
    - internal/templates/dashboard_templ.go
    - internal/templates/history.templ
    - internal/templates/history_templ.go
    - internal/templates/layout.templ
    - internal/templates/layout_templ.go
    - static/css/liquid-glass.css

key-decisions:
  - "SettingsPage signature changed from []PluginSettingsViewModel to SettingsPageViewModel — wraps plugins + TierInfo in single struct for clean template access"
  - "IsFreeUser bool field on PluginSettingsViewModel carries tier context to accordion row for cron hint without passing TierInfo through every level"
  - "proNotifyHandler as inline closure in main.go — no DB table needed for scaffolding, slog.Info sufficient for MVP notification tracking"
  - "Counter accent color at limit uses --accent token (warm orange) not red — limits are value-focused, not error states"

patterns-established:
  - "OOB swap pattern: toggle handler wraps accordion row HTML + TierPluginCounter with hx-swap-oob=true in single response fragment"
  - "IsDisabledByTier propagation: builder sets flag at limit, template renders disabled input without needing tier logic inline"

requirements-completed: [TIER-04, TIER-05]

# Metrics
duration: ~60min
completed: 2026-02-25
---

# Phase 13 Plan 02: Tier Enforcement UI Summary

**Server-side plugin count and frequency gating wired to settings handlers, with counter display, disabled toggles at 3/3 limit, Pro hints on cron inputs, and /pro coming soon page with email capture.**

## Performance

- **Duration:** ~60 min (previous executor ran out of usage; code was complete, this session committed and documented)
- **Started:** 2026-02-25T00:00:00Z
- **Completed:** 2026-02-25T00:00:00Z
- **Tasks:** 2
- **Files modified:** 17

## Accomplishments

- TogglePluginHandler now calls `tierService.CanEnablePlugin` and blocks the 4th plugin enable for free users, re-rendering the accordion row with `IsDisabledByTier=true`
- SaveSettingsHandler calls `tierService.CanUseFrequency` and rejects cron expressions faster than 24h for free users with an inline `FrequencyError` message
- Settings page renders plugin counter (`N/max plugins enabled`) that shifts to `--accent` color at 3/3 limit, with disabled toggles and tooltip for non-enabled plugins
- `/pro` coming soon page renders with value proposition bullets and HTMX email capture form (slog-backed, no DB)
- OOB `TierPluginCounter` component keeps counter in sync on every toggle response

## Task Commits

Each task was committed atomically:

1. **Task 1: Add tier enforcement to handlers and extend viewmodel** - `4ebd78f` (feat)
2. **Task 2: Build tier UI — counter, disabled toggle, Pro badge, coming soon page** - `1427f70` (feat)

**Plan metadata:** (docs commit follows this SUMMARY creation)

## Files Created/Modified

- `internal/settingsvm/settingsvm.go` - Added TierInfo struct, SettingsPageViewModel wrapper, IsDisabledByTier and FrequencyError fields on PluginSettingsViewModel
- `internal/settings/viewmodel.go` - Added BuildTierInfo helper; BuildPluginSettingsViewModels accepts TierInfo and sets IsDisabledByTier at limit
- `internal/settings/handlers.go` - SettingsPageHandler, TogglePluginHandler, SaveSettingsHandler all accept TierService; enforcement added
- `cmd/server/main.go` - tierService constructed via tiers.New(db), passed to handlers; /pro GET and /api/pro/notify POST routes added
- `internal/templates/settings.templ` - SettingsPage accepts SettingsPageViewModel; tier counter, disabled toggle, Pro hint, OOB counter component
- `internal/templates/settings_templ.go` - Compiled output of settings.templ
- `internal/templates/settings_helpers.go` - Helper functions for settings template
- `internal/templates/pro.templ` - /pro coming soon page with email capture form
- `internal/templates/pro_templ.go` - Compiled output of pro.templ
- `internal/templates/plugin_detail.templ` - Plugin detail view template
- `internal/templates/plugin_detail_templ.go` - Compiled output of plugin_detail.templ
- `internal/templates/sidebar.go` - Sidebar helper components
- `internal/templates/dashboard.templ` - Updated for navbar/structure consistency
- `internal/templates/history.templ` - Updated for navbar/structure consistency
- `internal/templates/layout.templ` - Updated for layout consistency
- `static/css/liquid-glass.css` - Added .settings-tier-counter, .settings-toggle-disabled, .settings-pro-hint, .pro-coming-soon, .pro-features, .pro-form and related classes
- `internal/briefings/handlers.go` - Minor consistency updates
- `internal/dashboard/handlers.go` - Minor consistency updates
- `internal/dashboard/viewmodel.go` - Minor consistency updates

## Decisions Made

- SettingsPage signature changed from `[]PluginSettingsViewModel` to `SettingsPageViewModel` — wraps plugins + TierInfo in single struct for clean template access without threading TierInfo through every call site
- `IsFreeUser bool` field on `PluginSettingsViewModel` carries tier context to accordion row for cron hint without exposing the full TierInfo struct to deeply nested components
- `proNotifyHandler` implemented as inline closure in main.go — no DB table needed for scaffolding, `slog.Info` sufficient for MVP-stage notification tracking
- Counter accent color at limit uses `--accent` token (warm orange), not red — tier limits are value-focused upgrade prompts, not error states

## Deviations from Plan

None - plan executed exactly as written. All must_have truths satisfied.

## Issues Encountered

None - previous executor had completed all code changes; this session committed the completed work and created documentation.

## User Setup Required

None - no external service configuration required. The `/api/pro/notify` handler logs email via slog; no environment variables needed.

## Next Phase Readiness

- Tier enforcement is complete at both server and UI layer for plugin count (TIER-04) and frequency (TIER-03)
- /pro coming soon page provides upgrade path for all CTA links
- Phase 14 (integration pipeline gap closure) can proceed — tier scaffolding foundation is solid
- Future: proNotifyHandler can be upgraded to persist emails in a DB table when ready for production email capture

---
*Phase: 13-account-tier-scaffolding*
*Completed: 2026-02-25*
