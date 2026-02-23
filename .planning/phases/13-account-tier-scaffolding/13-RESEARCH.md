# Phase 13: Account Tier Scaffolding - Research

**Researched:** 2026-02-23
**Domain:** Go GORM data modeling, service layer enforcement, HTMX-driven UI gating
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- Two tiers: **free** and **pro**, seeded in the database
- Free tier: max **3 enabled plugins**, fastest frequency **once daily**
- Pro tier: **unlimited plugins**, fastest frequency **every 2 hours**
- All new users default to free tier on registration
- Downgrade logic (pro → free) is NOT handled in this phase — scaffolding only
- **Hard block** when free user tries to enable a 4th plugin — must disable another or upgrade
- Enforcement happens **both UI + server**: UI prevents the action, server validates as backup
- Frequency options faster than once daily are **visible but locked with a "Pro" badge** in the cron picker — drives discovery of pro tier
- No grace periods, no soft warnings, no override mechanisms
- **Tone:** Value-focused — "You're using 3/3 plugins. Pro users get unlimited" — show what they're missing
- **Upgrade CTA** links to a "Pro is coming soon" page with email capture (no payment system yet)
- **Placement:** Upgrade messaging lives in the **plugin settings section only** — no dashboard-wide banners
- Plugin counter ("2/3 plugins enabled") always visible in plugin settings area
- Counter uses **normal color until 3/3**, then shifts to warning/accent color — subtle, not traffic-light
- At limit (3/3): enable buttons on other plugins are **disabled with a hover tooltip** explaining the limit
- Enable button is NOT replaced with upgrade CTA — stays as disabled button with tooltip

### Claude's Discretion

- Exact feedback mechanism when blocked (inline message vs toast vs other) — pick what fits existing UI patterns
- "Coming soon" page design and layout
- Tooltip wording and placement
- Exact color shift at 3/3 within the design system tokens

### Deferred Ideas (OUT OF SCOPE)

- Payment integration / Stripe — separate phase
- Downgrade handling (what happens when pro user reverts to free) — future phase
- Usage analytics / tier metrics dashboard — future phase
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| TIER-01 | AccountTier model with free/pro tiers seeded in database | SQL migration pattern (see Architecture §Migration), GORM model with seeding in SeedDevData / startup hook |
| TIER-02 | User.AccountTierID relationship (default: free) | ALTER TABLE migration adds nullable FK to users, GORM BelongsTo preload, default set via DB DEFAULT or seed lookup |
| TIER-03 | Tier service with constraint checking (max enabled plugins, max frequency) | Pure Go service function accepting `*gorm.DB + userID` — queries enabled plugin count, parses cron for min interval |
| TIER-04 | Enforcement in plugin enable handler — reject if tier limit reached | TogglePluginHandler already owns the enable path; add tier service call before DB write, return HTMX fragment with error state |
| TIER-05 | UI messaging for tier limits (upgrade prompt when limit approached/reached) | SettingsPage/PluginAccordionRow already drives from PluginSettingsViewModel; extend with TierInfo struct; disabled toggle with JS tooltip |
</phase_requirements>

---

## Summary

Phase 13 is a **pure Go + GORM + Templ task** — no new infrastructure, no third-party libraries. It requires: (1) a new `account_tiers` table and seeding, (2) a FK column added to `users`, (3) a tier service that enforces constraints, (4) enforcement injected into the existing toggle handler, and (5) UI changes to the settings page.

The codebase is well-structured for this phase. The settings handler (`TogglePluginHandler`) already owns the single entry point for enabling plugins — constraint enforcement can be inserted there with no routing changes. The `PluginSettingsViewModel` struct is the natural home for tier-related display data (counter, limit flag, locked state) since it already feeds the entire settings accordion. The migration system uses numbered SQL files under `internal/database/migrations/`; next up is `000008`.

The most nuanced implementation concern is **frequency enforcement at enable time vs save time**. The cron expression is set in `SaveSettingsHandler`, not `TogglePluginHandler`. The context decision limits frequency options to "once daily" or slower for free tier — this is enforced via UI locking (Pro badge on faster options) and must also be caught server-side in `SaveSettingsHandler`. The tier service must therefore cover two distinct constraint checks: plugin count (in toggle) and minimum cron interval (in save).

**Primary recommendation:** Create `internal/tiers/` package with `TierService` struct encapsulating both constraint checks. Inject into both `TogglePluginHandler` (count check) and `SaveSettingsHandler` (frequency check). Extend `PluginSettingsViewModel` (via `settingsvm` package) with tier display fields. Use one new migration (000008) for `account_tiers` table + `users.account_tier_id` column.

---

## Standard Stack

### Core (no new dependencies needed)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| gorm.io/gorm | v1.31.1 (already in go.mod) | ORM for AccountTier model + FK on User | Consistent with all other models in the project |
| golang-migrate/migrate/v4 | v4.19.1 (already in go.mod) | SQL migration for new tables/columns | Same migration system already used for migrations 001-007 |
| a-h/templ | v0.3.977 (already in go.mod) | Templ component for tier UI, counter, Pro badges | Project-mandated templating system |
| gin-gonic/gin | v1.11.0 (already in go.mod) | HTTP handler layer | No change needed |
| robfig/cron/v3 | already in go.mod (indirect via plugins) | Parse cron expression to determine minimum interval for frequency enforcement | Already used in `plugins/models.go` via `ValidateCronExpression` |

No new dependencies are required for this phase.

---

## Architecture Patterns

### Recommended Project Structure

```
internal/
├── models/
│   ├── user.go           # Add AccountTierID *uint field + AccountTier association
│   └── account_tier.go   # NEW: AccountTier model (ID, Name, MaxPlugins, MinFrequencyHours)
├── tiers/
│   └── service.go        # NEW: TierService — CanEnablePlugin(), CanUseFrequency()
├── settings/
│   ├── handlers.go       # Inject TierService into TogglePluginHandler + SaveSettingsHandler
│   └── viewmodel.go      # Extend BuildPluginSettingsViewModels with tier data
├── settingsvm/
│   └── settingsvm.go     # Add TierInfo struct, extend PluginSettingsViewModel
├── templates/
│   └── settings.templ    # Add counter, disabled state, Pro badge, upgrade link
├── database/
│   ├── migrations/
│   │   ├── 000008_add_account_tiers.up.sql    # NEW
│   │   └── 000008_add_account_tiers.down.sql  # NEW
│   └── seed.go           # Extend SeedDevData to seed free/pro tiers
└── auth/
    └── handlers.go       # Assign free tier on new user registration
```

### Pattern 1: AccountTier Model

**What:** GORM model for the `account_tiers` table. Two rows seeded: `free` and `pro`.
**When to use:** Referenced by User as BelongsTo FK.

```go
// internal/models/account_tier.go
package models

type AccountTier struct {
    ID                 uint   `gorm:"primaryKey"`
    Name               string `gorm:"uniqueIndex;not null"` // "free" or "pro"
    MaxEnabledPlugins  int    `gorm:"not null"`              // -1 = unlimited
    MinFrequencyHours  int    `gorm:"not null"`              // 24 = once daily, 2 = every 2h
}
```

**User model extension:**
```go
// internal/models/user.go — add to User struct
AccountTierID *uint        `gorm:"index"`           // nullable → defaults to free tier lookup
AccountTier   AccountTier  `gorm:"foreignKey:AccountTierID"`
```

Note: `*uint` (pointer) is intentional. It allows `nil` for existing users who predate this migration, which can be treated as "free" in the service layer. No `gorm:"default"` is used here because the DB-level default is set in the migration SQL; the application assigns the FK explicitly on create.

### Pattern 2: TierService

**What:** Stateless service with DB dependency. Two public methods: `CanEnablePlugin` and `CanUseFrequency`.
**When to use:** Called from settings handlers before writing to DB.

```go
// internal/tiers/service.go
package tiers

import (
    "github.com/jimdaga/first-sip/internal/models"
    "github.com/jimdaga/first-sip/internal/plugins"
    cron "github.com/robfig/cron/v3"
    "gorm.io/gorm"
    "time"
)

type TierService struct {
    db *gorm.DB
}

func New(db *gorm.DB) *TierService {
    return &TierService{db: db}
}

// GetUserTier loads the AccountTier for a user. Falls back to free tier if nil.
func (s *TierService) GetUserTier(userID uint) (*models.AccountTier, error) { ... }

// CanEnablePlugin returns true if the user can enable one more plugin under their tier.
// Counts currently enabled user_plugin_configs for this user.
func (s *TierService) CanEnablePlugin(userID uint) (bool, *models.AccountTier, error) { ... }

// CanUseFrequency returns true if the cron expression's shortest interval meets
// the tier's MinFrequencyHours limit.
func (s *TierService) CanUseFrequency(userID uint, cronExpr string) (bool, *models.AccountTier, error) { ... }
```

**Counting enabled plugins:**
```go
var count int64
s.db.Model(&plugins.UserPluginConfig{}).
    Where("user_id = ? AND enabled = true AND deleted_at IS NULL", userID).
    Count(&count)
```

**Checking minimum cron interval (reuses existing parser):**
```go
// Parse the cron expression and check the minimum interval between runs.
// robfig/cron/v3 Schedule.Next() can be used to compute interval:
var cronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
schedule, err := cronParser.Parse(cronExpr)
now := time.Now()
next1 := schedule.Next(now)
next2 := schedule.Next(next1)
interval := next2.Sub(next1)
// Convert tier.MinFrequencyHours to duration and compare
```

### Pattern 3: Enforcement in TogglePluginHandler

**What:** Before the enable write, call `tierService.CanEnablePlugin`. On rejection, return an updated accordion row HTML fragment with the disabled/blocked state.

```go
// In TogglePluginHandler — before config.Enabled = true write:
if newEnabledState {
    canEnable, tier, err := tierService.CanEnablePlugin(user.ID)
    if err != nil {
        c.Status(http.StatusInternalServerError)
        return
    }
    if !canEnable {
        // Re-render the row with AtLimit=true — toggle reverts visually
        vm, _ := BuildSinglePluginSettingsViewModel(...)
        vm.TierInfo = &settingsvm.TierInfo{AtPluginLimit: true, MaxPlugins: tier.MaxEnabledPlugins, ...}
        render(c, templates.PluginAccordionRow(*vm))
        return
    }
}
```

The handler already calls `BuildSinglePluginSettingsViewModel` and `render` — this fits the existing pattern cleanly.

### Pattern 4: Settings Page Tier Display

**What:** A persistent counter ("2/3 plugins enabled") above the plugin list, and per-plugin disabled toggle state when at limit.

**SettingsPage template extension:**
- Add a `TierInfo` field to `PluginSettingsViewModel` (or to a new page-level `SettingsPageViewModel` wrapper)
- Counter lives outside the accordion rows — needs to be at the page level or in a separate HTMX-targetable fragment
- Disabled toggle: HTML `<input disabled>` with `title="You've reached your plan limit..."` for JS tooltip

**Recommended approach:** Pass `TierInfo` at the page level via a new `SettingsPageViewModel` wrapper:
```go
// settingsvm/settingsvm.go addition
type TierInfo struct {
    TierName          string  // "free" or "pro"
    MaxEnabledPlugins int     // -1 = unlimited
    EnabledCount      int     // current count of enabled plugins
    AtPluginLimit     bool    // EnabledCount >= MaxEnabledPlugins (and not unlimited)
    UpgradeURL        string  // "/pro" — "coming soon" page
}

type SettingsPageViewModel struct {
    Plugins  []PluginSettingsViewModel
    TierInfo TierInfo
}
```

The `SettingsPage` templ function signature changes from `(plugins []settingsvm.PluginSettingsViewModel)` to `(page settingsvm.SettingsPageViewModel)`. This is a contained change — only `settings/handlers.go` calls `templates.SettingsPage()`.

**Counter color logic:**
- Normal color: `< MaxEnabledPlugins` → use `--text-secondary`
- At limit: `== MaxEnabledPlugins` → use `--accent` (warm orange, consistent with existing accent usage for attention)
- Do NOT use `--status-unread-text` (red) — the decisions specify "subtle, not traffic-light"

### Pattern 5: Pro Badge on Cron Picker

**What:** Frequency options faster than once daily (intervals < 24h) are rendered with a visual "Pro" badge in the cron picker section of `PluginSettingsForm`.

**Implementation strategy:**
- The current cron input is a free-text `<input type="text">` field
- For free users, add helper text below the cron input: "Schedules faster than once daily require Pro"
- Alternatively, render preset frequency buttons (daily, 12h, 6h, 2h) with Pro badges on the faster ones — this is more discoverable but changes the cron UX
- Simplest approach consistent with existing pattern: add a hint element below the cron input that appears only when user is on free tier, plus a `data-min-frequency-hours` attribute the JS can use to validate client-side before submit

The decisions say "visible but locked with a Pro badge" — the cron field stays editable (user can type any value) but server-side `SaveSettingsHandler` rejects intervals faster than 24h for free users. The Pro badge communicates the constraint without disabling the input entirely.

### Pattern 6: Coming Soon Page

**What:** A simple `/pro` route rendering a "Pro is coming soon" page with email capture form.

```go
// In main.go — add to protected or public routes:
r.GET("/pro", func(c *gin.Context) {
    render(c, templates.ProComingSoonPage())
})
r.POST("/api/pro/notify", handlers.ProNotifyHandler(db))
```

Email capture: store in a simple `pro_waitlist` table (email, created_at) or just log it. The decisions say "email capture" — a minimal table is appropriate. This can be a single migration addition.

### Pattern 7: Seeding Tiers

**What:** `account_tiers` must be seeded before the app creates users. This belongs in a startup seed (separate from dev-only seed data).

```go
// internal/database/seed.go — new function called from main.go after migrations:
func SeedAccountTiers(db *gorm.DB) error {
    tiers := []models.AccountTier{
        {ID: 1, Name: "free",  MaxEnabledPlugins: 3,  MinFrequencyHours: 24},
        {ID: 2, Name: "pro",   MaxEnabledPlugins: -1, MinFrequencyHours: 2},
    }
    for _, t := range tiers {
        db.Where("name = ?", t.Name).FirstOrCreate(&t)
    }
    return nil
}
```

This seed must run in ALL environments (not just development), immediately after `RunMigrations`. Call it from `main.go` between `RunMigrations` and `SeedDevData`.

### Pattern 8: Assigning Free Tier on Registration

**What:** When a new user is created in `HandleCallback`, look up the free tier ID and assign it.

```go
// auth/handlers.go — in the gorm.ErrRecordNotFound branch:
var freeTier models.AccountTier
db.Where("name = ?", "free").First(&freeTier)
user = models.User{
    Email:         gothUser.Email,
    Name:          gothUser.Name,
    LastLoginAt:   &now,
    AccountTierID: &freeTier.ID,  // assign free tier
}
db.Create(&user)
```

Existing users (created before this migration) will have `AccountTierID = NULL` — the tier service must treat `NULL` as `free`.

### Anti-Patterns to Avoid

- **Don't add tier enforcement to the scheduler or worker** — frequency limits are about what users can *configure*, not about blocking scheduled runs. If a cron expression was saved before the tier check existed, the scheduler should still honor it.
- **Don't hardcode tier IDs** — always look up by `name = 'free'` or `name = 'pro'`. IDs are seeded as 1 and 2 but this assumption should not leak into application code.
- **Don't re-render the whole settings page on toggle** — the existing pattern does `hx-swap="outerHTML"` on a single `#plugin-row-N` element. Preserve this pattern; only update the individual row and use HTMX out-of-band swaps (or a separate counter endpoint) if the counter needs updating.
- **Don't add tier checks to SaveSettingsHandler's schema validation path** — add it as a pre-validation guard that returns early, keeping the existing validation logic intact.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Cron interval calculation | Custom time math | `robfig/cron/v3` Schedule.Next() | Already in go.mod; two `Next()` calls give exact interval |
| DB migrations | GORM AutoMigrate | golang-migrate/migrate SQL files | Established project pattern; reversible, auditable |
| Tier constraint query | Complex SQL | Simple `COUNT(*)` via GORM | Enabled plugin count is a trivial query; no need for triggers or DB-level enforcement |

**Key insight:** The tier constraint logic is simple business rules over existing data — it does not require any new infrastructure. Pure Go service functions are the right fit.

---

## Common Pitfalls

### Pitfall 1: HTMX Counter Update on Toggle

**What goes wrong:** The plugin counter ("2/3 plugins enabled") is outside each accordion row, so a `hx-swap="outerHTML"` on a single row won't update the counter.

**Why it happens:** HTMX out-of-band swaps are the standard solution but require `hx-swap-oob` on the server response.

**How to avoid:** Two options:
1. Include the counter in each accordion row response as an OOB swap: add `hx-swap-oob="true"` on a counter fragment returned alongside the row fragment
2. Simpler: render the counter inside a dedicated `<div id="tier-plugin-counter">` and have the toggle response include both the row and the counter via `hx-swap-oob`

Option 1 is more consistent with existing patterns (server returns a single component). Use HTMX OOB swap:
```go
// In TogglePluginHandler response — append OOB counter fragment:
render(c, templates.PluginAccordionRowWithCounterOOB(*vm, tierInfo))
```

Or: wrap the accordion row Templ component with an OOB counter fragment as a sibling in the same response.

**Warning signs:** Counter shows stale count after enable/disable operations.

### Pitfall 2: NULL AccountTierID for Existing Users

**What goes wrong:** Existing dev user and any users created before migration have `account_tier_id = NULL`. Tier service panics or treats them as having no limits.

**Why it happens:** The migration adds a nullable column; existing rows remain NULL.

**How to avoid:** `GetUserTier` always falls back to looking up `name = 'free'` when `AccountTierID` is nil or the preloaded `AccountTier` has zero ID. Never trust that `user.AccountTier` is populated without a Preload call.

**Warning signs:** Free-tier users with NULL tier bypassing plugin count limits.

### Pitfall 3: GORM Preload vs Query-Time Join

**What goes wrong:** `db.First(&user)` does not automatically load `AccountTier` — GORM does not eager-load associations by default.

**Why it happens:** GORM's `BelongsTo` requires explicit `Preload("AccountTier")` or the tier service should load the tier separately by userID.

**How to avoid:** In `GetUserTier`, query the tier directly by joining or by loading `user.AccountTierID` and then doing a second lookup. Avoid relying on association preloads in handlers that use `getAuthUser()` (which does a bare `db.Where().First()`).

**Warning signs:** `user.AccountTier.ID == 0` even after migration, causing "unlimited" behavior for free users.

### Pitfall 4: Toggle Handler Treating Disable as Enable

**What goes wrong:** The existing `TogglePluginHandler` flips the current state — if a plugin is already enabled, clicking toggle *disables* it. Tier check must only apply when *enabling* (transitioning from disabled → enabled).

**Why it happens:** The current code does `config.Enabled = !config.Enabled` — so the tier check must only fire when the new state will be `true`.

**How to avoid:** Determine the new state before the tier check:
```go
newState := !config.Enabled  // or true if config doesn't exist yet
if newState == true {
    // run tier check
}
```

**Warning signs:** Disabling a plugin is rejected with a tier limit error.

### Pitfall 5: Frequency Enforcement Edge Cases

**What goes wrong:** Cron expressions like `*/30 * * * *` (every 30 min) have an interval < 24h. The cron parser is needed to compute the actual interval — do not try to parse the expression as a string pattern.

**Why it happens:** Cron expressions are flexible and cannot be classified by string matching alone.

**How to avoid:** Use `Schedule.Next()` twice to compute the actual minimum interval between firings. The `robfig/cron/v3` parser is already imported in `plugins/models.go` — reuse the same parser in the tier service (unexported var or function parameter).

**Warning signs:** Hourly schedules pass validation for free users.

### Pitfall 6: templ Compilation After Adding Fields to settingsvm

**What goes wrong:** Adding fields to `PluginSettingsViewModel` or creating a new `SettingsPageViewModel` wrapper means all callers of `templates.SettingsPage()` must update their argument.

**Why it happens:** Templ generates typed Go functions — changing the signature breaks compilation.

**How to avoid:** Update `settingsvm.go` first, then `settings/viewmodel.go` (BuildPluginSettingsViewModels), then `settings/handlers.go`, then `settings.templ`. Run `make templ-generate` after each `.templ` edit. The compiler will catch all callers.

**Warning signs:** Compile errors after templ-generate citing wrong argument count to `SettingsPage`.

### Pitfall 7: Pro Waitlist Email Storage

**What goes wrong:** Adding a `pro_waitlist` table in the same migration as `account_tiers` makes the migration harder to roll back independently.

**Why it happens:** Unrelated schema changes bundled together.

**How to avoid:** Either add the waitlist table in a separate migration (000009) or skip DB storage entirely for the "coming soon" page and just log the email submission. For scaffolding purposes, logging is acceptable. Decide at plan time.

---

## Code Examples

### Migration 000008: account_tiers + users FK

```sql
-- 000008_add_account_tiers.up.sql
CREATE TABLE account_tiers (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    name VARCHAR(50) NOT NULL UNIQUE,
    max_enabled_plugins INTEGER NOT NULL, -- -1 = unlimited
    min_frequency_hours INTEGER NOT NULL  -- minimum hours between runs
);

CREATE INDEX idx_account_tiers_deleted_at ON account_tiers(deleted_at);

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS account_tier_id BIGINT REFERENCES account_tiers(id);

CREATE INDEX IF NOT EXISTS idx_users_account_tier_id ON users(account_tier_id);
```

```sql
-- 000008_add_account_tiers.down.sql
ALTER TABLE users DROP COLUMN IF EXISTS account_tier_id;
DROP TABLE IF EXISTS account_tiers;
```

### Tier Service: CanEnablePlugin

```go
// internal/tiers/service.go
func (s *TierService) CanEnablePlugin(userID uint) (bool, *models.AccountTier, error) {
    tier, err := s.GetUserTier(userID)
    if err != nil {
        return false, nil, err
    }
    // Unlimited (-1) — always allowed
    if tier.MaxEnabledPlugins < 0 {
        return true, tier, nil
    }
    // Count currently enabled plugins
    var count int64
    if err := s.db.Model(&plugins.UserPluginConfig{}).
        Where("user_id = ? AND enabled = true AND deleted_at IS NULL", userID).
        Count(&count).Error; err != nil {
        return false, nil, err
    }
    return count < int64(tier.MaxEnabledPlugins), tier, nil
}
```

### Tier Service: GetUserTier with NULL fallback

```go
func (s *TierService) GetUserTier(userID uint) (*models.AccountTier, error) {
    var user models.User
    if err := s.db.Select("account_tier_id").Where("id = ?", userID).First(&user).Error; err != nil {
        return nil, err
    }
    if user.AccountTierID == nil {
        // Legacy/new user without tier assigned — treat as free
        var freeTier models.AccountTier
        if err := s.db.Where("name = ?", "free").First(&freeTier).Error; err != nil {
            return nil, err
        }
        return &freeTier, nil
    }
    var tier models.AccountTier
    if err := s.db.First(&tier, *user.AccountTierID).Error; err != nil {
        return nil, err
    }
    return &tier, nil
}
```

### TierInfo in settingsvm

```go
// Addition to internal/settingsvm/settingsvm.go
type TierInfo struct {
    TierName          string // "free" or "pro"
    MaxEnabledPlugins int    // -1 = unlimited
    EnabledCount      int    // currently enabled count
    AtPluginLimit     bool   // true when EnabledCount >= MaxEnabledPlugins (and not unlimited)
    UpgradeURL        string // "/pro"
}

type SettingsPageViewModel struct {
    Plugins  []PluginSettingsViewModel
    TierInfo TierInfo
}
```

### SettingsPage template counter fragment

```templ
// In settings.templ — above the plugin list
<div id="tier-plugin-counter" class="settings-tier-counter">
    if page.TierInfo.TierName == "free" {
        if page.TierInfo.AtPluginLimit {
            <span class="settings-tier-count settings-tier-count-at-limit">
                { fmt.Sprintf("%d/%d plugins enabled", page.TierInfo.EnabledCount, page.TierInfo.MaxEnabledPlugins) }
            </span>
            <a href={ templ.SafeURL(page.TierInfo.UpgradeURL) } class="settings-upgrade-link">
                Pro users get unlimited — learn more
            </a>
        } else {
            <span class="settings-tier-count">
                { fmt.Sprintf("%d/%d plugins enabled", page.TierInfo.EnabledCount, page.TierInfo.MaxEnabledPlugins) }
            </span>
        }
    }
}
```

### Disabled toggle at limit (in PluginAccordionRow)

```templ
// Replace the toggle input rendering — check if page-level AtPluginLimit applies:
if page.TierInfo.AtPluginLimit && !plugin.Enabled {
    // Render disabled toggle with tooltip
    <label class="settings-toggle settings-tooltip-trigger"
           title="You've reached your 3-plugin limit. Disable another plugin or upgrade to Pro."
           onclick="event.stopPropagation()">
        <input type="checkbox" class="settings-toggle-input" disabled/>
        <span class="settings-toggle-slider settings-toggle-disabled"></span>
    </label>
} else {
    // Normal toggle (existing markup)
}
```

Note: The disabled toggle needs the TierInfo to be accessible in `PluginAccordionRow`. Two approaches:
1. Pass `SettingsPageViewModel` down to `PluginAccordionRow` (requires changing component signature)
2. Add `IsDisabledByTier bool` field to `PluginSettingsViewModel` (set in the viewmodel builder)

Option 2 is simpler and avoids threading the page-level struct through every component. Set `IsDisabledByTier = true` for all non-enabled plugins when `TierInfo.AtPluginLimit == true`.

### Handler injection pattern

```go
// main.go — construct tier service and pass to settings handlers:
tierService := tiers.New(db)

protected.GET("/settings", settings.SettingsPageHandler(db, cfg.PluginDir, tierService))
protected.POST("/api/settings/:pluginID/toggle", settings.TogglePluginHandler(db, cfg.PluginDir, tierService))
protected.POST("/api/settings/:pluginID/save", settings.SaveSettingsHandler(db, cfg.PluginDir, tierService))
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| GORM AutoMigrate | golang-migrate SQL files | Project standard from Phase 1 | Stick with SQL migration files (000008) |
| Per-handler DB queries | ViewModel builders in viewmodel.go | Phase 12 | Tier data should be fetched in viewmodel builder, not handler |
| Single template function signature | SettingsPageViewModel wrapper (proposed) | Phase 13 | Cleaner separation; counter is page-level not per-plugin |

---

## Open Questions

1. **OOB swap for counter on toggle**
   - What we know: HTMX supports `hx-swap-oob` for updating multiple DOM elements in one response
   - What's unclear: Whether returning multiple components from a single Templ render is idiomatic in this codebase (Phase 11 used a different approach for tile status)
   - Recommendation: Use a single Templ component wrapper that renders both the accordion row and the counter fragment; the counter fragment carries `hx-swap-oob="true" id="tier-plugin-counter"`. Implement in Plan 2 once Plan 1 establishes the data model.

2. **Pro waitlist: DB or log-only?**
   - What we know: "Coming soon" page needs email capture; no payment system yet
   - What's unclear: Whether a DB table is worth the migration complexity for scaffolding
   - Recommendation: Log-only for Phase 13 (simpler). A `pro_waitlist` DB table is appropriate only if the planner decides it adds value to the MVP scaffolding. Either approach is acceptable.

3. **Frequency enforcement timing**
   - What we know: The cron expression is saved in `SaveSettingsHandler`, not `TogglePluginHandler`. Free tier minimum interval is 24h.
   - What's unclear: Should frequency enforcement also prevent enabling a plugin that already has a too-fast cron expression saved (edge case: saved before enforcement existed)?
   - Recommendation: For scaffolding, enforce only at *save time* (when cron expression is submitted). Do not retroactively block enablement based on existing cron data — the constraint decisions say "hard block when trying to enable a 4th plugin" (count constraint), but frequency constraint applies to the settings save path.

---

## Sources

### Primary (HIGH confidence)

- Codebase inspection (`internal/models/user.go`, `internal/plugins/models.go`, `internal/settings/handlers.go`, `internal/settings/viewmodel.go`, `internal/settingsvm/settingsvm.go`, `internal/templates/settings.templ`, `internal/database/migrations/*.sql`, `internal/database/seed.go`, `cmd/server/main.go`) — all patterns verified from source
- `go.mod` — confirmed all dependencies already present; no new packages needed
- `13-CONTEXT.md` — user decisions locked; frequency thresholds, tier definitions, UX behavior

### Secondary (MEDIUM confidence)

- GORM documentation patterns for BelongsTo FK with nullable pointer field — standard GORM usage verified against codebase existing patterns (Plugin → User FK in plugins/models.go uses same nullable pointer approach for optional associations)
- robfig/cron/v3 `Schedule.Next()` pattern for interval calculation — confirmed available via existing usage in `settings/viewmodel.go` (`computeNextRun`)

### Tertiary (LOW confidence)

- HTMX OOB swap for counter update — standard HTMX 2.0 feature, not yet used in this codebase but well-documented behavior. LOW because no existing example in project to verify exact templ rendering pattern.

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependencies; all patterns from existing codebase
- Architecture: HIGH — clear extension points in existing handlers and viewmodels
- Pitfalls: HIGH — most derived from direct code inspection (NULL tier, toggle direction, cron parsing)
- OOB swap pattern: MEDIUM — HTMX feature is known, templ integration not yet used in project

**Research date:** 2026-02-23
**Valid until:** 2026-03-25 (stable stack; 30-day estimate)
