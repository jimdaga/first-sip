# Stack Research

**Domain:** Go Web Application with Server-Side Rendering (HTMX/Templ)
**Researched:** 2026-02-10
**Overall Confidence:** MEDIUM-LOW (training data from January 2025, external verification unavailable)

## Recommended Stack

### Core Technologies

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| Go | 1.23+ | Runtime & standard library | Already in use, excellent stdlib support for HTTP servers, Go 1.23 adds routing improvements to net/http |
| Gin | v1.10.0+ | HTTP router & middleware | Battle-tested, excellent middleware ecosystem, 15x faster routing than stdlib in benchmarks, widespread adoption |
| Templ | v0.2.747+ | Type-safe HTML templating | Compile-time type safety, generates Go code, works seamlessly with HTMX, better DX than html/template |
| HTMX | v2.0.0+ | Client-side interactivity | Hypermedia-driven, minimal JS, perfect for server-rendered apps, v2 has breaking changes from v1 |
| GORM | v1.25.11+ | ORM for Postgres | Most popular Go ORM, excellent Postgres support, hooks system, migration support |
| Asynq | v0.24.1+ | Background job queue | Redis-backed, built-in retries, scheduled tasks, monitoring UI, better than gocraft/work |
| Goth | v1.80.0+ | Multi-provider OAuth | Supports 40+ providers including Google, simple API, active maintenance |

### Database & Cache

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| PostgreSQL | 16+ | Primary database | Rock-solid ACID compliance, excellent GORM support, JSON types for flexible schemas |
| Redis | 7+ | Job queue & cache | Required for Asynq, can also cache session data, pub/sub for real-time features |

### Frontend Styling

| Library | Version | Purpose | Why Recommended |
|---------|---------|---------|-----------------|
| Tailwind CSS | v3.4+ | Utility-first CSS | Industry standard, excellent with Templ (use class strings), JIT mode for dev |
| DaisyUI | v4.12+ | Tailwind component library | Pre-built components (buttons, modals, forms), reduces custom CSS, themeable |

### Supporting Libraries

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| gorilla/sessions | v1.3.0+ | Session management | Store OAuth tokens, user sessions; works with Redis backend |
| go-playground/validator | v10.22+ | Struct validation | Validate form inputs, API requests; integrates with Gin |
| joho/godotenv | v1.5.1+ | Environment config | Development environment variables; don't use in production (use k8s ConfigMaps) |
| lib/pq | v1.10.9+ | Postgres driver | Required by GORM for Postgres; use this not pgx for GORM compatibility |
| redis/go-redis | v9.6+ | Redis client | Required by Asynq, can also use directly for caching |
| rs/zerolog | v1.33+ | Structured logging | Better than log/slog for JSON logs, excellent performance, works with Gin middleware |

### Development Tools

| Tool | Purpose | Notes |
|------|---------|-------|
| templ CLI | Generate Go from .templ files | Install: `go install github.com/a-h/templ/cmd/templ@latest` |
| air | Hot reload during dev | Watch .go and .templ files, restart server automatically |
| golangci-lint | Linting | Use with gosec, govet, staticcheck enabled |
| sqlc (optional) | Type-safe SQL | Alternative to GORM if you want raw SQL with type safety |

## Installation

```bash
# Core framework
go get github.com/gin-gonic/gin@v1.10.0

# Templating
go get github.com/a-h/templ@v0.2.747
go install github.com/a-h/templ/cmd/templ@latest

# Database
go get gorm.io/gorm@v1.25.11
go get gorm.io/driver/postgres@v1.5.9

# Background jobs
go get github.com/hibiken/asynq@v0.24.1
go get github.com/redis/go-redis/v9@v9.6.0

# OAuth
go get github.com/markbates/goth@v1.80.0
go get github.com/gorilla/sessions@v1.3.0

# Validation & logging
go get github.com/go-playground/validator/v10@v10.22.0
go get github.com/rs/zerolog@v1.33.0

# Development only
go get github.com/joho/godotenv@v1.5.1
```

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|-------------------------|
| Gin | Echo, Fiber, stdlib net/http | Echo if you need HTTP/2 push; Fiber if migrating from Express.js; stdlib for minimalism |
| Templ | html/template, gomponents | html/template if no build step desired; gomponents if prefer pure Go over DSL |
| GORM | sqlc, sqlx | sqlc for type-safe raw SQL; sqlx for lighter-weight than GORM but no ORM features |
| Asynq | gocraft/work, machinery | machinery if you need multi-broker support; gocraft/work is older, less active |
| Goth | go-oauth2/oauth2 | google.golang.org/oauth2 if only need Google and want official library |
| Tailwind+DaisyUI | Bulma, Bootstrap | Only if team strongly prefers traditional CSS frameworks |

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| html/template for complex UIs | No type safety, runtime errors, poor composition | Templ |
| gorilla/mux | Maintenance mode, stdlib routing caught up in Go 1.22+ | Gin or stdlib net/http |
| GORM AutoMigrate in production | Dangerous schema changes, no rollback | Write explicit migrations with golang-migrate or GORM Migrator manually |
| pgx driver with GORM | GORM officially recommends lib/pq, pgx has compatibility issues | lib/pq (gorm.io/driver/postgres uses this) |
| Alpine.js with HTMX | Redundant, adds complexity | HTMX alone handles most use cases |
| JWT for sessions | Stateless = can't revoke, harder to manage | Server-side sessions with gorilla/sessions + Redis |

## Version Compatibility Matrix

| Package | Version | Compatible With | Notes |
|---------|---------|-----------------|-------|
| Go | 1.23+ | All packages | Go 1.22+ required for stdlib routing patterns |
| Gin | v1.10.0 | Go 1.21+ | v1.10 added better performance and bug fixes |
| Templ | v0.2.747+ | Go 1.21+ | Generates Go 1.21+ code, fast-moving project |
| HTMX | v2.0.0+ | Browser-side only | v2 breaking changes: hx-boost behavior, WebSocket syntax |
| GORM | v1.25.11 | lib/pq v1.10.9 | Use matching gorm.io/driver/postgres version |
| Asynq | v0.24.1 | Redis 6.2+, go-redis v9 | Redis 7+ recommended for better reliability |
| Goth | v1.80.0 | gorilla/sessions v1.2+ | Pin version, breaking changes common |

## Critical Gotchas

### Templ
- **Build step required**: Must run `templ generate` before `go build`
- **IDE setup**: Need templ LSP for syntax highlighting (VS Code extension: `a-h.templ`)
- **HTMX integration**: Use `templ.Raw()` sparingly, prefer typed attributes
- **Watch mode**: Use `templ generate --watch` or Air with `.templ` file watching

### HTMX
- **Version 2 breaking changes**: `hx-boost` now uses `hx-boost="true"`, WebSocket changed to `hx-ws`
- **Headers matter**: Return `HX-Trigger`, `HX-Redirect` headers for client-side behavior
- **Form handling**: Gin's `c.ShouldBind()` works with HTMX form posts
- **CDN vs bundle**: Use CDN for simplicity unless you need specific version pinning

### GORM
- **N+1 queries**: Always use `Preload()` for relations, not lazy loading
- **AutoMigrate dangers**: Only use in development, write explicit migrations for production
- **Connection pooling**: Configure `SetMaxOpenConns()` and `SetMaxIdleConns()` for performance
- **Soft deletes**: `gorm.Model` includes DeletedAt, queries auto-filter unless `Unscoped()`

### Asynq
- **Task serialization**: Task payloads must be JSON-serializable, use `json.Marshal`
- **Retry configuration**: Default is 25 retries with exponential backoff, tune per task
- **Queue naming**: Use separate queues for different priorities (`critical`, `default`, `low`)
- **Monitoring**: Asynqmon web UI is separate binary, deploy alongside workers

### Goth
- **Session storage**: MUST use gorilla/sessions, configure `goth.UseProviders()` before router
- **Callback URLs**: Set `GOOGLE_CALLBACK_URL` env var, must match Google Console exactly
- **State parameter**: Goth handles CSRF protection via state, don't disable
- **Multi-provider**: Even with only Google, initialize with slice for future providers

### Gin + Templ Integration
- **Content-Type header**: Set `c.Header("Content-Type", "text/html; charset=utf-8")`
- **Rendering**: `c.Status(200)` then `component.Render(c.Request.Context(), c.Writer)`
- **Middleware order**: Session middleware before Goth, logging middleware first
- **Static files**: Use `router.Static("/static", "./static")` for CSS/JS

## Project Structure Recommendation

```
/cmd/server/          - main.go, server bootstrap
/internal/
  /auth/              - Goth integration, OAuth handlers
  /handlers/          - HTTP handlers (Gin controllers)
  /models/            - GORM models
  /services/          - Business logic
  /workers/           - Asynq task handlers
  /templates/         - Templ components (.templ files)
  /middleware/        - Gin middleware (auth, logging)
/pkg/                 - Shared utilities
/migrations/          - SQL migration files
/static/              - CSS, JS, images
/config/              - Configuration loading
```

## Configuration Best Practices

### Environment Variables
```bash
# Database
DATABASE_URL=postgres://user:pass@localhost:5432/dbname

# Redis
REDIS_URL=redis://localhost:6379

# OAuth
GOOGLE_CLIENT_ID=your-client-id
GOOGLE_CLIENT_SECRET=your-secret
GOOGLE_CALLBACK_URL=http://localhost:8080/auth/google/callback

# Session
SESSION_SECRET=generate-with-openssl-rand-hex-32

# n8n
N8N_WEBHOOK_URL=http://n8n:5678/webhook/briefing
```

### Connection Pooling
```go
// Postgres (GORM)
sqlDB, _ := db.DB()
sqlDB.SetMaxOpenConns(25)
sqlDB.SetMaxIdleConns(5)
sqlDB.SetConnMaxLifetime(5 * time.Minute)

// Redis (Asynq)
asynq.RedisClientOpt{
  Addr: redisURL,
  PoolSize: 10,
}
```

## Performance Considerations

| Concern | Recommendation | Why |
|---------|---------------|-----|
| Database queries | Use GORM's `Preload` for relations | Avoid N+1, single query per relation |
| Session storage | Use Redis backend for gorilla/sessions | In-memory sessions lost on restart |
| Static assets | Serve via CDN or nginx in prod | Gin static serving is not optimized |
| Background jobs | Use Asynq queues with priority | Separate critical from low-priority |
| Template rendering | Templ pre-compiles to Go | Faster than html/template runtime parsing |
| HTMX responses | Return minimal HTML fragments | Don't re-render entire page |

## Security Checklist

- [ ] Use HTTPS in production (configure Gin with TLS or behind Traefik)
- [ ] Set `Secure`, `HttpOnly`, `SameSite=Strict` on session cookies
- [ ] Validate all form inputs with go-playground/validator
- [ ] Use GORM parameterized queries (default behavior, avoid raw SQL strings)
- [ ] Rate limit OAuth endpoints (use Gin rate-limit middleware)
- [ ] Set CORS headers appropriately if exposing APIs (use gin-contrib/cors)
- [ ] Don't log sensitive data (passwords, tokens, session IDs)
- [ ] Use gorilla/csrf for CSRF protection on forms (HTMX auto-includes tokens)

## Testing Strategy

| Layer | Tool | Approach |
|-------|------|----------|
| HTTP handlers | httptest | Mock Gin context, test handler functions |
| Templ components | templ_test | Render to buffer, assert HTML output |
| GORM models | testcontainers-go | Spin up Postgres container for integration tests |
| Asynq tasks | asynqtest | Test task handlers with mock Redis |
| OAuth flow | httptest | Mock Goth provider responses |

## Monitoring & Observability

| Tool | Purpose | Integration |
|------|---------|-------------|
| zerolog | Structured logging | Gin middleware for request logs |
| Asynqmon | Job queue monitoring | Separate web UI on port 8081 |
| GORM logger | SQL query logging | Configure log level (warn in prod, info in dev) |
| Gin recovery | Panic recovery | Built-in middleware, log stack traces |

## Migration Path from Stdlib

Your current setup uses `net/http.ServeMux`. Migration steps:

1. **Install Gin** → Replace `http.NewServeMux()` with `gin.Default()`
2. **Add Templ** → Replace any `html/template` with `.templ` files
3. **Add GORM** → Connect to Postgres, define models
4. **Add Asynq** → Set up worker and client, define task types
5. **Add Goth** → Configure Google provider, add auth routes
6. **Add HTMX** → Include CDN script, enhance forms with `hx-*` attributes
7. **Add Tailwind+DaisyUI** → Include CDN or build pipeline

Gin is compatible with existing `http.Handler` via `gin.WrapH()` if you need gradual migration.

## Common Pitfalls

### Templ + HTMX Integration
**Problem:** HTMX requests return full page instead of fragment
**Solution:** Check `c.GetHeader("HX-Request")` and render partial vs full layout

### GORM Connection Leaks
**Problem:** "Too many connections" errors
**Solution:** Always configure connection pool limits, use `defer db.Close()` pattern

### Asynq Task Serialization
**Problem:** Tasks fail to deserialize in workers
**Solution:** Use struct tags `json:"field_name"`, test serialization in unit tests

### Goth Session Management
**Problem:** "Session not found" errors after OAuth callback
**Solution:** Ensure session middleware is registered before Goth routes

### HTMX + Tailwind Class Conflicts
**Problem:** HTMX transitions break Tailwind animations
**Solution:** Use HTMX `hx-swap` with `swap:` timing to coordinate with Tailwind transitions

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Go stdlib | HIGH | Well-known, stable API |
| Gin framework | MEDIUM | Popular but version numbers may be dated |
| Templ | LOW | Fast-moving project, version may have changed |
| HTMX | MEDIUM | v2 released mid-2024, should be stable |
| GORM | MEDIUM | Version numbers likely dated but API stable |
| Asynq | LOW | Version numbers may be stale |
| Goth | LOW | Fast-moving, breaking changes common |
| Integration patterns | MEDIUM | Based on training data, not verified externally |

## Sources

**Note:** WebSearch and WebFetch were unavailable during research. All information based on training data (January 2025 cutoff). Recommend verifying:

1. Current versions via `go list -m -versions <package>`
2. Official documentation for breaking changes
3. GitHub release notes for each package
4. Community best practices (Reddit r/golang, Gophers Slack)

**Critical verification needed:**
- Templ version and API stability (fast-moving project)
- HTMX v2 adoption and ecosystem maturity
- Asynq latest features and monitoring tools
- Goth Google provider recent changes

---
*Stack research for: Daily Briefing Go Web App*
*Researched: 2026-02-10*
*Limitations: External verification unavailable, based on January 2025 training data*
