# Phase 1: Authentication - Research

**Researched:** 2026-02-10
**Domain:** Google OAuth Authentication with Go/Gin/Goth/GORM
**Confidence:** MEDIUM

## Summary

Google OAuth authentication in Go is best implemented using the Goth library (v1.82.0 as of August 2025) with Gin framework integration. The standard pattern uses gorilla/sessions for session management (ideally with Redis backend for persistence), GORM hooks for transparent OAuth token encryption, and HTMX for seamless client-side redirects. The authentication flow follows: user clicks "Login with Google" → Goth initiates OAuth → Google callback → store encrypted tokens and session → redirect to dashboard. Protected routes use Gin middleware that checks sessions and aborts requests with HX-Redirect headers for HTMX compatibility.

The stack is mature and well-documented with established patterns. Key risks include: Goth's fast-moving nature (breaking changes common), session security misconfiguration, and HTMX redirect handling complexity. The gin-contrib/sessions package provides clean Redis integration, avoiding the deprecated gin-gonic/contrib package.

**Primary recommendation:** Use Goth v1.82.0 with gothic helpers for Gin integration, store sessions in Redis via gin-contrib/sessions, encrypt OAuth tokens using GORM BeforeSave/AfterFind hooks with AES-256-GCM, and handle HTMX authentication redirects via HX-Redirect response header.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| github.com/markbates/goth | v1.82.0 | Multi-provider OAuth | 70+ providers including Google, 6.4k stars, active maintenance, gothic helpers simplify Gin integration |
| github.com/gin-contrib/sessions | Latest | Session middleware | Official Gin middleware, wraps gorilla/sessions, supports Redis/Postgres/MongoDB backends |
| github.com/gorilla/sessions | v1.3.0+ | Session management | Required by Goth, industry standard, flexible store backends |
| gorm.io/gorm | v1.25.11+ | ORM with hooks | BeforeSave/AfterFind hooks enable transparent encryption, most popular Go ORM |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| github.com/redis/go-redis | v9.6+ | Redis client | Session persistence across restarts, required for production |
| crypto/aes + crypto/cipher | stdlib | AES-256-GCM encryption | Encrypt OAuth tokens before DB storage |
| crypto/rand | stdlib | Secure random generation | Generate encryption keys and nonces |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Goth | golang.org/x/oauth2 | Official Google library, but only supports Google (Goth supports 70+ providers for future expansion) |
| gin-contrib/sessions | Custom gorilla/sessions | More control, but lose Gin middleware integration and need manual setup |
| GORM hooks for encryption | Application-layer encryption | More flexible, but loses transparency and requires manual encrypt/decrypt calls everywhere |

**Installation:**
```bash
# OAuth
go get github.com/markbates/goth@v1.82.0

# Sessions
go get github.com/gin-contrib/sessions
go get github.com/gorilla/sessions@v1.3.0

# Redis (for session backend)
go get github.com/redis/go-redis/v9@v9.6.0

# GORM (already in project)
# go get gorm.io/gorm@v1.25.11
```

## Architecture Patterns

### Recommended Project Structure
```
internal/
├── auth/              # OAuth integration
│   ├── goth.go        # Goth provider setup
│   ├── handlers.go    # Login/logout/callback handlers
│   └── middleware.go  # Auth required middleware
├── models/            # GORM models
│   └── user.go        # User with encrypted OAuth tokens
├── handlers/          # HTTP handlers
│   └── dashboard.go   # Protected dashboard handler
└── templates/         # Templ components
    ├── login.templ    # Login page with "Login with Google" button
    └── dashboard.templ # Protected dashboard
```

### Pattern 1: Goth Provider Initialization
**What:** Configure Goth with Google OAuth provider before router setup
**When to use:** Application startup, before defining routes
**Example:**
```go
// Source: https://github.com/markbates/goth README
package auth

import (
    "github.com/markbates/goth"
    "github.com/markbates/goth/providers/google"
    "os"
)

func InitProviders() {
    goth.UseProviders(
        google.New(
            os.Getenv("GOOGLE_CLIENT_ID"),
            os.Getenv("GOOGLE_CLIENT_SECRET"),
            os.Getenv("GOOGLE_CALLBACK_URL"), // Must match Google Console exactly
            "email", "profile", // Scopes
        ),
    )
}
```

### Pattern 2: Gin Session Middleware with Redis Backend
**What:** Configure gin-contrib/sessions with Redis for persistent session storage
**When to use:** Application startup, register as global middleware before routes
**Example:**
```go
// Source: https://pkg.go.dev/github.com/gin-contrib/sessions
package main

import (
    "github.com/gin-contrib/sessions"
    "github.com/gin-contrib/sessions/redis"
    "github.com/gin-gonic/gin"
)

func setupRouter() *gin.Engine {
    r := gin.Default()

    // Redis session store
    store, _ := redis.NewStore(
        10,                           // Pool size
        "tcp",
        "localhost:6379",
        "",                           // Password (if any)
        []byte(os.Getenv("SESSION_SECRET")), // Secret key
    )

    // Configure session options
    store.Options(sessions.Options{
        Path:     "/",
        MaxAge:   86400 * 30, // 30 days
        HttpOnly: true,
        Secure:   true, // HTTPS only in production
        SameSite: http.SameSiteStrictMode,
    })

    r.Use(sessions.Sessions("mysession", store))
    return r
}
```

### Pattern 3: OAuth Login Flow with Gothic
**What:** Use gothic helpers to simplify Goth integration with Gin
**When to use:** Authentication routes (login, callback, logout)
**Example:**
```go
// Source: https://dizzy.zone/2018/06/01/OAuth-with-Gin-and-Goth/
package auth

import (
    "github.com/gin-gonic/gin"
    "github.com/markbates/goth/gothic"
    "net/http"
)

func HandleGoogleLogin(c *gin.Context) {
    // Set provider in query for gothic
    q := c.Request.URL.Query()
    q.Add("provider", "google")
    c.Request.URL.RawQuery = q.Encode()

    // Begin OAuth flow
    gothic.BeginAuthHandler(c.Writer, c.Request)
}

func HandleGoogleCallback(c *gin.Context) {
    q := c.Request.URL.Query()
    q.Add("provider", "google")
    c.Request.URL.RawQuery = q.Encode()

    // Complete OAuth and get user info
    user, err := gothic.CompleteUserAuth(c.Writer, c.Request)
    if err != nil {
        c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
        return
    }

    // Store user ID in session
    session := sessions.Default(c)
    session.Set("user_id", user.UserID)
    session.Save()

    // Save/update user in database with encrypted tokens
    SaveOrUpdateUser(user) // Implement separately

    c.Redirect(http.StatusFound, "/dashboard")
}

func HandleLogout(c *gin.Context) {
    session := sessions.Default(c)
    session.Clear()
    session.Save()
    c.Redirect(http.StatusFound, "/")
}
```

### Pattern 4: Protected Route Middleware
**What:** Gin middleware that checks session and aborts unauthenticated requests
**When to use:** Apply to protected route groups (dashboard, API endpoints)
**Example:**
```go
// Source: https://leapcell.io/blog/secure-your-apis-with-jwt-authentication-in-gin-middleware
package auth

import (
    "github.com/gin-contrib/sessions"
    "github.com/gin-gonic/gin"
    "net/http"
)

func AuthRequired() gin.HandlerFunc {
    return func(c *gin.Context) {
        session := sessions.Default(c)
        userID := session.Get("user_id")

        if userID == nil {
            // Check if HTMX request
            if c.GetHeader("HX-Request") == "true" {
                // Tell HTMX to redirect entire page
                c.Header("HX-Redirect", "/login")
                c.AbortWithStatus(http.StatusUnauthorized)
            } else {
                c.Redirect(http.StatusFound, "/login")
                c.Abort()
            }
            return
        }

        // Store user ID in context for handlers
        c.Set("user_id", userID)
        c.Next()
    }
}

// Usage
func SetupRoutes(r *gin.Engine) {
    // Public routes
    r.GET("/login", renderLoginPage)
    r.GET("/auth/google", HandleGoogleLogin)
    r.GET("/auth/google/callback", HandleGoogleCallback)

    // Protected routes
    protected := r.Group("/")
    protected.Use(AuthRequired())
    {
        protected.GET("/dashboard", renderDashboard)
        protected.GET("/logout", HandleLogout)
    }
}
```

### Pattern 5: GORM Model with Encrypted OAuth Tokens
**What:** User model with BeforeSave/AfterFind hooks for transparent encryption
**When to use:** All models storing sensitive data (OAuth tokens, API keys)
**Example:**
```go
// Source: https://gorm.io/docs/hooks.html + https://www.twilio.com/en-us/blog/developers/community/encrypt-and-decrypt-data-in-go-with-aes-256
package models

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "encoding/base64"
    "gorm.io/gorm"
    "io"
)

type User struct {
    gorm.Model
    Email          string `gorm:"uniqueIndex;not null"`
    GoogleID       string `gorm:"uniqueIndex;not null"`
    Name           string
    AccessToken    string // Encrypted in DB
    RefreshToken   string // Encrypted in DB
    encryptionKey  []byte `gorm:"-"` // Not stored in DB
}

// BeforeSave encrypts tokens before database write
func (u *User) BeforeSave(tx *gorm.DB) error {
    if u.AccessToken != "" && !u.isEncrypted(u.AccessToken) {
        encrypted, err := u.encrypt(u.AccessToken)
        if err != nil {
            return err
        }
        u.AccessToken = encrypted
    }

    if u.RefreshToken != "" && !u.isEncrypted(u.RefreshToken) {
        encrypted, err := u.encrypt(u.RefreshToken)
        if err != nil {
            return err
        }
        u.RefreshToken = encrypted
    }

    return nil
}

// AfterFind decrypts tokens after database read
func (u *User) AfterFind(tx *gorm.DB) error {
    if u.AccessToken != "" {
        decrypted, err := u.decrypt(u.AccessToken)
        if err != nil {
            return err
        }
        u.AccessToken = decrypted
    }

    if u.RefreshToken != "" {
        decrypted, err := u.decrypt(u.RefreshToken)
        if err != nil {
            return err
        }
        u.RefreshToken = decrypted
    }

    return nil
}

func (u *User) encrypt(plaintext string) (string, error) {
    key := u.getEncryptionKey() // Load from env var

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

func (u *User) decrypt(ciphertext string) (string, error) {
    key := u.getEncryptionKey()

    data, err := base64.StdEncoding.DecodeString(ciphertext)
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
    nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]

    plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
    if err != nil {
        return "", err
    }

    return string(plaintext), nil
}

func (u *User) getEncryptionKey() []byte {
    // Load 32-byte key from environment variable
    // Generate with: openssl rand -hex 32
    if u.encryptionKey == nil {
        u.encryptionKey = []byte(os.Getenv("ENCRYPTION_KEY"))
    }
    return u.encryptionKey
}

func (u *User) isEncrypted(data string) bool {
    // Check if already base64 encoded (simple heuristic)
    _, err := base64.StdEncoding.DecodeString(data)
    return err == nil
}
```

### Pattern 6: HTMX Login Form with Templ
**What:** Login page with "Login with Google" button using HTMX
**When to use:** Login page template
**Example:**
```templ
// Source: https://templ.guide/server-side-rendering/htmx/
package templates

templ LoginPage() {
    <!DOCTYPE html>
    <html>
    <head>
        <title>Login - First Sip</title>
        <script src="https://unpkg.com/htmx.org@2.0.0"></script>
        <link href="https://cdn.jsdelivr.net/npm/daisyui@4.12.0/dist/full.min.css" rel="stylesheet" />
    </head>
    <body>
        <div class="hero min-h-screen bg-base-200">
            <div class="hero-content text-center">
                <div class="max-w-md">
                    <h1 class="text-5xl font-bold">First Sip</h1>
                    <p class="py-6">Your daily briefing, personalized.</p>
                    <a href="/auth/google" class="btn btn-primary">
                        Login with Google
                    </a>
                </div>
            </div>
        </div>
    </body>
    </html>
}
```

### Anti-Patterns to Avoid
- **Storing tokens in plaintext:** Always encrypt OAuth tokens in database (use GORM hooks)
- **Using gin-gonic/contrib/sessions:** Deprecated since 2016, use gin-contrib/sessions instead
- **Cookie-only sessions in production:** Sessions lost on restart, use Redis backend
- **Forgetting Secure flag on cookies:** Must be true in production with HTTPS
- **Not handling HTMX redirects:** Standard redirects fail with HTMX, use HX-Redirect header
- **Hardcoding provider in route handler:** Pass provider as route parameter for multi-provider support
- **Skipping CSRF protection:** Goth handles this via state parameter, don't disable it
- **Not configuring session MaxAge:** Default may be too short or too long for your use case

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| OAuth 2.0 flow | Custom OAuth implementation | Goth + Gothic | OAuth has subtle security requirements (state parameter, token refresh, provider quirks). Goth handles 70+ providers correctly. "It is rarely a good idea to implement your own Auth solution!" |
| Session management | Custom cookie handling | gorilla/sessions + gin-contrib/sessions | Session security is complex (CSRF, fixation attacks, expiry). gorilla/sessions is battle-tested since 2012. |
| Token encryption | Custom crypto logic | stdlib crypto/aes + crypto/cipher with GCM mode | AES implementation has timing attacks, padding oracles, and nonce reuse vulnerabilities. stdlib is audited and correct. |
| Session persistence | In-memory session store | Redis via gin-contrib/sessions/redis | In-memory sessions lost on restart. Redis provides TTL, replication, and works with horizontal scaling. |
| Protected route middleware | Custom auth checks in each handler | Gin middleware with c.Abort() | Easy to forget auth checks, middleware enforces consistently across all protected routes. |

**Key insight:** Authentication has catastrophic failure modes (account takeover, token theft, session hijacking). Use mature, audited libraries. The time saved building custom solutions is dwarfed by the time spent fixing security vulnerabilities.

## Common Pitfalls

### Pitfall 1: Session Store Not Configured Before Goth Routes
**What goes wrong:** Goth's `gothic.CompleteUserAuth()` returns "session not found" error after OAuth callback
**Why it happens:** Goth/gothic depends on gorilla/sessions being available in the request context, which requires session middleware to run first
**How to avoid:** Register session middleware globally before defining auth routes: `r.Use(sessions.Sessions("name", store))` must come before `r.GET("/auth/google/callback", ...)`
**Warning signs:** "could not find a matching session for this request" error during OAuth callback

### Pitfall 2: Callback URL Mismatch with Google Console
**What goes wrong:** Google OAuth returns "redirect_uri_mismatch" error
**Why it happens:** The callback URL registered in Google Console must exactly match the URL passed to `google.New()`. Common issues: http vs https, localhost vs 127.0.0.1, missing trailing slash, wrong port
**How to avoid:** Use environment variables for callback URL and verify Google Console settings match exactly, including protocol and port
**Warning signs:** 400 error from Google during OAuth flow with message about redirect URI

### Pitfall 3: Secure Flag on Cookies in Development
**What goes wrong:** Sessions don't work in local development (HTTP, not HTTPS)
**Why it happens:** Setting `Secure: true` on session cookies prevents them from being sent over HTTP connections
**How to avoid:** Use environment-based configuration: `Secure: os.Getenv("ENV") == "production"`, false for local dev
**Warning signs:** Session values don't persist across requests in local development

### Pitfall 4: HTMX Redirects Show Fragment Instead of Full Page
**What goes wrong:** After session timeout, user clicks button and login form appears in middle of page instead of redirecting to login page
**Why it happens:** HTMX intercepts redirects and swaps content into target element. Standard 302/303 redirects don't trigger full page navigation
**How to avoid:** Check for `HX-Request` header in auth middleware, respond with `HX-Redirect` header and 401 status instead of standard redirect
**Warning signs:** Login form appears as fragment in dashboard after session expires

### Pitfall 5: Encryption Key Change Breaks Existing Sessions
**What goes wrong:** After changing ENCRYPTION_KEY, all users are logged out and can't log back in
**Why it happens:** Encrypted tokens in database can't be decrypted with new key, GORM hooks fail during AfterFind
**How to avoid:** Implement key rotation: store key version with encrypted data, support decrypting with old keys, re-encrypt with new key on next save
**Warning signs:** All authentication suddenly fails after deployment, database tokens become unreadable

### Pitfall 6: BeforeSave Hook Returns Error, Rolls Back Transaction
**What goes wrong:** User creation fails silently, no database record created
**Why it happens:** GORM runs save operations in transactions by default. Returning error from BeforeSave hook (e.g., encryption failure) rolls back entire transaction including user creation
**How to avoid:** Log errors from hooks, ensure encryption always succeeds (validate key exists, handle missing tokens gracefully)
**Warning signs:** Users complete OAuth but aren't created in database, no error shown to user

### Pitfall 7: Goth Version Pinning Not Enforced
**What goes wrong:** Dependency update breaks OAuth flow with obscure errors
**Why it happens:** Goth has breaking changes between minor versions, automatic updates via `go get -u` can break authentication
**How to avoid:** Pin Goth version explicitly in go.mod: `require github.com/markbates/goth v1.82.0`, review release notes before updating
**Warning signs:** Authentication stops working after `go get -u` or dependency update

### Pitfall 8: Session Secret Key Too Weak or Hardcoded
**What goes wrong:** Session hijacking, attacker can forge valid session cookies
**Why it happens:** Weak keys can be brute-forced, hardcoded keys in source control are public
**How to avoid:** Generate strong random key: `openssl rand -hex 32`, store in environment variable, rotate periodically
**Warning signs:** Security audit flags weak session secret, session cookies easily forged

## Code Examples

Verified patterns from official sources:

### Complete Authentication Setup
```go
// Source: Multiple verified sources combined
package main

import (
    "github.com/gin-contrib/sessions"
    "github.com/gin-contrib/sessions/redis"
    "github.com/gin-gonic/gin"
    "github.com/markbates/goth"
    "github.com/markbates/goth/gothic"
    "github.com/markbates/goth/providers/google"
    "os"
)

func main() {
    // Initialize Goth providers
    goth.UseProviders(
        google.New(
            os.Getenv("GOOGLE_CLIENT_ID"),
            os.Getenv("GOOGLE_CLIENT_SECRET"),
            os.Getenv("GOOGLE_CALLBACK_URL"),
            "email", "profile",
        ),
    )

    // Setup Gin router
    r := gin.Default()

    // Redis session store (before routes!)
    store, err := redis.NewStore(10, "tcp", "localhost:6379", "", []byte(os.Getenv("SESSION_SECRET")))
    if err != nil {
        panic(err)
    }
    store.Options(sessions.Options{
        Path:     "/",
        MaxAge:   86400 * 30,
        HttpOnly: true,
        Secure:   os.Getenv("ENV") == "production",
        SameSite: http.SameSiteStrictMode,
    })
    r.Use(sessions.Sessions("auth_session", store))

    // Public routes
    r.GET("/login", renderLoginPage)
    r.GET("/auth/google", handleGoogleLogin)
    r.GET("/auth/google/callback", handleGoogleCallback)

    // Protected routes
    protected := r.Group("/")
    protected.Use(authRequired())
    {
        protected.GET("/dashboard", renderDashboard)
        protected.GET("/logout", handleLogout)
    }

    r.Run(":8080")
}
```

### HTMX-Compatible Auth Middleware
```go
// Source: https://htmx.org/essays/web-security-basics-with-htmx/
func authRequired() gin.HandlerFunc {
    return func(c *gin.Context) {
        session := sessions.Default(c)
        userID := session.Get("user_id")

        if userID == nil {
            if c.GetHeader("HX-Request") == "true" {
                // HTMX request - tell client to redirect entire page
                c.Header("HX-Redirect", "/login")
                c.AbortWithStatus(401)
            } else {
                // Standard request - normal redirect
                c.Redirect(302, "/login")
                c.Abort()
            }
            return
        }

        c.Set("user_id", userID)
        c.Next()
    }
}
```

### AES-256-GCM Encryption for GORM
```go
// Source: https://www.twilio.com/en-us/blog/developers/community/encrypt-and-decrypt-data-in-go-with-aes-256
import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "io"
)

func encryptAESGCM(plaintext string, key []byte) ([]byte, error) {
    block, err := aes.NewCipher(key) // key must be 32 bytes for AES-256
    if err != nil {
        return nil, err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }

    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return nil, err
    }

    // Seal prepends nonce to ciphertext
    ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
    return ciphertext, nil
}

func decryptAESGCM(ciphertext []byte, key []byte) (string, error) {
    block, err := aes.NewCipher(key)
    if err != nil {
        return "", err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }

    nonceSize := gcm.NonceSize()
    if len(ciphertext) < nonceSize {
        return "", errors.New("ciphertext too short")
    }

    nonce, ciphertextBytes := ciphertext[:nonceSize], ciphertext[nonceSize:]
    plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
    if err != nil {
        return "", err
    }

    return string(plaintext), nil
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| gin-gonic/contrib/sessions | gin-contrib/sessions | Deprecated 2016 | Old package unmaintained, new package required for Gin + gorilla/sessions |
| JWT for session management | Server-side sessions with Redis | Ongoing shift | JWTs can't be revoked, sessions provide better control, HTMX apps favor server state |
| Cookie-only session storage | Redis backend sessions | Best practice since ~2018 | Cookie sessions lost on restart, Redis persists across deploys |
| HTMX v1 | HTMX v2 | Released 2024 | v2 changes: hx-boost requires "true" value, WebSocket syntax changed to hx-ws |
| Goth <v1.80 | Goth v1.82.0 | Released Aug 2025 | Ongoing provider updates, Google OAuth changes tracked |
| AES-CBC encryption | AES-GCM encryption | Best practice since ~2015 | GCM provides authentication tag preventing tampering attacks |

**Deprecated/outdated:**
- **gin-gonic/contrib/sessions**: Deprecated 2016, use gin-contrib/sessions (different import path)
- **gorilla/mux**: Maintenance mode 2024, Gin or stdlib net/http preferred for new projects
- **HTMX v1 syntax**: hx-boost without value, old WebSocket attributes deprecated in v2

## Open Questions

1. **Google OAuth Token Refresh Strategy**
   - What we know: Goth returns access tokens and refresh tokens from OAuth callback
   - What's unclear: When/how to refresh expired access tokens for background n8n calls
   - Recommendation: Store refresh tokens, implement token refresh before n8n webhook calls, test token expiry flow

2. **Multi-User Session Isolation with Single Redis Instance**
   - What we know: Redis backend stores sessions with unique keys
   - What's unclear: Session key collision risks, optimal key prefixing strategy
   - Recommendation: Review gin-contrib/sessions/redis key generation, ensure user_id included in session key

3. **Encryption Key Rotation Impact on Active Sessions**
   - What we know: Changing ENCRYPTION_KEY breaks decryption of existing tokens
   - What's unclear: Safe key rotation procedure without logging out all users
   - Recommendation: Implement key versioning (store version byte with encrypted data), support multiple keys during transition

4. **HTMX Redirect with OAuth Callback**
   - What we know: OAuth callback uses standard HTTP redirect to dashboard
   - What's unclear: Whether HTMX interferes with OAuth callback redirects (likely not, as callback is full page navigation)
   - Recommendation: Test OAuth flow, ensure callback uses standard redirect not HTMX swap

5. **GORM Hook Transaction Behavior with Encryption Failures**
   - What we know: BeforeSave errors roll back entire transaction
   - What's unclear: Best error handling strategy (fail loudly vs. graceful degradation)
   - Recommendation: Fail loudly during user creation (don't create user without encrypted tokens), log encryption errors for debugging

## Sources

### Primary (HIGH confidence)
- [Goth GitHub Repository](https://github.com/markbates/goth) - Official documentation, v1.82.0 release notes, provider setup
- [gin-contrib/sessions Package Docs](https://pkg.go.dev/github.com/gin-contrib/sessions) - Official API, Redis integration, session options
- [GORM Hooks Documentation](https://gorm.io/docs/hooks.html) - BeforeSave/AfterFind lifecycle, transaction behavior
- [Go crypto/cipher Package](https://pkg.go.dev/crypto/cipher) - AES-GCM implementation, official Go cryptography

### Secondary (MEDIUM confidence)
- [Twilio: Encrypt and Decrypt Data in Go with AES-256](https://www.twilio.com/en-us/blog/developers/community/encrypt-and-decrypt-data-in-go-with-aes-256) - Complete AES-GCM example with best practices
- [HTMX Web Security Basics](https://htmx.org/essays/web-security-basics-with-htmx/) - Cookie-based auth, HX-Redirect pattern
- [OAuth with Gin and Goth](https://dizzy.zone/2018/06/01/OAuth-with-Gin-and-Goth/) - Integration patterns, gothic helpers, common gotchas
- [Handling Django Authentication Redirects in HTMX Applications](https://saurabh-kumar.com/articles/2025/05/handling-django-authentication-redirects-in-htmx-applications/) - HX-Redirect pattern (Django but applicable)
- [How to Manage Sessions in Golang using Gin Framework And Redis](https://articles.wesionary.team/session-management-in-golang-gin-framework-using-redis-with-e-1f17b6980924) - Redis session backend configuration
- [Leapcell: Secure Your APIs with JWT Authentication in Gin Middleware](https://leapcell.io/blog/secure-your-apis-with-jwt-authentication-in-gin-middleware) - Middleware patterns, c.Abort() usage
- [Building user authentication and authorisation API in Go using Gin and Gorm](https://ututuv.medium.com/building-user-authentication-and-authorisation-api-in-go-using-gin-and-gorm-93dfe38e0612) - Complete auth flow example
- [Templ Guide: HTMX Integration](https://templ.guide/server-side-rendering/htmx/) - Templ+HTMX patterns

### Tertiary (LOW confidence)
- Various Medium articles and blog posts on Go authentication - General patterns, need verification with official docs
- GitHub Gist examples of AES encryption - Code snippets, not official documentation
- Reddit r/golang discussions on Goth usage - Community opinions, not authoritative

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Goth v1.82.0 verified from GitHub, gin-contrib/sessions official package, AES-GCM stdlib
- Architecture: MEDIUM - Patterns verified across multiple sources, some specifics (key rotation, hook transaction behavior) need testing
- Pitfalls: MEDIUM - Documented in official sources (session middleware ordering, callback URL matching), some from community experience

**Research date:** 2026-02-10
**Valid until:** ~2026-03-15 (30 days) - Authentication libraries relatively stable, but Goth has frequent provider updates. Verify Goth version before implementation.

**Critical verification before implementation:**
- Run `go list -m -versions github.com/markbates/goth` to check for newer versions since v1.82.0
- Review [Goth CHANGELOG](https://github.com/markbates/goth/releases) for breaking changes
- Test Redis session persistence across server restarts
- Verify HTMX v2 HX-Redirect behavior with Gin redirects
- Test GORM encryption hooks don't cause silent failures during user creation
