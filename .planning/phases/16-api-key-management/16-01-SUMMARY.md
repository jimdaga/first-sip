---
phase: 16-api-key-management
plan: 01
subsystem: database
tags: [gorm, postgres, aes-256-gcm, encryption, api-keys, migrations]

# Dependency graph
requires:
  - phase: 08-auth-identity
    provides: "encryptor package-level variable and AES-256-GCM pattern in models package"
provides:
  - "UserAPIKey GORM model with transparent AES-256-GCM encryption via BeforeSave/AfterFind hooks"
  - "SQL migrations: user_api_keys table (with partial unique index) and llm_preferred_provider/llm_preferred_model columns on users"
  - "apikeys.SaveKey, DeleteKey, GetKeysForUser, GetKeyByID CRUD operations"
  - "apikeys.SaveLLMPreference with provider/model validation"
  - "apikeys.MaskAPIKey for safe display"
  - "SupportedLLMProviders: OpenAI, Anthropic, Groq with curated model lists"
affects:
  - "16-02-api-key-management-ui"
  - "17-litellm-integration"
  - "18-ai-briefing-generation"

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "GORM BeforeSave/AfterFind hooks for transparent encryption (same pattern as AuthIdentity)"
    - "Partial unique index (WHERE deleted_at IS NULL) for soft-delete-compatible uniqueness"
    - "User-scoped CRUD (userID check in DeleteKey and GetKeyByID prevents cross-user access)"
    - "Upsert pattern via First + Create/Save in SaveKey"

key-files:
  created:
    - internal/models/user_api_key.go
    - internal/database/migrations/000010_create_user_api_keys.up.sql
    - internal/database/migrations/000010_create_user_api_keys.down.sql
    - internal/database/migrations/000011_add_llm_preferences_to_users.up.sql
    - internal/database/migrations/000011_add_llm_preferences_to_users.down.sql
    - internal/apikeys/providers.go
    - internal/apikeys/service.go
  modified:
    - internal/models/user.go

key-decisions:
  - "Shared encryptor variable from auth_identity.go — no new encryption setup needed in UserAPIKey"
  - "Partial unique index on (user_id, key_type, provider) WHERE deleted_at IS NULL — allows re-adding keys after soft delete"
  - "EncryptedValue field stores both the plaintext (pre-hook) and ciphertext (post-hook) in same struct field — GORM hook pattern from existing codebase"

patterns-established:
  - "Encryption pattern: new encrypted models extend the models package and reference the shared encryptor variable"
  - "Service layer pattern: apikeys package with domain-specific CRUD wrapping GORM operations"
  - "MaskAPIKey: first3...last4 display masking called at view-model build time (Plan 02)"

requirements-completed:
  - KEYS-01
  - KEYS-02
  - KEYS-04
  - KEYS-05

# Metrics
duration: 1min
completed: 2026-03-02
---

# Phase 16 Plan 01: API Key Data Layer Summary

**AES-256-GCM encrypted UserAPIKey model, SQL migrations for user_api_keys table and LLM preference columns, and apikeys service package with CRUD + MaskAPIKey + provider/model validation**

## Performance

- **Duration:** ~2 min
- **Started:** 2026-03-02T14:07:59Z
- **Completed:** 2026-03-02T14:09:31Z
- **Tasks:** 2
- **Files modified:** 8

## Accomplishments

- UserAPIKey GORM model using the shared `encryptor` from `auth_identity.go` — transparent AES-256-GCM encryption with nil-guard for test compatibility
- Two SQL migrations: `000010` creates `user_api_keys` table with partial unique index, `000011` adds `llm_preferred_provider` and `llm_preferred_model` to `users`
- `apikeys` service package with SaveKey (upsert), DeleteKey (user-scoped soft delete), GetKeysForUser, GetKeyByID, SaveLLMPreference (validated), MaskAPIKey
- SupportedLLMProviders: OpenAI (4 models), Anthropic (3 models), Groq (3 models)

## Task Commits

Each task was committed atomically:

1. **Task 1: UserAPIKey model, SQL migrations, LLM preference columns** - `31b6a44` (feat)
2. **Task 2: API key service layer and provider definitions** - `d6f35c7` (feat)

**Plan metadata:** (docs commit pending)

## Files Created/Modified

- `internal/models/user_api_key.go` — UserAPIKey struct with BeforeSave/AfterFind encryption hooks
- `internal/models/user.go` — Added LLMPreferredProvider, LLMPreferredModel fields and APIKeys association
- `internal/database/migrations/000010_create_user_api_keys.up.sql` — user_api_keys table with partial unique index
- `internal/database/migrations/000010_create_user_api_keys.down.sql` — DROP TABLE user_api_keys
- `internal/database/migrations/000011_add_llm_preferences_to_users.up.sql` — ADD COLUMN llm_preferred_provider, llm_preferred_model
- `internal/database/migrations/000011_add_llm_preferences_to_users.down.sql` — DROP COLUMN both preference columns
- `internal/apikeys/providers.go` — SupportedLLMProviders, GetProviderByID, GetAllModelsForProvider
- `internal/apikeys/service.go` — SaveKey, DeleteKey, GetKeysForUser, GetKeyByID, SaveLLMPreference, MaskAPIKey

## Decisions Made

- Used the shared `encryptor` variable from `auth_identity.go` — no new encryptor declaration needed, follows established codebase pattern
- Partial unique index `(user_id, key_type, provider) WHERE deleted_at IS NULL` allows users to re-add a key after deleting it (soft delete leaves row, unique constraint only applies to active rows)
- `EncryptedValue` field name reflects final storage state — GORM hook encrypts on write, decrypts on read, presenting plaintext to application code

## Deviations from Plan

None — plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None — no external service configuration required. Migrations will apply automatically on next `make dev` startup.

## Next Phase Readiness

- Data layer complete — Plan 02 (UI layer) can consume `apikeys.GetKeysForUser`, `apikeys.SaveKey`, `apikeys.MaskAPIKey`, and `SupportedLLMProviders`
- Migrations ready to apply; `make dev` will run 000010 and 000011 on next startup
- No blockers for Plan 02

---
*Phase: 16-api-key-management*
*Completed: 2026-03-02*
