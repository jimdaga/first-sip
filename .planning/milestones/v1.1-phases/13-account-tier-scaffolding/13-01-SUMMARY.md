---
phase: 13-account-tier-scaffolding
plan: 01
subsystem: database
tags: [account-tiers, gorm, postgres, migration, cron, plugins]

# Dependency graph
requires:
  - phase: 12-dynamic-settings-ui
    provides: UserPluginConfig model with enabled flag used for plugin count queries
  - phase: 10-scheduling
    provides: robfig/cron/v3 already in go.mod for cron expression parsing

provides:
  - AccountTier GORM model (free: 3 plugins/24h, pro: unlimited/2h)
  - Migration 000008 — account_tiers table and users.account_tier_id FK
  - SeedAccountTiers function (idempotent, all envs)
  - TierService with GetUserTier (NULL fallback), CanEnablePlugin, CanUseFrequency, GetEnabledCount
  - Free tier assigned to new users on Google OAuth registration

affects:
  - 13-02 (plan 02 — tier enforcement in handlers and UI depends on this foundation)
  - settings handlers (TogglePluginHandler must call CanEnablePlugin before enabling)
  - auth handlers (free tier assignment already wired)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Tier NULL fallback: users with no AccountTierID are treated as free tier by GetUserTier"
    - "FirstOrCreate with name lookup for idempotent tier seeding"
    - "cron interval check: two consecutive Next() calls to measure schedule frequency"

key-files:
  created:
    - internal/models/account_tier.go
    - internal/database/migrations/000008_add_account_tiers.up.sql
    - internal/database/migrations/000008_add_account_tiers.down.sql
    - internal/tiers/service.go
  modified:
    - internal/models/user.go
    - internal/database/seed.go
    - internal/auth/handlers.go
    - cmd/server/main.go

key-decisions:
  - "AccountTierID as *uint (nullable pointer) — NULL means pre-migration user, gracefully falls back to free tier in TierService"
  - "MaxEnabledPlugins = -1 sentinel for unlimited (pro tier) avoids extra boolean column"
  - "SeedAccountTiers called in ALL environments (not dev-only) — tiers are production data, not dev fixtures"
  - "TierService is a standalone package (not embedded in auth/plugins) — single responsibility, importable by any handler"
  - "CanUseFrequency uses two Next() calls to compute actual schedule interval — handles irregular cron expressions correctly"

patterns-established:
  - "Tier constraint check pattern: call TierService method, receive (bool, *AccountTier, error), use tier for error messaging"
  - "Free tier fallback: GetUserTier returns free tier for NULL AccountTierID (no migration needed for existing users)"

requirements-completed: [TIER-01, TIER-02, TIER-03]

# Metrics
duration: 2min
completed: 2026-02-23
---

# Phase 13 Plan 01: Account Tier Scaffolding Summary

**AccountTier GORM model + migration 000008 + TierService (plugin count and cron interval checks) + free tier assigned on OAuth registration**

## Performance

- **Duration:** ~2 min
- **Started:** 2026-02-24T02:37:27Z
- **Completed:** 2026-02-24T02:39:16Z
- **Tasks:** 2
- **Files modified:** 8

## Accomplishments
- AccountTier model with free (3 plugins/24h) and pro (unlimited/2h) definitions
- Migration 000008 creates account_tiers table and adds nullable account_tier_id FK to users
- SeedAccountTiers function seeds both tiers on every startup (idempotent via FirstOrCreate)
- TierService with GetUserTier (NULL fallback to free), CanEnablePlugin, CanUseFrequency, GetEnabledCount
- New user registration via Google OAuth assigns free tier ID

## Task Commits

Each task was committed atomically:

1. **Task 1: AccountTier model, migration, and seed function** - `cc8a169` (feat)
2. **Task 2: TierService and free tier on registration** - `e316487` (feat)

**Plan metadata:** (docs commit follows)

## Files Created/Modified
- `internal/models/account_tier.go` — AccountTier GORM struct (Name, MaxEnabledPlugins, MinFrequencyHours)
- `internal/models/user.go` — Added AccountTierID *uint and AccountTier association
- `internal/database/migrations/000008_add_account_tiers.up.sql` — CREATE TABLE account_tiers + ALTER TABLE users
- `internal/database/migrations/000008_add_account_tiers.down.sql` — Rollback: DROP FK + DROP TABLE
- `internal/database/seed.go` — SeedAccountTiers function (FirstOrCreate pattern, all envs)
- `internal/tiers/service.go` — TierService with all four methods
- `internal/auth/handlers.go` — Free tier lookup and assignment on new user creation
- `cmd/server/main.go` — SeedAccountTiers wired after RunMigrations, before SeedDevData

## Decisions Made
- AccountTierID as `*uint` (nullable pointer) — NULL means pre-migration user; TierService.GetUserTier falls back to free tier automatically, no data migration required for existing users
- MaxEnabledPlugins = -1 sentinel for unlimited (pro tier) — avoids extra boolean column, consistent with common API design
- SeedAccountTiers called in ALL environments — tiers are production data (not dev fixtures), needed before any user can register
- TierService is a standalone `internal/tiers` package — clean separation, importable by settings/dashboard handlers in Plan 02 without circular imports
- CanUseFrequency computes interval via two consecutive `schedule.Next()` calls — correctly handles non-uniform cron expressions (e.g., `0 0 * * 1` is weekly, not daily)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Plan 02 can immediately import `internal/tiers` and call CanEnablePlugin in TogglePluginHandler
- TierService.GetUserTier NULL fallback means existing users with NULL AccountTierID work correctly today
- Migration 000008 applies cleanly — verified via `go build ./...` passing

---
*Phase: 13-account-tier-scaffolding*
*Completed: 2026-02-23*

## Self-Check: PASSED

- internal/models/account_tier.go: FOUND
- internal/tiers/service.go: FOUND
- internal/database/migrations/000008_add_account_tiers.up.sql: FOUND
- internal/database/migrations/000008_add_account_tiers.down.sql: FOUND
- Commit cc8a169: FOUND
- Commit e316487: FOUND
