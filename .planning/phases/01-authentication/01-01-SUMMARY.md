---
phase: 01-authentication
plan: 01
subsystem: auth-backend
tags: [gin, goth, oauth, sessions, middleware]
dependency-graph:
  requires: [go-environment]
  provides: [gin-router, google-oauth, session-management, auth-middleware]
  affects: [cmd/server/main.go, internal/auth/*, internal/config/*]
tech-stack:
  added: [gin-gonic/gin, gin-contrib/sessions, markbates/goth]
  patterns: [oauth2-flow, session-cookies, route-middleware]
key-files:
  created:
    - internal/config/config.go
    - internal/auth/goth.go
    - internal/auth/handlers.go
    - internal/auth/middleware.go
  modified:
    - cmd/server/main.go
    - go.mod
decisions:
  - choice: "Use SameSite=Lax for session cookies"
    rationale: "Strict mode blocks OAuth redirect callbacks from Google, breaking the auth flow"
  - choice: "Store user info directly in session (not database) for Phase 1"
    rationale: "Defers GORM/Postgres complexity to Phase 2 while still satisfying session persistence requirement"
  - choice: "Cookie-based session store (not Redis)"
    rationale: "Redis infrastructure doesn't exist until Phase 3; cookie store still provides 30-day persistence"
metrics:
  duration: 6
  completed: 2026-02-11
  tasks: 2
  commits: 2
---

# Phase 01 Plan 01: Backend Authentication Infrastructure Summary

**Replaced net/http with Gin framework, configured Google OAuth via Goth, implemented session-based authentication with cookie storage, and created complete auth handlers plus route protection middleware with HTMX support.**

## What Was Built

### Core Infrastructure
- **Gin Router Setup**: Replaced net/http.ServeMux with gin.Default(), providing robust routing and middleware support
- **Configuration Management**: Created centralized config package loading environment variables with sensible defaults
- **Session Middleware**: Configured cookie-based session store with 30-day MaxAge, HttpOnly, and SameSite=Lax settings
- **Goth OAuth Integration**: Initialized Google OAuth provider with email and profile scopes

### Authentication Handlers
- **HandleLogin**: Initiates Google OAuth flow by calling gothic.BeginAuthHandler
- **HandleCallback**: Completes OAuth flow, stores user info (ID, email, name, avatar) in session, redirects to dashboard
- **HandleLogout**: Clears session and redirects to login page

### Route Protection
- **RequireAuth Middleware**: Checks session for authenticated user, supports both standard HTTP redirects (302) and HTMX-aware redirects (401 + HX-Redirect header)
- **Route Organization**: Public routes (/login, /auth/google, /auth/google/callback) and protected route group (/, /dashboard, /logout)

## Deviations from Plan

None - plan executed exactly as written. All tasks completed without requiring architectural changes or unplanned work.

## Technical Implementation Details

### Session Configuration
```go
store.Options(sessions.Options{
    Path:     "/",
    MaxAge:   86400 * 30, // 30 days - satisfies AUTH-02 persistence requirement
    HttpOnly: true,
    Secure:   cfg.Env == "production",
    SameSite: http.SameSiteLaxMode, // Critical: Strict breaks OAuth redirects
})
```

### HTMX-Compatible Auth
The middleware detects HTMX requests via the `HX-Request` header and returns appropriate responses:
- Standard requests: 302 redirect to /login
- HTMX requests: 401 status with `HX-Redirect: /login` header

This allows HTMX-powered UIs to handle authentication redirects client-side without full page reloads.

### Environment Variables Required for OAuth
- `GOOGLE_CLIENT_ID`: OAuth client ID from Google Cloud Console
- `GOOGLE_CLIENT_SECRET`: OAuth client secret
- `GOOGLE_CALLBACK_URL`: Callback URL (http://localhost:8080/auth/google/callback for dev)
- `SESSION_SECRET`: Secure random string (generate with `openssl rand -hex 32`)

The application starts without credentials (with warnings) to allow build verification, but OAuth login requires all variables to be set.

## Verification Results

All verification steps passed:
- `go build ./...`: Clean compilation with zero errors
- `go vet ./...`: No issues reported
- Server startup: Successful on port 8080
- `/health` endpoint: Returns `{"status":"ok"}`
- `/login` endpoint: Returns placeholder text (200)
- `/dashboard` (unauthenticated): Redirects to /login (302)
- `/dashboard` (HTMX request): Returns 401 with HX-Redirect header
- `/auth/google`: Returns 400 without credentials (expected behavior)

## Files Created/Modified

### Created
- `internal/config/config.go` (41 lines): Centralized configuration loading from environment
- `internal/auth/goth.go` (27 lines): Goth OAuth provider initialization
- `internal/auth/handlers.go` (64 lines): Login, callback, and logout handlers
- `internal/auth/middleware.go` (35 lines): Authentication middleware with HTMX support

### Modified
- `cmd/server/main.go`: Replaced net/http with Gin, added session middleware, wired auth routes
- `go.mod`: Added dependencies for Gin, sessions, and Goth

## Task Commits

| Task | Description | Commit | Files |
|------|-------------|--------|-------|
| 1 | Replace net/http with Gin, configure sessions and Goth | 561e8fe | cmd/server/main.go, go.mod, go.sum, internal/config/config.go, internal/auth/goth.go |
| 2 | Create auth handlers and middleware, wire routes | 8324f51 | cmd/server/main.go, internal/auth/handlers.go, internal/auth/middleware.go |

## Dependencies Added

```
github.com/gin-gonic/gin v1.11.0
github.com/gin-contrib/sessions v1.0.4
github.com/gin-contrib/sessions/cookie (via sessions)
github.com/markbates/goth v1.82.0
github.com/markbates/goth/gothic (via goth)
github.com/markbates/goth/providers/google (via goth)
```

## Next Steps

Plan 02 will build the UI layer:
- Create Templ templates for login and dashboard pages
- Replace placeholder handlers with template rendering
- Add user profile display to dashboard
- Implement complete user authentication flow (login button → OAuth → dashboard)

## Self-Check: PASSED

All claims verified successfully.

### File Existence Check
- FOUND: internal/config/config.go
- FOUND: internal/auth/goth.go
- FOUND: internal/auth/handlers.go
- FOUND: internal/auth/middleware.go

### Commit Existence Check
- FOUND: 561e8fe (Task 1)
- FOUND: 8324f51 (Task 2)
