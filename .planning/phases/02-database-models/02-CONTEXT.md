# Phase 2: Database & Models - Context

**Gathered:** 2026-02-11
**Status:** Ready for planning

<domain>
## Phase Boundary

Persistent storage for users and briefings using Postgres and GORM. Includes User model with multi-provider auth support, Briefing model with JSON content storage, encrypted OAuth tokens, versioned migrations via golang-migrate, and Docker Compose for local development. This phase replaces session-only user storage from Phase 1 with database-backed persistence.

</domain>

<decisions>
## Implementation Decisions

### User model shape
- Include preference fields: timezone, preferred briefing time (with smart defaults — timezone from browser, briefing time defaults to 6 AM)
- Track activity: last_login_at, last_briefing_at timestamps on User model
- Simple role enum: string field with 'user' or 'admin' values
- Auto-create user record on first Google login (no allowlist/invite)
- Soft delete via GORM deleted_at — never hard delete user records
- All timestamps stored in UTC, converted to user timezone in UI layer

### Multi-provider auth
- Separate auth_identities table linked to User (not provider fields on User)
- Store full OAuth tokens (access_token, refresh_token, expiry) in auth_identities
- Provider + provider_user_id for identification
- Designed for multi-provider from day one, only Google implemented now

### Briefing model shape
- Single JSON 'content' field for all sections (News, Weather, Work) — not normalized
- Status lifecycle: Pending → Processing → Completed / Failed (four states)
- Minimal metadata: error_message string for failures, no generation metadata JSON
- read_at timestamp (NULL = unread, timestamp = when read)
- Simple foreign key: user_id on Briefing table, one user owns many briefings

### Dev/prod database strategy
- **Postgres everywhere** — no SQLite. Dev and prod both use Postgres containers
- Docker Compose for local dev Postgres (one command to start)
- golang-migrate for versioned SQL migration files (not GORM AutoMigrate)
- Include seed data for local development (test user, sample briefing)

### Encryption & security
- AES-256-GCM encryption for OAuth tokens (access_token, refresh_token)
- Encryption via GORM hooks (BeforeSave/AfterFind)
- Tokens only — email and name stay plaintext for querying
- Encryption key: env variable for local dev (`source env.local`), external-secrets-operator in Kubernetes
- DB SSL handled at infrastructure level (K8s network), not app-level sslmode

### Claude's Discretion
- GORM model struct organization and field tags
- Migration file naming and ordering conventions
- Docker Compose configuration details (ports, volumes, health checks)
- Seed data content and structure
- Connection pooling settings

</decisions>

<specifics>
## Specific Ideas

- "Postgres everywhere" — user explicitly wants dev/prod parity, no SQLite abstraction layer
- Kubernetes secrets via external-secrets-operator in production — the app just reads env vars
- `source env.local` pattern for local development secrets

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 02-database-models*
*Context gathered: 2026-02-11*
