# Phase 02: Database & Models - Research

**Researched:** 2026-02-11
**Domain:** PostgreSQL with GORM ORM and golang-migrate
**Confidence:** HIGH

## Summary

Phase 02 introduces persistent storage using PostgreSQL with GORM v2, replacing the session-only user storage from Phase 1. The research confirms that the user's locked decisions (GORM, Postgres everywhere, golang-migrate, AES-256-GCM encryption) align with current Go ecosystem best practices for production applications.

GORM v2 (current latest ~v1.30.0+) provides robust ORM capabilities including hooks for encryption (BeforeSave/AfterFind), soft deletes via DeletedAt, and comprehensive Postgres support through the pgx driver. golang-migrate v4 is the standard tool for versioned SQL migrations with strong community adoption (24,100+ dependent projects) and production-proven reliability.

The "Postgres everywhere" approach simplifies development by ensuring dev/prod parity. Docker Compose handles local Postgres instances, while golang-migrate's embedded migration support enables single-binary deployments. The encryption approach using GORM hooks with Go's crypto/aes GCM is well-established and provides AEAD (Authenticated Encryption with Associated Data) guarantees.

**Primary recommendation:** Use GORM v2 with Postgres driver, golang-migrate v4 for versioned SQL migrations (not GORM AutoMigrate), implement encryption via GORM hooks with AES-256-GCM from Go's standard library, configure connection pooling explicitly, and embed migrations in binary for production deployments.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### User model shape
- Include preference fields: timezone, preferred briefing time (with smart defaults — timezone from browser, briefing time defaults to 6 AM)
- Track activity: last_login_at, last_briefing_at timestamps on User model
- Simple role enum: string field with 'user' or 'admin' values
- Auto-create user record on first Google login (no allowlist/invite)
- Soft delete via GORM deleted_at — never hard delete user records
- All timestamps stored in UTC, converted to user timezone in UI layer

#### Multi-provider auth
- Separate auth_identities table linked to User (not provider fields on User)
- Store full OAuth tokens (access_token, refresh_token, expiry) in auth_identities
- Provider + provider_user_id for identification
- Designed for multi-provider from day one, only Google implemented now

#### Briefing model shape
- Single JSON 'content' field for all sections (News, Weather, Work) — not normalized
- Status lifecycle: Pending → Processing → Completed / Failed (four states)
- Minimal metadata: error_message string for failures, no generation metadata JSON
- read_at timestamp (NULL = unread, timestamp = when read)
- Simple foreign key: user_id on Briefing table, one user owns many briefings

#### Dev/prod database strategy
- **Postgres everywhere** — no SQLite. Dev and prod both use Postgres containers
- Docker Compose for local dev Postgres (one command to start)
- golang-migrate for versioned SQL migration files (not GORM AutoMigrate)
- Include seed data for local development (test user, sample briefing)

#### Encryption & security
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

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope

</user_constraints>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| gorm.io/gorm | v1.30.0+ | ORM for database operations | Most popular Go ORM (39,452 GitHub stars), developer-friendly API, robust hooks system, production-proven |
| gorm.io/driver/postgres | v1.6.0+ | PostgreSQL driver for GORM | Official GORM Postgres driver using pgx, prepared statement caching, 15,874+ projects |
| github.com/golang-migrate/migrate/v4 | v4.x | Versioned SQL migrations | Industry standard (18.1k stars, 24,100+ projects), explicit version control, rollback support |
| github.com/jackc/pgx/v5 | v5.x (via GORM driver) | PostgreSQL driver/toolkit | High-performance Postgres driver, used by GORM's Postgres driver, native protocol support |
| crypto/aes + crypto/cipher | stdlib | AES-256-GCM encryption | Go standard library, AEAD encryption, well-audited, zero dependencies |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| github.com/lib/pq | v1.10.x | Alternative Postgres driver | Not recommended for GORM (use pgx-based driver instead) |
| gorm.io/datatypes | v1.2.x | GORM data types (JSON, etc.) | For PostgreSQL jsonb field handling with type safety |
| gorm.io/plugin/soft_delete | v1.2.x | Alternative soft delete formats | Only if unix timestamp/flag format needed (not needed with gorm.Model) |

### Installation
```bash
# Core dependencies
go get -u gorm.io/gorm
go get -u gorm.io/driver/postgres
go get -u github.com/golang-migrate/migrate/v4
go get -u github.com/golang-migrate/migrate/v4/database/postgres
go get -u github.com/golang-migrate/migrate/v4/source/file

# Optional: For jsonb type handling
go get -u gorm.io/datatypes
```

**CLI tool for migrations:**
```bash
# macOS
brew install golang-migrate

# Docker (alternative)
docker pull migrate/migrate
```

## Architecture Patterns

### Recommended Project Structure
```
.
├── cmd/
│   └── server/
│       └── main.go              # App entry point, DB initialization
├── internal/
│   ├── config/
│   │   └── config.go            # Config including DB settings
│   ├── database/
│   │   ├── db.go                # Database connection setup
│   │   ├── migrations.go        # Embedded migrations runner
│   │   └── seed.go              # Seed data for local dev
│   └── models/
│       ├── user.go              # User model
│       ├── auth_identity.go     # AuthIdentity model with encryption
│       └── briefing.go          # Briefing model
├── migrations/
│   ├── 000001_create_users.up.sql
│   ├── 000001_create_users.down.sql
│   ├── 000002_create_auth_identities.up.sql
│   ├── 000002_create_auth_identities.down.sql
│   ├── 000003_create_briefings.up.sql
│   ├── 000003_create_briefings.down.sql
│   └── 000004_seed_dev_data.up.sql     # Seed data migration
└── docker-compose.yml           # Local Postgres service
```

### Pattern 1: Database Connection with Connection Pooling

**What:** Initialize GORM with Postgres and configure connection pool settings explicitly.

**When to use:** Application startup, before starting HTTP server.

**Example:**
```go
// Source: https://gorm.io/docs/connecting_to_the_database.html
import (
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
    "time"
)

func InitDatabase(dsn string) (*gorm.DB, error) {
    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    if err != nil {
        return nil, err
    }

    // Get underlying *sql.DB to configure connection pool
    sqlDB, err := db.DB()
    if err != nil {
        return nil, err
    }

    // Connection pool settings (recommended for production)
    sqlDB.SetMaxOpenConns(20)                  // Max open connections
    sqlDB.SetMaxIdleConns(10)                  // Idle connection pool size
    sqlDB.SetConnMaxLifetime(5 * time.Minute)  // Connection reuse duration

    return db, nil
}
```

**DSN Format:**
```
host=localhost user=myuser password=mypass dbname=mydb port=5432 sslmode=disable TimeZone=UTC
```

### Pattern 2: GORM Models with Soft Delete

**What:** Define models embedding gorm.Model for automatic ID, timestamps, and soft delete support.

**When to use:** All persistent models that should never be hard-deleted.

**Example:**
```go
// Source: https://gorm.io/docs/models.html
import "gorm.io/gorm"

type User struct {
    gorm.Model              // Includes: ID, CreatedAt, UpdatedAt, DeletedAt
    Email              string `gorm:"uniqueIndex;not null"`
    Name               string
    Timezone           string `gorm:"default:'UTC'"`
    PreferredBriefingTime string `gorm:"default:'06:00'"`
    Role               string `gorm:"default:'user'"`  // 'user' or 'admin'
    LastLoginAt        *time.Time
    LastBriefingAt     *time.Time
}

// Soft delete behavior
db.Delete(&user)              // Sets DeletedAt to current time
db.Find(&users)               // Excludes soft-deleted records
db.Unscoped().Find(&users)    // Includes soft-deleted records
```

### Pattern 3: Foreign Key Relationships (Belongs To)

**What:** Define belongs-to relationships with explicit foreign keys.

**When to use:** Models that reference another model (e.g., AuthIdentity belongs to User).

**Example:**
```go
// Source: https://gorm.io/docs/belongs_to.html
type AuthIdentity struct {
    gorm.Model
    UserID         uint   `gorm:"not null;index"`
    User           User   `gorm:"constraint:OnDelete:CASCADE;"`
    Provider       string `gorm:"not null"`              // 'google', 'github', etc.
    ProviderUserID string `gorm:"not null"`
    AccessToken    string `gorm:"type:text"`             // Encrypted
    RefreshToken   string `gorm:"type:text"`             // Encrypted
    TokenExpiry    *time.Time
}

// GORM will automatically create foreign key: auth_identities.user_id -> users.id
// With CASCADE delete constraint
```

### Pattern 4: JSON Fields in Postgres

**What:** Store JSON/JSONB data in Postgres columns.

**When to use:** Semi-structured data that doesn't require normalization (e.g., briefing content).

**Example:**
```go
// Source: https://gorm.io/docs/data_types.html
import "gorm.io/datatypes"

type Briefing struct {
    gorm.Model
    UserID     uint              `gorm:"not null;index"`
    User       User              `gorm:"constraint:OnDelete:CASCADE;"`
    Content    datatypes.JSON    `gorm:"type:jsonb"`
    Status     string            `gorm:"not null;default:'pending'"` // pending, processing, completed, failed
    ErrorMsg   string            `gorm:"type:text"`
    ReadAt     *time.Time
}

// Usage
briefing := Briefing{
    Content: datatypes.JSON([]byte(`{"news": [...], "weather": {...}, "work": [...]}`)),
}
db.Create(&briefing)
```

### Pattern 5: Encryption via GORM Hooks

**What:** Implement BeforeSave/AfterFind hooks to encrypt/decrypt sensitive fields.

**When to use:** Fields containing secrets, tokens, or PII requiring encryption at rest.

**Example:**
```go
// Source: https://gorm.io/docs/hooks.html
import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "encoding/base64"
    "io"
    "gorm.io/gorm"
)

type AuthIdentity struct {
    gorm.Model
    UserID         uint
    AccessToken    string `gorm:"type:text"`  // Stored encrypted
    RefreshToken   string `gorm:"type:text"`  // Stored encrypted
    // ... other fields
}

// BeforeSave encrypts tokens before writing to DB
func (a *AuthIdentity) BeforeSave(tx *gorm.DB) error {
    if a.AccessToken != "" {
        encrypted, err := encrypt(a.AccessToken)
        if err != nil {
            return err
        }
        a.AccessToken = encrypted
    }
    if a.RefreshToken != "" {
        encrypted, err := encrypt(a.RefreshToken)
        if err != nil {
            return err
        }
        a.RefreshToken = encrypted
    }
    return nil
}

// AfterFind decrypts tokens after reading from DB
func (a *AuthIdentity) AfterFind(tx *gorm.DB) error {
    if a.AccessToken != "" {
        decrypted, err := decrypt(a.AccessToken)
        if err != nil {
            return err
        }
        a.AccessToken = decrypted
    }
    if a.RefreshToken != "" {
        decrypted, err := decrypt(a.RefreshToken)
        if err != nil {
            return err
        }
        a.RefreshToken = decrypted
    }
    return nil
}

// encrypt using AES-256-GCM
func encrypt(plaintext string) (string, error) {
    key := getEncryptionKey() // 32 bytes for AES-256

    block, err := aes.NewCipher(key)
    if err != nil {
        return "", err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }

    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return "", err
    }

    ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
    return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decrypt using AES-256-GCM
func decrypt(ciphertext string) (string, error) {
    key := getEncryptionKey()

    enc, err := base64.StdEncoding.DecodeString(ciphertext)
    if err != nil {
        return "", err
    }

    block, err := aes.NewCipher(key)
    if err != nil {
        return "", err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }

    nonceSize := gcm.NonceSize()
    if len(enc) < nonceSize {
        return "", errors.New("ciphertext too short")
    }

    nonce, ciphertext := enc[:nonceSize], enc[nonceSize:]
    plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return "", err
    }

    return string(plaintext), nil
}
```

### Pattern 6: golang-migrate with Embedded Migrations

**What:** Embed SQL migration files in the binary using Go 1.16+ embed directive.

**When to use:** Production deployments to avoid shipping SQL files separately.

**Example:**
```go
// Source: https://github.com/golang-migrate/migrate
import (
    "embed"
    "github.com/golang-migrate/migrate/v4"
    _ "github.com/golang-migrate/migrate/v4/database/postgres"
    "github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func RunMigrations(dbURL string) error {
    source, err := iofs.New(migrationsFS, "migrations")
    if err != nil {
        return err
    }

    m, err := migrate.NewWithSourceInstance("iofs", source, dbURL)
    if err != nil {
        return err
    }

    if err := m.Up(); err != nil && err != migrate.ErrNoChange {
        return err
    }

    return nil
}
```

### Pattern 7: Migration File Structure

**What:** Versioned SQL migrations with up/down pairs.

**When to use:** All schema changes and seed data.

**Example:**
```sql
-- migrations/000001_create_users.up.sql
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    email VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255),
    timezone VARCHAR(50) NOT NULL DEFAULT 'UTC',
    preferred_briefing_time VARCHAR(5) NOT NULL DEFAULT '06:00',
    role VARCHAR(20) NOT NULL DEFAULT 'user',
    last_login_at TIMESTAMPTZ,
    last_briefing_at TIMESTAMPTZ
);

CREATE INDEX idx_users_deleted_at ON users(deleted_at);
CREATE INDEX idx_users_email ON users(email) WHERE deleted_at IS NULL;

-- migrations/000001_create_users.down.sql
DROP TABLE IF EXISTS users;
```

**Naming convention:** `{version}_{description}.{up|down}.sql`
- Sequential versioning: `000001`, `000002`, `000003` (use `-seq` flag with migrate create)
- Timestamp versioning: `20260211120000` (default, prevents conflicts in team settings)

### Pattern 8: Docker Compose for Local Postgres

**What:** Local Postgres container with health checks and automatic startup.

**When to use:** Local development environment.

**Example:**
```yaml
# Source: https://docs.docker.com/reference/samples/postgres/
version: '3.8'

services:
  postgres:
    image: postgres:16-alpine
    container_name: first-sip-db
    environment:
      POSTGRES_DB: first_sip
      POSTGRES_USER: first_sip
      POSTGRES_PASSWORD: local_dev_password
      POSTGRES_INITDB_ARGS: "-E UTF8 --locale=en_US.utf8"
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U first_sip -d first_sip"]
      interval: 10s
      timeout: 5s
      retries: 5
    restart: unless-stopped

volumes:
  postgres_data:
```

**Usage:**
```bash
# Start Postgres
docker compose up -d

# Check health
docker compose ps

# Stop Postgres
docker compose down

# Reset database (destroys data)
docker compose down -v
```

### Anti-Patterns to Avoid

- **Don't use GORM AutoMigrate in production:** AutoMigrate doesn't provide version control, rollback capability, or drop unused columns. Use versioned migrations.
- **Don't close DB connection per request:** GORM manages a connection pool. Only close on application shutdown.
- **Don't use Save() for partial updates:** Save() updates all fields, potentially overwriting with zero values. Use Updates() with map or Select/Omit.
- **Don't forget Preload for associations:** Loading associations in a loop causes N+1 queries. Use Preload() or Joins().
- **Don't reuse nonces in GCM encryption:** Each encryption operation must use a unique nonce. Never encrypt more than 2^32 messages with the same key.
- **Don't store encryption keys in code:** Use environment variables for local dev, external-secrets-operator for Kubernetes.
- **Don't modify existing migrations:** Once a migration runs in any environment, treat it as immutable. Create new migrations for changes.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| SQL migrations | Custom migration tracking table | golang-migrate | Handles version tracking, dirty state detection, rollback logic, concurrent migration prevention |
| Database connection pooling | Custom pool management | database/sql + GORM config | Built-in pooling, tested across millions of deployments, handles edge cases |
| AES-GCM encryption | Custom crypto implementation | crypto/aes + crypto/cipher (stdlib) | Audited, constant-time operations (with AES-NI), correct nonce handling, AEAD guarantees |
| Soft delete logic | Custom deleted_at filtering | gorm.Model + gorm.DeletedAt | Automatic filtering in queries, Unscoped() for explicit access, integrates with associations |
| JSON/JSONB handling | Custom marshaling for Postgres | gorm.io/datatypes.JSON | Implements Scanner/Valuer, handles Postgres JSONB, type-safe |
| Timestamp tracking | Manual CreatedAt/UpdatedAt | gorm.Model | Automatic timestamp management, works with hooks, consistent across all operations |
| Foreign key constraints | Manual relationship tracking | GORM associations with constraint tags | Generates proper FK constraints, CASCADE/SET NULL support, validates on save |

**Key insight:** Database operations involve subtle edge cases (transaction isolation, connection lifecycle, character encoding, timezone handling, null value semantics). The Go ecosystem has mature, well-tested libraries for all database concerns. Custom implementations typically miss edge cases that appear under load or in production scenarios.

## Common Pitfalls

### Pitfall 1: N+1 Query Problem with Associations

**What goes wrong:** Loading a collection, then accessing associations in a loop generates one query per record instead of using joins or preloading.

**Why it happens:** GORM doesn't automatically load associations. Developers access relationships without Preload().

**How to avoid:**
```go
// BAD: N+1 queries (1 + N)
users := []User{}
db.Find(&users)
for _, user := range users {
    db.Model(&user).Association("AuthIdentities").Find(&user.AuthIdentities)  // N queries
}

// GOOD: Preload (2 queries total)
users := []User{}
db.Preload("AuthIdentities").Find(&users)
```

**Warning signs:** Slow queries in development with small datasets that timeout in production; database query count grows linearly with result set size.

### Pitfall 2: Migration "Dirty" State in Production

**What goes wrong:** Migration fails midway, database marked "dirty", application won't start, requires manual intervention with DB access.

**Why it happens:** Migration contains error (syntax, constraint violation), long-running operation times out, connection drops during migration.

**How to avoid:**
- Test migrations in dev environment first (both up and down)
- Keep migrations small and focused (one logical change per migration)
- Use transactions in migration files where possible
- Set reasonable timeouts for long operations
- For long operations (large table indexes), use `CREATE INDEX CONCURRENTLY` in Postgres (must be outside transaction)

**Warning signs:** Migration file contains multiple unrelated schema changes; migration creates index on large existing table without CONCURRENTLY; no down migration provided.

### Pitfall 3: Connection Pool Exhaustion

**What goes wrong:** Application opens database connections but never releases them, connection pool exhausts, new requests hang waiting for available connections.

**Why it happens:** Not configuring pool limits, leaking connections in error paths, holding transactions too long.

**How to avoid:**
```go
// Configure pool limits explicitly
sqlDB, _ := db.DB()
sqlDB.SetMaxOpenConns(20)       // Limit total connections
sqlDB.SetMaxIdleConns(10)       // Limit idle connections
sqlDB.SetConnMaxLifetime(5 * time.Minute)  // Recycle old connections

// Always handle errors in transactions
tx := db.Begin()
defer func() {
    if r := recover(); r != nil {
        tx.Rollback()
    }
}()
// ... operations
if err := tx.Commit(); err != nil {
    tx.Rollback()
    return err
}
```

**Warning signs:** Application performance degrades over time; connection count grows but never decreases; errors like "too many connections" or timeouts waiting for connections.

### Pitfall 4: Encryption Key Rotation Not Planned

**What goes wrong:** Encryption key needs rotation (compliance, breach, expiry) but no mechanism exists to re-encrypt existing data.

**Why it happens:** Implementing encryption without considering key lifecycle management.

**How to avoid:**
- Plan for key rotation from the start (even if deferred to later phase)
- Consider storing key version with encrypted data
- For Phase 2: Document that key rotation is deferred, single key acceptable for MVP
- Future: Migration to re-encrypt data with new key, or dual-key decryption during transition

**Warning signs:** No key versioning in encrypted data; no documentation on key rotation process; encryption key hard-coded or version-controlled.

### Pitfall 5: Timezone Confusion in Timestamps

**What goes wrong:** Timestamps stored in local time or inconsistent timezones, causing incorrect comparisons and display bugs.

**Why it happens:** Not enforcing UTC storage, database/application timezone mismatch.

**How to avoid:**
```go
// GORM model: Use time.Time (not string) for timestamps
type User struct {
    LastLoginAt    *time.Time  // Stored as TIMESTAMPTZ in Postgres (UTC)
}

// Database: Use TIMESTAMPTZ columns
CREATE TABLE users (
    last_login_at TIMESTAMPTZ  -- Postgres stores UTC, converts on read
);

// Application: Store in UTC, convert in UI layer
user.LastLoginAt = &time.Now().UTC()

// DSN: Specify UTC timezone
dsn := "host=localhost ... TimeZone=UTC"
```

**Warning signs:** Time comparisons fail across daylight saving changes; timestamps display differently in different environments; using string fields for timestamps.

### Pitfall 6: Soft Delete Queries Without Awareness

**What goes wrong:** Queries return fewer results than expected because soft-deleted records are filtered, or updates accidentally include soft-deleted records.

**Why it happens:** Forgetting that gorm.DeletedAt automatically filters queries.

**How to avoid:**
```go
// Understand default behavior
db.Find(&users)              // Excludes soft-deleted (deleted_at IS NULL)
db.Unscoped().Find(&users)   // Includes soft-deleted

// Restore soft-deleted record
db.Model(&user).Unscoped().Update("deleted_at", nil)

// Count including deleted
db.Unscoped().Model(&User{}).Count(&count)
```

**Warning signs:** User reports "missing" records that exist in database; count queries don't match expectations; restore functionality doesn't work.

### Pitfall 7: Nonce Reuse in GCM Encryption

**What goes wrong:** Reusing the same nonce with the same key completely breaks GCM security, allowing plaintext recovery and authentication bypass.

**Why it happens:** Using predictable nonces (counter, timestamp), or accidentally using the same nonce for multiple encryptions.

**How to avoid:**
```go
// CORRECT: Random nonce for every encryption
nonce := make([]byte, gcm.NonceSize())  // 12 bytes for standard GCM
if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
    return "", err
}
ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)  // Prepend nonce to ciphertext

// WRONG: Predictable or reused nonce
nonce := make([]byte, 12)  // All zeros - NEVER DO THIS
```

**Warning signs:** Nonce generation not using crypto/rand; nonce is constant or sequential; same nonce used for multiple Seal() calls.

### Pitfall 8: Not Testing Down Migrations

**What goes wrong:** Down migration fails in production during rollback, leaving database in inconsistent state.

**Why it happens:** Developers test up migrations but skip testing down migrations, assuming they'll work.

**How to avoid:**
```bash
# Test migration cycle in development
migrate -path migrations -database "postgres://..." up
migrate -path migrations -database "postgres://..." down 1
migrate -path migrations -database "postgres://..." up

# Verify data integrity after down/up cycle
```

**Warning signs:** Down migration files exist but are untested; down migration SQL is incomplete or incorrect; no CI/CD validation of down migrations.

## Code Examples

Verified patterns from official sources:

### Database Initialization with Connection Pool
```go
// Source: https://gorm.io/docs/connecting_to_the_database.html
package database

import (
    "fmt"
    "time"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
)

func Init(host, user, password, dbname string, port int) (*gorm.DB, error) {
    dsn := fmt.Sprintf(
        "host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=UTC",
        host, user, password, dbname, port,
    )

    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    if err != nil {
        return nil, fmt.Errorf("failed to connect database: %w", err)
    }

    sqlDB, err := db.DB()
    if err != nil {
        return nil, fmt.Errorf("failed to get database instance: %w", err)
    }

    // Connection pool configuration
    sqlDB.SetMaxOpenConns(20)
    sqlDB.SetMaxIdleConns(10)
    sqlDB.SetConnMaxLifetime(5 * time.Minute)

    return db, nil
}
```

### Complete Model with Soft Delete and Associations
```go
// Source: https://gorm.io/docs/models.html, https://gorm.io/docs/belongs_to.html
package models

import (
    "time"
    "gorm.io/gorm"
)

type User struct {
    gorm.Model
    Email              string          `gorm:"uniqueIndex;not null"`
    Name               string          `gorm:"not null"`
    Timezone           string          `gorm:"not null;default:'UTC'"`
    PreferredBriefingTime string       `gorm:"not null;default:'06:00'"`
    Role               string          `gorm:"not null;default:'user'"`
    LastLoginAt        *time.Time
    LastBriefingAt     *time.Time

    // Associations
    AuthIdentities     []AuthIdentity  `gorm:"constraint:OnDelete:CASCADE;"`
    Briefings          []Briefing      `gorm:"constraint:OnDelete:CASCADE;"`
}

type AuthIdentity struct {
    gorm.Model
    UserID         uint       `gorm:"not null;index"`
    User           User       `gorm:"constraint:OnDelete:CASCADE;"`
    Provider       string     `gorm:"not null;index"`
    ProviderUserID string     `gorm:"not null"`
    AccessToken    string     `gorm:"type:text"`
    RefreshToken   string     `gorm:"type:text"`
    TokenExpiry    *time.Time
}

// Composite unique index: one identity per provider per user
func (AuthIdentity) TableName() string {
    return "auth_identities"
}

type Briefing struct {
    gorm.Model
    UserID   uint            `gorm:"not null;index"`
    User     User            `gorm:"constraint:OnDelete:CASCADE;"`
    Content  datatypes.JSON  `gorm:"type:jsonb"`
    Status   string          `gorm:"not null;default:'pending';index"`
    ErrorMsg string          `gorm:"type:text"`
    ReadAt   *time.Time
}
```

### AES-256-GCM Encryption Helper
```go
// Source: https://pkg.go.dev/crypto/cipher, Go standard library examples
package crypto

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "encoding/base64"
    "errors"
    "io"
    "os"
)

var encryptionKey []byte

func init() {
    // Load 32-byte key from environment (AES-256)
    keyStr := os.Getenv("ENCRYPTION_KEY")
    if keyStr == "" {
        panic("ENCRYPTION_KEY environment variable not set")
    }

    decoded, err := base64.StdEncoding.DecodeString(keyStr)
    if err != nil {
        panic("invalid ENCRYPTION_KEY format: " + err.Error())
    }

    if len(decoded) != 32 {
        panic("ENCRYPTION_KEY must be 32 bytes for AES-256")
    }

    encryptionKey = decoded
}

func Encrypt(plaintext string) (string, error) {
    if plaintext == "" {
        return "", nil
    }

    block, err := aes.NewCipher(encryptionKey)
    if err != nil {
        return "", err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }

    // Generate random nonce (CRITICAL: must be unique per encryption)
    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return "", err
    }

    // Encrypt and prepend nonce to ciphertext
    ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

    return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func Decrypt(ciphertext string) (string, error) {
    if ciphertext == "" {
        return "", nil
    }

    enc, err := base64.StdEncoding.DecodeString(ciphertext)
    if err != nil {
        return "", err
    }

    block, err := aes.NewCipher(encryptionKey)
    if err != nil {
        return "", err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }

    nonceSize := gcm.NonceSize()
    if len(enc) < nonceSize {
        return "", errors.New("ciphertext too short")
    }

    // Extract nonce from beginning of ciphertext
    nonce, ciphertext := enc[:nonceSize], enc[nonceSize:]

    plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return "", err
    }

    return string(plaintext), nil
}
```

### GORM Hooks for Encryption
```go
// Source: https://gorm.io/docs/hooks.html
package models

import (
    "gorm.io/gorm"
    "yourapp/internal/crypto"
)

// BeforeSave encrypts tokens before saving to database
func (a *AuthIdentity) BeforeSave(tx *gorm.DB) error {
    // Only encrypt if values have changed (not already encrypted)
    if tx.Statement.Changed("AccessToken") && a.AccessToken != "" {
        encrypted, err := crypto.Encrypt(a.AccessToken)
        if err != nil {
            return err
        }
        a.AccessToken = encrypted
    }

    if tx.Statement.Changed("RefreshToken") && a.RefreshToken != "" {
        encrypted, err := crypto.Encrypt(a.RefreshToken)
        if err != nil {
            return err
        }
        a.RefreshToken = encrypted
    }

    return nil
}

// AfterFind decrypts tokens after loading from database
func (a *AuthIdentity) AfterFind(tx *gorm.DB) error {
    if a.AccessToken != "" {
        decrypted, err := crypto.Decrypt(a.AccessToken)
        if err != nil {
            return err
        }
        a.AccessToken = decrypted
    }

    if a.RefreshToken != "" {
        decrypted, err := crypto.Decrypt(a.RefreshToken)
        if err != nil {
            return err
        }
        a.RefreshToken = decrypted
    }

    return nil
}
```

### Running Embedded Migrations
```go
// Source: https://github.com/golang-migrate/migrate
package database

import (
    "embed"
    "fmt"
    "github.com/golang-migrate/migrate/v4"
    _ "github.com/golang-migrate/migrate/v4/database/postgres"
    "github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func RunMigrations(dbURL string) error {
    d, err := iofs.New(migrationsFS, "migrations")
    if err != nil {
        return fmt.Errorf("failed to create migration source: %w", err)
    }

    m, err := migrate.NewWithSourceInstance("iofs", d, dbURL)
    if err != nil {
        return fmt.Errorf("failed to create migrate instance: %w", err)
    }

    // Run all pending migrations
    if err := m.Up(); err != nil && err != migrate.ErrNoChange {
        return fmt.Errorf("migration failed: %w", err)
    }

    return nil
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| GORM v1 with database/sql driver | GORM v2 with pgx driver | 2020 | Better performance, prepared statement cache, native Postgres features |
| GORM AutoMigrate for production | golang-migrate versioned migrations | Always recommended | Version control, rollbacks, team collaboration, audit trail |
| Manual timestamp tracking | gorm.Model with CreatedAt/UpdatedAt | Always available | Automatic tracking, consistency, less boilerplate |
| String-based soft delete flags | gorm.DeletedAt with nullable timestamp | GORM v2+ | Tracks when deleted, automatic filtering, better semantics |
| lib/pq driver | pgx/v5 driver | 2023+ | Active maintenance, better performance, more features |
| Go 1.15 bindata tools | Go 1.16+ embed directive | Go 1.16 (2021) | Standard library support, simpler, type-safe |
| GORM generics not available | GORM generics support (v1.30.0+) | 2025 | Type safety, better IDE support, reduced errors |

**Deprecated/outdated:**
- **lib/pq driver**: Maintenance mode, use pgx-based drivers (gorm.io/driver/postgres uses pgx)
- **GORM v1**: Use GORM v2 for new projects (different import path: gorm.io/gorm)
- **go-bindata for embedding**: Use Go 1.16+ //go:embed directive
- **GORM AutoMigrate in production**: Never recommended, use versioned migrations
- **Hard deletes for user data**: Use soft deletes to prevent accidental data loss and support restore functionality

## Open Questions

1. **Migration failure recovery in production**
   - What we know: golang-migrate enters "dirty" state on failure, requires manual intervention
   - What's unclear: Best practice for automated recovery or alerting
   - Recommendation: Start with manual recovery process, document procedure, add monitoring for dirty state in future phase

2. **Encryption key rotation mechanism**
   - What we know: Encryption via GORM hooks works for single key
   - What's unclear: How to rotate keys for existing encrypted data
   - Recommendation: Single key acceptable for Phase 2, document that rotation is deferred, consider key versioning field for future

3. **Connection pool sizing for production**
   - What we know: Recommended starting point is MaxOpenConns=20, MaxIdleConns=10
   - What's unclear: Optimal settings for production workload
   - Recommendation: Start with conservative defaults, monitor connection usage, tune based on metrics in later phase

4. **Seed data strategy for multiple environments**
   - What we know: Seed data can be a migration file
   - What's unclear: How to prevent seed data from running in production
   - Recommendation: Use environment variable check in seed migration, or separate seed command for dev only

## Sources

### Primary (HIGH confidence)
- [GORM Official Documentation](https://gorm.io/docs/index.html) - Core GORM features, setup, conventions
- [GORM Hooks Documentation](https://gorm.io/docs/hooks.html) - Hook implementation and lifecycle
- [GORM Models Documentation](https://gorm.io/docs/models.html) - Model declaration, tags, best practices
- [GORM Conventions Documentation](https://gorm.io/docs/conventions.html) - Naming conventions, timestamps
- [GORM Delete Documentation](https://gorm.io/docs/delete.html) - Soft delete behavior with DeletedAt
- [GORM Database Connection Documentation](https://gorm.io/docs/connecting_to_the_database.html) - Connection setup and pooling
- [GORM Migration Documentation](https://gorm.io/docs/migration.html) - AutoMigrate capabilities and limitations
- [GORM Belongs To Documentation](https://gorm.io/docs/belongs_to.html) - Foreign key relationships
- [GORM Constraints Documentation](https://gorm.io/docs/constraints.html) - Foreign key constraints, cascade
- [GORM Indexes Documentation](https://gorm.io/docs/indexes.html) - Index and unique constraint tags
- [golang-migrate GitHub Repository](https://github.com/golang-migrate/migrate) - Official documentation and examples
- [gorm.io/driver/postgres Package Documentation](https://pkg.go.dev/gorm.io/driver/postgres) - Postgres driver configuration
- [crypto/cipher Package Documentation](https://pkg.go.dev/crypto/cipher) - GCM AEAD implementation
- [Docker Postgres Official Documentation](https://docs.docker.com/reference/samples/postgres/) - Docker Compose patterns

### Secondary (MEDIUM confidence)
- [How to Implement Database Migrations in Go with golang-migrate (2026-01-07)](https://oneuptime.com/blog/post/2026-01-07-go-database-migrations/) - Recent golang-migrate best practices
- [How to Handle Database Migrations in Go Projects (2026-02-01)](https://oneuptime.com/blog/post/2026-02-01-go-database-migrations/) - Production migration patterns
- [Encryption Using AES-GCM in Go (2026-01)](https://karbhawono.medium.com/encryption-using-aes-gcm-b981bf4890f3) - Recent Go AES-GCM example
- [Custom Tag-based field encryption/decryption using GORM hooks](https://medium.com/impelsys/custom-tag-based-field-encryption-decryption-using-gorm-hooks-blog-ii-of-blog-series-5f527293ca39) - GORM encryption pattern
- [Database migrations in Go with golang-migrate](https://betterstack.com/community/guides/scaling-go/golang-migrate/) - Comprehensive guide
- [Building Robust Go Applications with GORM: Best Practices](https://www.pingcap.com/article/building-robust-go-applications-with-gorm-best-practices/) - GORM patterns
- [Common GORM Mistakes You Should Avoid](https://hadijafari.me/common-gorm-mistakes-you-should-avoid/) - Pitfalls catalog
- [AutoMigrate VS golang-migrate Discussion](https://github.com/go-gorm/gorm/discussions/5730) - Community comparison
- [Handling Migration Errors: How Atlas Improves on golang-migrate](https://atlasgo.io/blog/2025/04/06/atlas-and-golang-migrate) - Migration pitfalls analysis
- [Best Practices for Running PostgreSQL in Docker](https://sliplane.io/blog/best-practices-for-postgres-in-docker) - Docker Compose patterns
- [PostgreSQL in Docker: Quick Setup and Getting Started Guide (2026)](https://utho.com/blog/postgresql-docker-setup/) - Recent Docker setup

### Tertiary (LOW confidence)
- WebSearch results for ecosystem trends and community patterns (verified against official docs)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Official GORM and golang-migrate documentation, widespread adoption verified
- Architecture: HIGH - Patterns from official docs, verified code examples
- Encryption: HIGH - Go standard library documentation, established patterns
- Pitfalls: MEDIUM-HIGH - Mix of official documentation and verified community reports
- Connection pooling: MEDIUM - Recommended defaults from multiple sources, production tuning is workload-dependent

**Research date:** 2026-02-11
**Valid until:** 2026-03-11 (30 days - stable ecosystem, GORM v2 and golang-migrate v4 are mature)
