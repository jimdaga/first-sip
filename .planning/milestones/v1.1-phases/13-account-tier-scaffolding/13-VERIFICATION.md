---
phase: 13-account-tier-scaffolding
verified: 2026-02-25T00:00:00Z
status: passed
score: 13/13 must-haves verified
re_verification: false
gaps: []
human_verification:
  - test: "Enable 3 plugins as a free user, then attempt to enable a 4th"
    expected: "4th plugin toggle is disabled (grayed out) with tooltip explaining the 3-plugin limit; server returns 4th plugin row with IsDisabledByTier rendering"
    why_human: "Requires live DB + actual plugin rows; can't simulate tier state programmatically from static analysis"
  - test: "Submit a cron expression faster than daily (e.g. '*/30 * * * *') as a free user"
    expected: "SaveSettingsHandler returns FrequencyError inline message; accordion row stays expanded with error text visible"
    why_human: "Requires a seeded free-tier user and live HTTP request to verify the error message renders inline"
  - test: "Visit /pro as a logged-in user and submit an email address"
    expected: "Page renders with value bullets and email form; submitting shows 'Thanks! We'll notify you when Pro launches.'"
    why_human: "HTMX form swap requires a running server to verify the response fragment is inserted correctly"
---

# Phase 13: Account Tier Scaffolding Verification Report

**Phase Goal:** Tier-based constraint enforcement for plugin count and frequency limits (scaffolding only, no payment)
**Verified:** 2026-02-25
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| #  | Truth | Status | Evidence |
|----|-------|--------|----------|
| 1  | AccountTier table exists with free and pro rows seeded on startup | VERIFIED | `internal/database/migrations/000008_add_account_tiers.up.sql` creates table; `SeedAccountTiers` in `seed.go` seeds free (3 plugins/24h) and pro (-1 plugins/2h) via `FirstOrCreate`; `main.go` line 91 calls it after `RunMigrations` |
| 2  | New users are assigned free tier on Google OAuth registration | VERIFIED | `internal/auth/handlers.go` lines 46–53: `db.Where("name = ?", "free").First(&freeTier)` then `AccountTierID: freeTierID` set on new `User` struct before `db.Create` |
| 3  | Existing users with NULL AccountTierID are treated as free tier by service | VERIFIED | `internal/tiers/service.go` lines 34–37: `if user.AccountTierID == nil { return s.freeTier() }` — falls back to DB lookup of `name = 'free'` |
| 4  | TierService correctly reports whether a user can enable another plugin (count check) | VERIFIED | `CanEnablePlugin` in `service.go` lines 49–67: queries `UserPluginConfig` count where `enabled = true`, returns `count < MaxEnabledPlugins`; -1 sentinel handled correctly |
| 5  | TierService correctly reports whether a cron expression meets tier frequency minimum (interval check) | VERIFIED | `CanUseFrequency` in `service.go` lines 73–97: two-consecutive-`Next()` calls compute interval; compares to `MinFrequencyHours * time.Hour` |
| 6  | Free user trying to enable a 4th plugin gets a hard block (server rejects + UI shows disabled toggle) | VERIFIED | `TogglePluginHandler` lines 135–157: calls `tierService.CanEnablePlugin`; on `canEnable == false` sets `IsDisabledByTier = true` and re-renders row, returns early; template renders `disabled` input with tooltip |
| 7  | Plugin counter (e.g. 2/3 plugins enabled) is always visible in plugin settings area | VERIFIED | `TierPluginCounter` component in `settings.templ` line 97; rendered unconditionally by `PluginSettingsPage` at line 164 via `@TierPluginCounter(page.TierInfo)` |
| 8  | Counter shifts to accent color at 3/3 limit | VERIFIED | `TierPluginCounter` template lines 104–106: `settings-tier-count-at-limit` class applied when `tierInfo.AtPluginLimit`; CSS line 1932 sets `color: var(--accent)` for that class |
| 9  | Disabled plugins show disabled toggle with hover tooltip at 3/3 | VERIFIED | `PluginAccordionRow` lines 394–401: when `plugin.IsDisabledByTier`, renders `<input disabled>` inside label with class `settings-tooltip-trigger` and `title` attribute; CSS `.settings-toggle-disabled .settings-toggle-slider` sets opacity 0.4 |
| 10 | Cron frequencies faster than once daily show a Pro hint for free users | VERIFIED | `PluginSettingsForm` lines 464–468: `if plugin.IsFreeUser` renders `settings-pro-hint` span with `/pro` link; `IsFreeUser` set in `BuildPluginSettingsViewModels` at line 164 |
| 11 | SaveSettingsHandler rejects cron expressions faster than 24h for free users | VERIFIED | `SaveSettingsHandler` lines 241–258: `tierService.CanUseFrequency` called before cron validation; on rejection sets `FrequencyError` on viewmodel and re-renders with error, returns early |
| 12 | Upgrade CTA links to /pro coming soon page with email capture | VERIFIED | `TierPluginCounter` renders `<a href="/pro">` at limit; `pro-hint` in `PluginSettingsForm` also links to `/pro`; `/pro` route registered in `main.go` line 254 |
| 13 | Pro coming soon page renders with email capture form | VERIFIED | `internal/templates/pro.templ` renders full page with `.pro-features` list, `hx-post="/api/pro/notify"` form, and `#pro-form-result` swap target; `/api/pro/notify` POST route calls `settings.ProNotifyHandler()` which logs email via `slog.Info` |

**Score:** 13/13 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/models/account_tier.go` | AccountTier GORM model | VERIFIED | `type AccountTier struct` with `gorm.Model`, `Name`, `MaxEnabledPlugins`, `MinFrequencyHours` fields |
| `internal/tiers/service.go` | TierService with CanEnablePlugin and CanUseFrequency | VERIFIED | Exports `New`, `GetUserTier`, `CanEnablePlugin`, `CanUseFrequency`, `GetEnabledCount`; all fully implemented |
| `internal/database/migrations/000008_add_account_tiers.up.sql` | SQL migration for account_tiers table and users FK | VERIFIED | Creates `account_tiers` table, index on `deleted_at`, `ALTER TABLE users ADD COLUMN account_tier_id`, index on FK |
| `internal/database/seed.go` | SeedAccountTiers function | VERIFIED | `SeedAccountTiers(db *gorm.DB) error` using `FirstOrCreate` with name lookup; seeds free and pro tiers |
| `internal/settingsvm/settingsvm.go` | TierInfo struct and SettingsPageViewModel wrapper | VERIFIED | `type TierInfo struct` at line 50; `type SettingsPageViewModel struct` at line 59; `IsDisabledByTier bool` and `FrequencyError string` on `PluginSettingsViewModel` |
| `internal/settings/handlers.go` | Tier enforcement in TogglePluginHandler and SaveSettingsHandler | VERIFIED | `CanEnablePlugin` called at line 136; `CanUseFrequency` called at line 242; both handlers accept `*tiers.TierService` |
| `internal/templates/settings.templ` | Counter display, disabled toggle, Pro badge, upgrade link | VERIFIED | `id="tier-plugin-counter"`, `hx-swap-oob="true"`, `IsDisabledByTier` branch, `IsFreeUser` Pro hint, upgrade link |
| `internal/templates/pro.templ` | Pro coming soon page | VERIFIED | `ProComingSoonPage` component with Layout wrapper, `.pro-features` list, email capture form, `#pro-form-result` div |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `internal/tiers/service.go` | `internal/models/account_tier.go` | GORM query for AccountTier by user ID | WIRED | `models.AccountTier` used at lines 40, 114; `models.User` queried at line 30 |
| `internal/auth/handlers.go` | `internal/models/account_tier.go` | Free tier lookup on registration | WIRED | `var freeTier models.AccountTier` and `AccountTierID: freeTierID` at lines 47–59 |
| `cmd/server/main.go` | `internal/database/seed.go` | SeedAccountTiers called after RunMigrations | WIRED | `database.SeedAccountTiers(db)` at line 91, between `RunMigrations` (line 87) and `SeedDevData` (line 97) |
| `internal/settings/handlers.go` | `internal/tiers/service.go` | TierService injected into TogglePluginHandler | WIRED | `tierService.CanEnablePlugin(user.ID)` at line 136 |
| `internal/settings/handlers.go` | `internal/tiers/service.go` | TierService injected into SaveSettingsHandler for frequency check | WIRED | `tierService.CanUseFrequency(user.ID, cronExpression)` at line 242 |
| `internal/templates/settings.templ` | `internal/settingsvm/settingsvm.go` | SettingsPageViewModel with TierInfo drives counter and disabled state | WIRED | `page.TierInfo` passed to `TierPluginCounter` at line 164; `plugin.IsDisabledByTier` at line 394; `plugin.IsFreeUser` at line 464 |
| `cmd/server/main.go` | `internal/tiers/service.go` | TierService constructed and passed to settings handlers | WIRED | `tierService := tiers.New(db)` at line 106; passed to `PluginSettingsPageHandler`, `TogglePluginHandler`, `SaveSettingsHandler` at lines 247–249 |

### Requirements Coverage

| Requirement | Description | Status | Notes |
|-------------|-------------|--------|-------|
| TIER-01 | AccountTier model with free/pro tiers seeded in database | SATISFIED | `account_tier.go`, migration `000008`, `SeedAccountTiers` all verified |
| TIER-02 | User.AccountTierID relationship (default: free) | SATISFIED | `User.AccountTierID *uint` with `gorm:"index"` and `AccountTier` association; free tier assigned on registration |
| TIER-03 | Tier service with constraint checking (max enabled plugins, max frequency) | SATISFIED | `TierService` with `CanEnablePlugin` (count check) and `CanUseFrequency` (cron interval check) fully implemented |
| TIER-04 | Enforcement in plugin enable handler — reject if tier limit reached | SATISFIED | `TogglePluginHandler` calls `CanEnablePlugin`; blocks enabling and re-renders with `IsDisabledByTier=true` |
| TIER-05 | UI messaging for tier limits (upgrade prompt when limit approached/reached) | SATISFIED | Counter, disabled toggle, tooltip, Pro hint on cron input, upgrade link to `/pro`, Pro coming soon page all present |

**Note:** REQUIREMENTS.md tracking table (lines 115–119) still shows all five TIER requirements as `Pending`. The requirements were completed but the tracking table was not updated. This is a documentation-only gap and does not affect the code.

### Anti-Patterns Found

| File | Pattern | Severity | Impact |
|------|---------|----------|--------|
| None detected | — | — | — |

No TODO/FIXME/placeholder patterns found in any of the key phase files. No empty returns or stub implementations detected. The `proNotifyHandler` intentionally logs via `slog.Info` with no DB persistence — this is a documented scaffolding decision, not a stub.

### Human Verification Required

#### 1. 4th Plugin Block End-to-End

**Test:** With a free-tier user who has 3 plugins enabled, navigate to `/settings/plugins` and attempt to click the enable toggle on a 4th plugin.
**Expected:** Toggle is already rendered disabled with `title` tooltip visible on hover. If somehow the request fires, the server returns the same row with the toggle disabled and no DB change occurs.
**Why human:** Requires live DB seeded with a free-tier user and 3 enabled plugins; the disabled state is conditional on DB-backed plugin count.

#### 2. Frequency Gate on Save

**Test:** As a free-tier user, save plugin settings with cron expression `*/30 * * * *` (every 30 minutes).
**Expected:** Accordion row stays expanded with inline error message: "Schedules faster than once daily require Pro. Your tier allows minimum 24h intervals."
**Why human:** Requires a running server with a live DB free-tier session to trigger the handler branch and verify the HTMX outerHTML swap inserts the error correctly.

#### 3. Pro Page Email Capture

**Test:** Visit `/pro`, enter an email, and submit the form.
**Expected:** The `#pro-form-result` div shows "Thanks! We'll notify you when Pro launches." and the form disappears or is replaced.
**Why human:** HTMX `hx-swap="innerHTML"` on `#pro-form-result` requires a running server to verify the response fragment is injected correctly.

### Gaps Summary

No gaps found. All 13 observable truths are verified against actual code. All artifacts exist at sufficient depth (not stubs). All key links are wired — imports, calls, and return values confirmed.

The only administrative item is that REQUIREMENTS.md tracking table was not updated to reflect `Satisfied` status for TIER-01 through TIER-05. This should be updated separately.

---

_Verified: 2026-02-25_
_Verifier: Claude (gsd-verifier)_
