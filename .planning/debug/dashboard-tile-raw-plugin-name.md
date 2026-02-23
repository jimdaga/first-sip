---
status: diagnosed
trigger: "Dashboard tiles show raw plugin name 'daily-news-digest' instead of humanized 'Daily News Digest'"
created: 2026-02-23T00:00:00Z
updated: 2026-02-23T00:00:00Z
---

## Current Focus

hypothesis: TileViewModel.PluginName is assigned the raw DB value; no humanized field exists on the struct
test: read tiles/viewmodel.go, dashboard/viewmodel.go, templates, and settings/viewmodel.go
expecting: confirmed — PluginName is set directly from p.name (raw kebab-case); no DisplayName field on TileViewModel
next_action: DONE — root cause confirmed, returning diagnosis

## Symptoms

expected: Dashboard tile header reads "Daily News Digest"
actual: Dashboard tile header reads "daily-news-digest"
errors: none (visual regression only)
reproduction: Load /dashboard with an enabled plugin
started: unknown — likely always; humanization only exists in settings path

## Eliminated

(none — first hypothesis confirmed immediately)

## Evidence

- timestamp: 2026-02-23
  checked: internal/tiles/viewmodel.go
  found: TileViewModel struct has PluginName string and PluginIcon string but NO DisplayName field
  implication: There is nowhere to store a humanized name on the tile VM

- timestamp: 2026-02-23
  checked: internal/dashboard/viewmodel.go lines 137-144 (getDashboardTiles) and 252-259 (GetSingleTile)
  found: tile.PluginName = cfg.PluginName, which comes directly from p.name in the SQL query
  implication: The raw DB value (e.g. "daily-news-digest") is assigned unchanged

- timestamp: 2026-02-23
  checked: internal/settings/viewmodel.go lines 117-120 and 204-207
  found: PluginSettingsViewModel has both PluginName (raw) and DisplayName (humanized via humanizePluginName())
  implication: The fix pattern already exists in the settings path; it just wasn't applied to TileViewModel

- timestamp: 2026-02-23
  checked: internal/templates/dashboard.templ line 141
  found: { tile.PluginName } — template renders the raw PluginName field directly
  implication: If PluginName were humanized, or if a DisplayName field were added and used, the tile would display correctly

- timestamp: 2026-02-23
  checked: internal/settings/viewmodel.go humanizePluginName() lines 516-524
  found: function is unexported (lowercase), lives in the `settings` package
  implication: Cannot be called from `dashboard` package without either exporting it, moving it to a shared package, or duplicating it

## Resolution

root_cause: >
  TileViewModel (internal/tiles/viewmodel.go) has no DisplayName field. In
  internal/dashboard/viewmodel.go, both getDashboardTiles and GetSingleTile assign
  tile.PluginName = cfg.PluginName directly from the raw database value (p.name),
  which is kebab-case (e.g. "daily-news-digest"). The template
  internal/templates/dashboard.templ renders tile.PluginName verbatim. The
  humanizePluginName() helper that converts kebab-case to title-case exists in
  internal/settings/viewmodel.go but is unexported and only called in the settings
  view model builder — never in the dashboard tile path.

fix: NOT APPLIED (diagnose-only mode)

verification: N/A

files_changed: []
