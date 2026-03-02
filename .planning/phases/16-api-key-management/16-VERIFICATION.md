---
phase: 16-api-key-management
verified: 2026-03-02T15:00:00Z
status: human_needed
score: 11/11 must-haves verified
re_verification: false
human_verification:
  - test: "Navigate to /settings/api-keys in browser"
    expected: "Glass-morphism page renders with LLM Provider Keys, Tavily Search Key, and LLM Preference sections"
    why_human: "Visual rendering and layout correctness cannot be verified programmatically"
  - test: "Add an OpenAI API key via the form (select OpenAI, enter 'sk-testkey12345678', click Save LLM Key)"
    expected: "Key appears in list as 'sk-...5678' masked value with Delete button. No plaintext in HTML source."
    why_human: "End-to-end encryption round-trip requires live database with encryption key configured"
  - test: "Add a Tavily key, then delete it using the Delete button with confirmation"
    expected: "Confirmation dialog appears, key disappears from list after confirmation. HTMX fragment swap — no full page reload."
    why_human: "HTMX fragment swap behavior requires a running browser session"
  - test: "Select a preferred LLM provider (e.g., Anthropic) and model, click Save Preference"
    expected: "Model dropdown filters to Anthropic models client-side (no round-trip). Preference persists after page refresh."
    why_human: "Client-side JS model filtering requires browser execution; persistence requires live DB"
  - test: "Verify sidebar under Settings shows 'API Keys' sub-link and settings hub at /settings shows API Keys tile"
    expected: "Sub-link visible in sidebar when on any settings page; tile visible in hub grid"
    why_human: "Navigation visibility requires browser rendering"
---

# Phase 16: API Key Management Verification Report

**Phase Goal:** Users can securely store and manage their LLM and search API keys
**Verified:** 2026-03-02T15:00:00Z
**Status:** human_needed (all automated checks passed)
**Re-verification:** No — initial verification

---

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | LLM API keys are encrypted at rest using AES-256-GCM via GORM BeforeSave/AfterFind hooks | VERIFIED | `internal/models/user_api_key.go` — BeforeSave calls `encryptor.Encrypt`, AfterFind calls `encryptor.Decrypt`, using package-level `encryptor` from `auth_identity.go` |
| 2 | Tavily search API keys are encrypted at rest using the same mechanism | VERIFIED | Same `UserAPIKey` model with `key_type="tavily"` — no separate model needed; encryption applies to all rows |
| 3 | A user can have one active key per provider (enforced by partial unique index) | VERIFIED | Migration 000010: `CREATE UNIQUE INDEX idx_user_api_keys_unique_active ON user_api_keys(user_id, key_type, provider) WHERE deleted_at IS NULL` |
| 4 | Preferred LLM provider and model are stored as columns on the users table | VERIFIED | Migration 000011 adds `llm_preferred_provider` and `llm_preferred_model`; `User` struct has `LLMPreferredProvider` and `LLMPreferredModel` fields |
| 5 | Service layer provides CRUD + mask + LLM preference operations | VERIFIED | `internal/apikeys/service.go` — SaveKey (upsert), DeleteKey (user-scoped), GetKeysForUser, GetKeyByID, SaveLLMPreference (validated), MaskAPIKey all implemented |
| 6 | User can navigate to API Keys settings page from sidebar and settings hub | VERIFIED | `settings.templ` — sidebar sub-link at `/settings/api-keys` inside `isSettingsActive` block; hub tile with key SVG, title "API Keys", description "Manage your LLM and search API keys" |
| 7 | User can add an LLM provider API key via a form (selecting provider from dropdown) | VERIFIED | `APIKeysSection` templ component — provider `<select>` from `vm.Providers`, password input, HTMX post to `/api/user/api-keys` with `key_type=llm` hidden field; `SaveKeyHandler` validates and calls `SaveKey` |
| 8 | User can add a Tavily search API key via a form | VERIFIED | `APIKeysSection` — Tavily form with hidden `key_type=tavily` and `provider=tavily`; same HTMX endpoint |
| 9 | User can see stored keys with masked values (e.g., sk-...xxxx) | VERIFIED | `BuildViewModel` calls `MaskAPIKey(key.EncryptedValue)` on every key; template renders `{ key.MaskedValue }` — plaintext never reaches HTML |
| 10 | User can delete a stored key with confirmation | VERIFIED | Delete button with `hx-confirm="Delete this API key?"`, posts to `/api/user/api-keys/{id}/delete`; `DeleteKeyHandler` calls `DeleteKey` with user-scoped ownership check |
| 11 | User can select preferred LLM provider and model from dropdowns | VERIFIED | `LLMPreferenceSection` — provider/model selects with pre-selection from `vm.PreferredProvider/PreferredModel`; client-side JS model filtering via embedded JSON; `SaveLLMPreferenceHandler` validates and calls `SaveLLMPreference` |

**Score:** 11/11 truths verified

---

### Required Artifacts

#### Plan 01 Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/models/user_api_key.go` | UserAPIKey model with BeforeSave/AfterFind encryption hooks | VERIFIED | 52 lines; `BeforeSave` and `AfterFind` present; uses shared `encryptor` variable; nil-guard for test compatibility |
| `internal/database/migrations/000010_create_user_api_keys.up.sql` | user_api_keys table with partial unique index | VERIFIED | Creates table with all required columns; `CREATE UNIQUE INDEX ... WHERE deleted_at IS NULL` present |
| `internal/database/migrations/000011_add_llm_preferences_to_users.up.sql` | llm_preferred_provider and llm_preferred_model columns | VERIFIED | `ALTER TABLE users ADD COLUMN IF NOT EXISTS llm_preferred_provider` and `llm_preferred_model` |
| `internal/apikeys/service.go` | CRUD functions for API key management | VERIFIED | 109 lines; SaveKey, DeleteKey, GetKeysForUser, GetKeyByID, SaveLLMPreference, MaskAPIKey all implemented with real logic |
| `internal/apikeys/providers.go` | Curated list of supported LLM providers and models | VERIFIED | SupportedLLMProviders with OpenAI (4 models), Anthropic (3 models), Groq (3 models); GetProviderByID, GetAllModelsForProvider helpers |

#### Plan 02 Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/apikeys/handlers.go` | Gin handlers for API keys page, save, delete, LLM preference | VERIFIED | 162 lines; PageHandler, SaveKeyHandler, DeleteKeyHandler, SaveLLMPreferenceHandler all implemented; proper error handling |
| `internal/apikeys/viewmodel.go` | View model builder with masked key values | VERIFIED | BuildViewModel fetches keys, separates by type, masks values via MaskAPIKey, maps provider names |
| `internal/apikeysvm/types.go` | Leaf view model package (avoids import cycle) | VERIFIED | APIKeysPageViewModel, APIKeyViewModel, LLMProviderVM; no internal imports — clean leaf package |
| `internal/templates/settings.templ` | APIKeysSettingsPage, APIKeysSection, LLMPreferenceSection, sidebar sub-link, hub tile | VERIFIED | All six components present; error alert fragments included |
| `cmd/server/main.go` | Route registration for /settings/api-keys and API endpoints | VERIFIED | Four routes registered in protected group: GET /settings/api-keys, POST /api/user/api-keys, POST /api/user/api-keys/:id/delete, POST /api/user/llm-preference |

---

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `internal/models/user_api_key.go` | `internal/models/auth_identity.go` | shared package-level `encryptor` variable | WIRED | `user_api_key.go` calls `encryptor.Encrypt` / `encryptor.Decrypt` — same variable declared in `auth_identity.go` as `var encryptor *crypto.TokenEncryptor` |
| `internal/apikeys/service.go` | `internal/models/user_api_key.go` | GORM Create/Save/Delete operations | WIRED | `db.Create`, `db.Save`, `db.Delete` in SaveKey/DeleteKey; `db.Where(...).Find` in GetKeysForUser |
| `internal/apikeys/handlers.go` | `internal/apikeys/service.go` | calls SaveKey, DeleteKey, GetKeysForUser, SaveLLMPreference | WIRED | `SaveKey(db, ...)` in SaveKeyHandler; `DeleteKey(db, ...)` in DeleteKeyHandler; `SaveLLMPreference(db, ...)` in SaveLLMPreferenceHandler; BuildViewModel calls GetKeysForUser |
| `internal/apikeys/viewmodel.go` | `internal/apikeys/service.go` | calls MaskAPIKey to mask values before template rendering | WIRED | `MaskAPIKey(key.EncryptedValue)` called for every key in BuildViewModel |
| `cmd/server/main.go` | `internal/apikeys/handlers.go` | route registration in protected group | WIRED | `apikeys.PageHandler(db)`, `apikeys.SaveKeyHandler(db)`, `apikeys.DeleteKeyHandler(db)`, `apikeys.SaveLLMPreferenceHandler(db)` all registered |
| `internal/templates/settings.templ` | `/settings/api-keys` | sidebar sub-link and settings hub tile href | WIRED | Sidebar: `<a href="/settings/api-keys" ...>`; Hub: `<a href="/settings/api-keys" class="glass-card settings-hub-tile">` |

---

### Requirements Coverage

| Requirement | Description | Status | Evidence |
|-------------|-------------|--------|----------|
| KEYS-01 | User can store an encrypted LLM provider API key | SATISFIED | UserAPIKey model with AES-256-GCM hooks; SaveKey upserts LLM keys; SaveKeyHandler validates provider |
| KEYS-02 | User can store an encrypted Tavily search API key | SATISFIED | Same UserAPIKey model with key_type="tavily"; Tavily form in APIKeysSection |
| KEYS-03 | User can view stored keys with masked display | SATISFIED | MaskAPIKey in BuildViewModel; template renders MaskedValue only — plaintext never in HTML |
| KEYS-04 | User can update or delete stored API keys | SATISFIED | SaveKey is upsert (update if exists); DeleteKeyHandler soft-deletes with ownership check |
| KEYS-05 | User can select preferred LLM provider and model | SATISFIED | LLMPreferenceSection with validated dropdowns; SaveLLMPreferenceHandler + SaveLLMPreference with provider/model validation |

**Note:** REQUIREMENTS.md still shows all KEYS-01 through KEYS-05 as `[ ]` (unchecked) and status "Pending". The phase completed the implementation but did not close the requirements in the tracking file. This is a documentation gap — the requirements are functionally satisfied in code.

---

### Anti-Patterns Found

No anti-patterns detected in `internal/apikeys/`, `internal/apikeysvm/`, or `internal/models/user_api_key.go`.

- No TODO/FIXME/HACK comments
- No placeholder return values
- No stub implementations
- `make build` passes cleanly (templ generate + go build)
- All four task commits verified: 31b6a44, d6f35c7, 9a9b60a, 4f8b335

---

### Human Verification Required

#### 1. API Keys Page Renders Correctly

**Test:** Log in, navigate to `/settings/api-keys`
**Expected:** Full glass-morphism page with three sections: "LLM Provider Keys", "Tavily Search Key", "LLM Preference"
**Why human:** Visual rendering and CSS design system correctness cannot be verified programmatically

#### 2. Add LLM Key — Encrypted Storage and Masked Display

**Test:** On the API Keys page, select "OpenAI" from the provider dropdown, enter `sk-testkey12345678`, click "Save LLM Key"
**Expected:** Key appears in list as `sk-...5678` (masked). Inspecting the HTML source should show only the masked value — never the plaintext. Row stored encrypted in the database.
**Why human:** AES-256-GCM encryption requires a live database with `ENCRYPTION_KEY` environment variable; end-to-end round-trip cannot be tested statically

#### 3. Delete Key with Confirmation

**Test:** Add a Tavily key (any value), then click "Delete" on it
**Expected:** Browser confirmation dialog "Delete this API key?" appears. On confirm, the key row disappears. HTMX swaps only the `#api-keys-section` div — not a full page reload.
**Why human:** HTMX fragment swap behavior, browser confirmation dialogs require live browser execution

#### 4. LLM Preference — Client-Side Model Filtering

**Test:** In "LLM Preference" section, change the Provider dropdown to "Anthropic"
**Expected:** Model dropdown immediately filters to show only Anthropic models (claude-opus-4-5, claude-sonnet-4-5, claude-haiku-3-5) without a page reload or HTMX request
**Why human:** Client-side JavaScript model filtering requires browser execution

#### 5. Preference Persistence After Reload

**Test:** Select provider "Groq", model "llama-3.3-70b-versatile", click "Save Preference". Reload the page.
**Expected:** Provider and model dropdowns are pre-selected to the saved values on reload
**Why human:** Database persistence and SSR pre-selection requires live environment

#### 6. Navigation — Sidebar and Hub

**Test:** Navigate to `/settings`, confirm "API Keys" tile is visible. Click it. Confirm sidebar shows "API Keys" sub-link highlighted.
**Expected:** Hub tile with key icon is present; sidebar sub-link is visible and highlighted on the API Keys page
**Why human:** Navigation visibility and CSS active state require browser rendering

---

### Deviations from Plan (Documented by Claude)

- **apikeysvm leaf package introduced**: Plan 02 specified `internal/apikeys/viewmodel.go` would hold view model types. Due to Go's circular import prohibition (`apikeys` imports `templates`, `templates` imports `apikeys`), a `internal/apikeysvm/types.go` leaf package was created. This is an improvement on the plan, following the existing `settingsvm` pattern. All PLAN artifacts still exist; `viewmodel.go` uses `apikeysvm` types correctly.

---

### Build Verification

```
make build: PASSED
templ generate: 46 updates, 113ms
go build: clean (no errors)
```

---

## Gaps Summary

No gaps found. All 11 observable truths are verified by code inspection. All artifacts exist, are substantive (not stubs), and are correctly wired. The build passes. Five human verification items remain — these require a live browser/database session to confirm end-to-end behavior, encryption round-trips, and visual rendering.

The one documentation gap (REQUIREMENTS.md not updated to mark KEYS-01 through KEYS-05 as complete) does not block the phase goal. It should be addressed in phase documentation cleanup.

---

_Verified: 2026-03-02T15:00:00Z_
_Verifier: Claude (gsd-verifier)_
