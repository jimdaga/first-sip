# Phase 16: API Key Management - Research

**Researched:** 2026-02-27
**Domain:** Encrypted secret storage, GORM model hooks, settings UI patterns (Go/Gin/Templ/HTMX)
**Confidence:** HIGH — all findings are from direct codebase inspection of proven patterns already shipping in production

---

## Summary

Phase 16 adds per-user encrypted storage for LLM provider API keys (OpenAI, Anthropic, Groq, etc.) and Tavily search API keys, plus a UI to add, view (masked), update, delete, and configure the preferred LLM provider and model. This is a settings feature, not an infrastructure feature — it is fundamentally a CRUD page for sensitive strings plus a preferred-provider selection.

The project already has every building block needed: AES-256-GCM encryption (`internal/crypto/`), the GORM `BeforeSave`/`AfterFind` hook pattern for transparent encryption (`models/auth_identity.go`), the `golang-migrate` SQL migration pattern (`internal/database/migrations/`), the Templ + HTMX settings page pattern (`internal/templates/settings.templ`, `internal/settings/handlers.go`), and a single `ENCRYPTION_KEY` environment variable already wired through `Config` and initialized at startup. No new libraries are required.

The two plans are correctly scoped: 16-01 for the data layer (model, migration, GORM hooks, service functions), and 16-02 for the UI layer (new settings sub-page, HTMX handlers, sidebar link, settings hub tile). The highest-risk area is the provider/model selection UX (KEYS-05): a curated hardcoded list of providers and their available models is the right approach for v1.2 — do not over-engineer with a dynamic registry.

**Primary recommendation:** Follow the `AuthIdentity` encryption pattern exactly. The crypto package, encryptor initialization, GORM hooks, and masked display logic are all well-established in this codebase — replicate, don't invent.

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| KEYS-01 | User can store an encrypted LLM provider API key (OpenAI, Anthropic, Groq, etc.) | AES-256-GCM via `crypto.TokenEncryptor` with GORM `BeforeSave`/`AfterFind` hooks — same pattern as `AuthIdentity` |
| KEYS-02 | User can store an encrypted Tavily search API key | Same model and encryption pattern as KEYS-01; Tavily key is a separate row with `key_type = "tavily"` |
| KEYS-03 | User can view stored keys with masked display (sk-...xxxx) | Go string manipulation at read time; never expose plaintext in HTML — mask format: first 3 chars + "..." + last 4 chars |
| KEYS-04 | User can update or delete their stored API keys | HTMX form POST for update, HTMX DELETE or POST for soft-delete via GORM `DeletedAt` |
| KEYS-05 | User can select their preferred LLM provider and model | Separate `user_llm_preferences` table OR column on the `UserAPIKey` model; curated hardcoded provider/model list in Go |
</phase_requirements>

---

## Standard Stack

### Core (No New Libraries Needed)

| Component | Already in Project | Purpose |
|-----------|-------------------|---------|
| `internal/crypto` | `crypto.TokenEncryptor` (AES-256-GCM) | Encrypt/decrypt API key values at rest |
| `gorm.io/gorm` | v1.31.1 | ORM with `BeforeSave`/`AfterFind` lifecycle hooks |
| `github.com/golang-migrate/migrate/v4` | v4.19.1 | SQL migration files in `internal/database/migrations/` |
| `github.com/a-h/templ` | v0.3.977 | Type-safe HTML templates for new settings page |
| `github.com/gin-gonic/gin` | v1.11.0 | HTTP handlers for CRUD endpoints |
| HTMX 2.0 | CDN | Fragment-based UI updates (no full page reloads) |

### No New Dependencies

The entire feature can be implemented with the existing stack. Do not add external libraries for:
- Encryption (already have `crypto` package)
- Secret storage (DB is the store; key management vault is out of scope)
- API key validation (KEYS-06 deferred to future milestones)

---

## Architecture Patterns

### Recommended Project Structure

```
internal/
├── models/
│   └── user_api_key.go          # UserAPIKey model with BeforeSave/AfterFind hooks
├── apikeys/                     # New package (mirrors settings/ pattern)
│   ├── handlers.go              # Gin handlers (page, add, update, delete)
│   └── viewmodel.go             # View model types for the Templ templates
├── templates/
│   └── settings.templ           # ADD: API Keys settings sub-page + settings hub tile
├── database/
│   └── migrations/
│       ├── 000010_create_user_api_keys.up.sql
│       └── 000010_create_user_api_keys.down.sql
```

**Why a new `apikeys/` package:** The `settings/` package is already large (handlers + viewmodel) and focused on plugin settings. API key management is a parallel settings domain with its own data model. Following the existing separation pattern avoids adding more complexity to `settings/handlers.go`.

### Pattern 1: GORM Encryption Hooks (established in codebase)

The `AuthIdentity` model in `/Users/jim/git/jimdaga/first-sip/internal/models/auth_identity.go` defines the canonical pattern:

```go
// Source: internal/models/auth_identity.go — proven pattern, replicate directly

// Package-level encryptor — initialized once at startup via InitEncryption()
var encryptor *crypto.TokenEncryptor

// BeforeSave encrypts the value before writing to DB
func (k *UserAPIKey) BeforeSave(tx *gorm.DB) error {
    if encryptor == nil {
        return nil // allow tests without encryption
    }
    if k.EncryptedValue != "" {
        encrypted, err := encryptor.Encrypt(k.EncryptedValue)
        if err != nil {
            return err
        }
        k.EncryptedValue = encrypted
    }
    return nil
}

// AfterFind decrypts after loading from DB
func (k *UserAPIKey) AfterFind(tx *gorm.DB) error {
    if encryptor == nil {
        return nil
    }
    if k.EncryptedValue != "" {
        decrypted, err := encryptor.Decrypt(k.EncryptedValue)
        if err != nil {
            return err
        }
        k.EncryptedValue = decrypted
    }
    return nil
}
```

**Critical note:** `InitEncryption()` is called once in `main.go` and sets the package-level `encryptor`. The `UserAPIKey` model must follow the same `var encryptor` pattern OR reuse the one in the `models` package by sharing the init function. Simpler: add `UserAPIKey` to the `models` package alongside `AuthIdentity`, sharing the same package-level `encryptor` variable.

### Pattern 2: SQL Migration Files

```sql
-- Source: internal/database/migrations/000010_create_user_api_keys.up.sql

CREATE TABLE user_api_keys (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key_type VARCHAR(50) NOT NULL,        -- "llm" or "tavily"
    provider VARCHAR(50) NOT NULL DEFAULT '', -- "openai", "anthropic", "groq", "" for tavily
    encrypted_value TEXT NOT NULL,
    UNIQUE(user_id, key_type, provider)   -- one key per user per type/provider combo
);

CREATE INDEX idx_user_api_keys_deleted_at ON user_api_keys(deleted_at);
CREATE INDEX idx_user_api_keys_user_id ON user_api_keys(user_id);
```

The `UNIQUE(user_id, key_type, provider)` constraint enforces one key per provider per user without needing application-level upsert guards. For Tavily, `provider` is empty string or `"tavily"`.

**For preferred LLM selection (KEYS-05):** Two approaches:
1. Add `llm_preferred_provider` and `llm_preferred_model` columns to the `users` table via a new migration (simpler, fewer joins)
2. Store as a special `key_type = "llm_preference"` row in `user_api_keys` (avoids schema change to users table)

**Recommendation:** Add columns to `users` table. It is semantically correct (preferences are user attributes, not secrets), and the `user_api_keys` table is purpose-built for secrets. This also avoids awkward "key_type as preference" logic. Migration: `000011_add_llm_preferences_to_users.up.sql`.

```sql
-- 000011_add_llm_preferences_to_users.up.sql
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS llm_preferred_provider VARCHAR(50) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS llm_preferred_model VARCHAR(100) NOT NULL DEFAULT '';
```

### Pattern 3: Settings Page Handler Structure

```go
// Source: internal/settings/handlers.go — replicate this structure in apikeys/handlers.go

func APIKeysPageHandler(db *gorm.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        user, err := getAuthUser(c, db)
        if err != nil {
            c.Redirect(http.StatusFound, "/login")
            return
        }
        sidebarPlugins := dashboard.GetSidebarPlugins(db, user.ID)
        // Build view model from DB
        vm, err := buildAPIKeysViewModel(db, user)
        if err != nil {
            // Render empty state, don't 500
        }
        render(c, templates.APIKeysSettingsPage(vm, sidebarPlugins))
    }
}
```

**Note on `getAuthUser` and `render` helpers:** These are duplicated across packages in the codebase (dashboard, settings) to avoid import cycles. The `apikeys` package should duplicate them as well — this is the established pattern.

### Pattern 4: Masked Key Display

```go
// Masking function for display — never expose plaintext in HTML
func maskAPIKey(plaintext string) string {
    if len(plaintext) <= 7 {
        return "***"
    }
    return plaintext[:3] + "..." + plaintext[len(plaintext)-4:]
}
// "sk-abc123xyz789" → "sk-...z789"
// "sk-ant-abc123xyz789" → "sk-...z789"
```

This function is called in the view model builder, NEVER in the Templ template or HTTP handler.

### Pattern 5: Provider/Model Curated List

KEYS-05 requires provider and model selection. The right approach is a hardcoded Go slice/map of supported providers and their models. This is read-only configuration, not a database table.

```go
// In apikeys/viewmodel.go or a separate providers.go file

type LLMProvider struct {
    ID     string   // "openai", "anthropic", "groq"
    Name   string   // "OpenAI", "Anthropic", "Groq"
    Models []string // ["gpt-4o", "gpt-4o-mini", "gpt-3.5-turbo"]
}

var SupportedLLMProviders = []LLMProvider{
    {ID: "openai",    Name: "OpenAI",    Models: []string{"gpt-4o", "gpt-4o-mini", "gpt-4-turbo", "gpt-3.5-turbo"}},
    {ID: "anthropic", Name: "Anthropic", Models: []string{"claude-opus-4-5", "claude-sonnet-4-5", "claude-haiku-3-5"}},
    {ID: "groq",      Name: "Groq",      Models: []string{"llama-3.3-70b-versatile", "llama-3.1-8b-instant", "mixtral-8x7b-32768"}},
}
```

The UI renders a `<select>` for provider, and a dependent `<select>` for model (filtered by provider). With HTMX, selecting a provider can trigger `hx-get` to reload the model dropdown. Alternatively, embed all provider/model data in the initial page render and use JavaScript `onchange` to filter — the simpler approach for a personal tool.

**Recommendation:** Embed all data at page render (no HTMX chaining needed). When the provider select changes, JavaScript filters the model select. One round-trip less, simpler handler.

### Anti-Patterns to Avoid

- **Storing plaintext API keys**: Always encrypt before DB write. The `BeforeSave` hook prevents this if implemented correctly.
- **Exposing decrypted keys in HTML**: Mask at view model build time. Templ templates receive only masked strings.
- **Putting encryption in handlers**: Keep it in the model's GORM hooks exactly like `AuthIdentity`.
- **Dynamic provider registry**: Don't build a database-backed provider list. Hardcode for v1.2, iterate in v1.3.
- **Using `api/settings/:pluginID/` routes for API keys**: API keys are a separate domain. Use distinct routes: `GET /settings/api-keys`, `POST /api/user/api-keys`, `DELETE /api/user/api-keys/:id`.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| AES-256-GCM encryption | Custom encryption | `internal/crypto.TokenEncryptor` | Already tested, handles nonce generation correctly |
| Transparent encrypt/decrypt | Manual encrypt calls in handlers | GORM `BeforeSave`/`AfterFind` hooks | `AuthIdentity` proves the pattern; hooks fire automatically |
| SQL migrations | GORM AutoMigrate | `golang-migrate` `.sql` files | Established project pattern; reversible; matches all 9 existing migrations |
| Masked key display | Regex or complex logic | Simple Go string slicing | `plaintext[:3] + "..." + plaintext[len-4:]` is sufficient |

**Key insight:** The project already solved encrypted secret storage for OAuth tokens. The only new thing is applying that solution to user-supplied API keys and building the CRUD UI for it.

---

## Common Pitfalls

### Pitfall 1: Double-Encryption on Update
**What goes wrong:** If a model is loaded (triggering `AfterFind` decrypt), then saved (triggering `BeforeSave` encrypt), the value is correctly round-tripped. But if `BeforeSave` runs on an already-encrypted value (e.g., from a direct DB update), the value is double-encrypted.
**Why it happens:** GORM `BeforeSave` fires on both `Create` and `Save`. If the plaintext field is populated from the form (not from DB), single encryption is correct. If the handler accidentally passes the already-encrypted DB value through a model update, it double-encrypts.
**How to avoid:** The handler for update receives the new key value from the form POST. Always overwrite the model's `EncryptedValue` field with the form value before saving — never pass through the DB-loaded value when the user provided a new one.
**Warning signs:** Decryption failures on subsequent loads; increasing ciphertext length on each update.

### Pitfall 2: UNIQUE Constraint on Soft Deletes
**What goes wrong:** The `UNIQUE(user_id, key_type, provider)` constraint fires on soft-deleted rows (those with `deleted_at IS NOT NULL`), preventing a user from re-adding a key for a provider they previously deleted.
**Why it happens:** Standard PostgreSQL UNIQUE constraints apply to all rows, including soft-deleted ones.
**How to avoid:** Use a partial unique index: `CREATE UNIQUE INDEX ... WHERE deleted_at IS NULL`. This matches the pattern used for `users.email` in migration 000001.
**Example:**
```sql
CREATE UNIQUE INDEX idx_user_api_keys_unique_active
    ON user_api_keys(user_id, key_type, provider)
    WHERE deleted_at IS NULL;
```

### Pitfall 3: Model Package Encryptor Sharing
**What goes wrong:** `models/auth_identity.go` uses a package-level `var encryptor *crypto.TokenEncryptor`. Adding `UserAPIKey` to the `models` package means it shares this same variable — which is correct. But if `UserAPIKey` is placed in a different package (e.g., `apikeys`), it needs its own `InitEncryption()` call in `main.go`.
**How to avoid:** Put `UserAPIKey` in the `models` package, alongside `AuthIdentity`. This way it reuses the existing `encryptor` and `InitEncryption()` without any main.go changes.

### Pitfall 4: Settings Hub Missing New Tile
**What goes wrong:** The API Keys settings page is reachable by URL but not discoverable from the `/settings` hub.
**How to avoid:** Add a tile to `SettingsHubPage` in `settings.templ` and add a sub-link to `AppSidebar` under the settings section — exactly as "Plugin Settings" and "Account" are listed.

### Pitfall 5: Templ File Needs Regeneration After Edit
**What goes wrong:** Editing `.templ` files without running `make templ-generate` causes compile errors (the `_templ.go` file is stale).
**How to avoid:** Always run `make templ-generate` after any `.templ` change. This is documented in CLAUDE.md.

---

## Code Examples

### UserAPIKey Model

```go
// Source: pattern from internal/models/auth_identity.go

package models

// UserAPIKey stores an encrypted API key for a specific provider.
// EncryptedValue is transparently encrypted/decrypted via GORM hooks.
type UserAPIKey struct {
    gorm.Model
    UserID         uint   `gorm:"not null;index"`
    User           User   `gorm:"constraint:OnDelete:CASCADE;"`
    KeyType        string `gorm:"not null"` // "llm" or "tavily"
    Provider       string `gorm:"not null;default:''"` // "openai", "anthropic", "groq", "" for tavily
    EncryptedValue string `gorm:"type:text;not null"`  // stored encrypted, decrypted after load
}

// BeforeSave encrypts EncryptedValue before writing to DB.
func (k *UserAPIKey) BeforeSave(tx *gorm.DB) error {
    if encryptor == nil {
        return nil
    }
    if k.EncryptedValue != "" {
        encrypted, err := encryptor.Encrypt(k.EncryptedValue)
        if err != nil {
            return err
        }
        k.EncryptedValue = encrypted
    }
    return nil
}

// AfterFind decrypts EncryptedValue after loading from DB.
func (k *UserAPIKey) AfterFind(tx *gorm.DB) error {
    if encryptor == nil {
        return nil
    }
    if k.EncryptedValue != "" {
        decrypted, err := encryptor.Decrypt(k.EncryptedValue)
        if err != nil {
            return err
        }
        k.EncryptedValue = decrypted
    }
    return nil
}
```

### Masked Key View Model Builder

```go
// In apikeys/viewmodel.go

type APIKeyViewModel struct {
    ID           uint
    KeyType      string // "llm" or "tavily"
    Provider     string // "openai", "anthropic", "groq", ""
    ProviderName string // "OpenAI", "Anthropic", "Groq", "Tavily"
    MaskedValue  string // "sk-...z789"
    IsSet        bool
}

type APIKeysPageViewModel struct {
    LLMKeys          []APIKeyViewModel
    TavilyKey        *APIKeyViewModel // nil if not set
    Providers        []LLMProvider     // for provider dropdown
    PreferredProvider string
    PreferredModel   string
    SidebarPlugins   []templates.SidebarPlugin
}

func maskAPIKey(plaintext string) string {
    if len(plaintext) <= 7 {
        return "***"
    }
    return plaintext[:3] + "..." + plaintext[len(plaintext)-4:]
}
```

### Route Registration

```go
// In cmd/server/main.go — append to protected group

protected.GET("/settings/api-keys", apikeys.PageHandler(db))
protected.POST("/api/user/api-keys", apikeys.SaveKeyHandler(db))
protected.DELETE("/api/user/api-keys/:id", apikeys.DeleteKeyHandler(db))
protected.POST("/api/user/api-keys/:id/delete", apikeys.DeleteKeyHandler(db)) // HTMX fallback (no DELETE from forms)
protected.POST("/api/user/llm-preference", apikeys.SaveLLMPreferenceHandler(db))
```

**Note:** HTMX 2.0 does not send DELETE directly from form elements. Use `POST /api/user/api-keys/:id/delete` as the delete endpoint to stay consistent with HTMX patterns in this codebase (which uses `hx-post` everywhere).

### HTMX Delete Pattern

```templ
<!-- In API Keys page template -->
<button
    class="glass-btn glass-btn-ghost glass-btn-sm"
    hx-post={ fmt.Sprintf("/api/user/api-keys/%d/delete", key.ID) }
    hx-target="#api-keys-section"
    hx-swap="outerHTML"
    hx-confirm="Delete this API key?"
>
    Delete
</button>
```

---

## State of the Art

| Old Approach | Current Approach | Notes |
|--------------|-----------------|-------|
| N/A — this is new feature | Use existing `crypto.TokenEncryptor` | Proven by `AuthIdentity` for 6+ weeks |
| GORM AutoMigrate | SQL migration files via `golang-migrate` | Established project standard |

---

## Open Questions

1. **Where to store preferred provider/model**
   - What we know: Must be persisted per user (KEYS-05). Two options: `users` table columns or `user_api_keys` table with special `key_type`.
   - What's unclear: Whether adding columns to `users` is preferred over a new row type.
   - Recommendation: Add `llm_preferred_provider` and `llm_preferred_model` to `users` table. Semantically clean; avoids mixing preferences with secrets. Requires migration 000011.

2. **Multi-provider key storage structure**
   - What we know: User may have keys for multiple LLM providers (OpenAI + Anthropic simultaneously). KEYS-05 selects which one to USE.
   - What's unclear: Do we support multiple concurrent LLM provider keys?
   - Recommendation: Yes — allow one key per provider (UNIQUE by user+type+provider). User sets a preferred provider. Phase 17 reads the key for the preferred provider. This matches the `UNIQUE(user_id, key_type, provider)` table design above.

3. **Delete behavior — soft vs hard**
   - What we know: All existing models use GORM soft delete (`DeletedAt`). The partial UNIQUE index (Pitfall 2) handles re-adding a key after soft delete.
   - Recommendation: Soft delete for consistency. The `deleted_at IS NULL` partial index handles re-add.

---

## Sources

### Primary (HIGH confidence)

- `/Users/jim/git/jimdaga/first-sip/internal/models/auth_identity.go` — GORM hook encryption pattern
- `/Users/jim/git/jimdaga/first-sip/internal/crypto/crypto.go` — AES-256-GCM TokenEncryptor
- `/Users/jim/git/jimdaga/first-sip/internal/database/migrations/` — SQL migration file patterns (migrations 000001–000009)
- `/Users/jim/git/jimdaga/first-sip/internal/settings/handlers.go` — Gin handler patterns, getAuthUser, render helpers
- `/Users/jim/git/jimdaga/first-sip/internal/templates/settings.templ` — Templ/HTMX settings UI patterns
- `/Users/jim/git/jimdaga/first-sip/internal/settingsvm/settingsvm.go` — View model separation pattern
- `/Users/jim/git/jimdaga/first-sip/internal/config/config.go` — EncryptionKey already in Config
- `/Users/jim/git/jimdaga/first-sip/cmd/server/main.go` — Route registration pattern, protected group

### Secondary (MEDIUM confidence)

- HTMX 2.0 DELETE method: HTMX supports `hx-delete` but HTML forms do not natively submit DELETE. In this codebase, all mutations use `hx-post`. Consistent to use `POST /api/user/api-keys/:id/delete` pattern.

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new libraries; all patterns exist in the codebase
- Architecture: HIGH — direct inspection of auth_identity.go, settings handlers, migration pattern
- Pitfalls: HIGH — soft delete UNIQUE pitfall is common and specifically visible in migration 000001 (partial index on email)
- Provider/model list: MEDIUM — model names are from training knowledge; planner should verify current model IDs at implementation time

**Research date:** 2026-02-27
**Valid until:** 2026-04-01 (stable Go stack; provider model lists may drift but structure is fixed)
