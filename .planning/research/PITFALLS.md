# Pitfalls Research

**Domain:** Go web application with Templ + HTMX + GORM + Asynq + OAuth
**Researched:** 2026-02-10
**Overall Confidence:** MEDIUM (based on training data without current web verification)

## Critical Pitfalls

### Pitfall 1: GORM Hooks Causing N+1 Queries

**What goes wrong:**
Encryption/decryption hooks (AfterFind, BeforeCreate) trigger for every row in a query result. With `db.Find(&tokens)` returning 100 records, each row's AfterFind hook fires individually. If hooks do ANY database queries, you get exponential query explosion.

**Why it happens:**
GORM hooks run in-process per-record. Developers add "just one query" to a hook without realizing it multiplies by result set size. Encryption seems like a model concern, so hooks feel natural.

**How to avoid:**
- Never query the database inside GORM hooks
- Keep hooks to pure computation only (encrypt/decrypt bytes)
- Use `db.Session(&gorm.Session{SkipHooks: true})` for bulk operations
- Monitor query count in development (add middleware that panics if >5 queries per request)
- Consider explicit service-layer encryption instead of hooks for complex cases

**Warning signs:**
- Slow queries that don't show in database logs (computation time)
- Query count scales with result set size
- Performance degrades non-linearly with data growth
- Database connection pool exhaustion under moderate load

**Phase to address:**
Bootstrap phase - Set up query monitoring middleware immediately. Document hook limitations in architecture decisions.

**Confidence:** HIGH (well-documented GORM behavior)

---

### Pitfall 2: HTMX Polling Without Backoff Creates Thundering Herd

**What goes wrong:**
`hx-trigger="every 2s"` on job status endpoints means every user polls every 2 seconds. 100 concurrent users = 50 requests/second doing database queries. No built-in backoff when jobs complete. Polls continue even after success.

**Why it happens:**
HTMX makes polling trivial (`every Ns`), but doesn't handle completion/backoff. Developers forget to stop polling after terminal states. Each poll hits your app, database, and potentially external services.

**How to avoid:**
- Return `HX-Trigger: {"stopPolling": true}` header when job completes
- Use exponential backoff: start at 1s, increase to 2s, 5s, 10s
- Implement client-side using `hx-trigger="load delay:1s"` pattern with server-controlled delay
- Add cache layer (Redis) for job status - most polls should hit cache
- Consider WebSockets/SSE for real-time updates instead of polling

**Example pattern:**
```go
// Return dynamic delay header
if job.Status == "pending" && job.Attempts < 3 {
    w.Header().Set("HX-Trigger-After-Settle", `{"poll": {"delay": "1000"}}`)
} else if job.Status == "pending" {
    w.Header().Set("HX-Trigger-After-Settle", `{"poll": {"delay": "5000"}}`)
} else {
    // Terminal state - stop polling
    w.Header().Set("HX-Trigger", "stopPolling")
}
```

**Warning signs:**
- Database CPU spikes with many concurrent users
- Polls continue after job completion
- Linear scaling breaks at low user counts (<100)
- Redis/cache hit rate is low for status endpoints

**Phase to address:**
Bootstrap phase - Implement polling backoff in initial HTMX integration. Add Redis caching before performance testing.

**Confidence:** HIGH (common HTMX pattern, documented extensively)

---

### Pitfall 3: Asynq Task Serialization Breaks with Sensitive Data

**What goes wrong:**
Asynq serializes task payloads to JSON and stores in Redis. Passing OAuth tokens, encryption keys, or PII directly in task payloads exposes sensitive data in Redis. Redis persistence means data lives beyond job execution.

**Why it happens:**
Convenience - easiest to pass `Task{UserID: 1, Token: "secret"}` directly. Developers treat task queues like function calls. Not obvious that payloads are persisted.

**How to avoid:**
- Only pass identifiers in task payloads (user ID, job ID)
- Fetch sensitive data from database inside task handler
- Enable Redis encryption at rest
- Set task retention policies (`asynq.Retention(24*time.Hour)`)
- Use Redis ACLs to limit queue access
- Never log full task payloads

**Example:**
```go
// BAD
type EmailTask struct {
    Email string
    OAuthToken string // Exposed in Redis
}

// GOOD
type EmailTask struct {
    UserID int // Fetch token in handler
}

func HandleEmail(ctx context.Context, t *asynq.Task) error {
    var p EmailTask
    json.Unmarshal(t.Payload(), &p)

    // Fetch token securely
    token, err := db.GetEncryptedToken(p.UserID)
    // ...
}
```

**Warning signs:**
- Task payloads contain tokens, keys, or PII
- Redis memory grows unbounded
- Compliance audit flags Redis data
- Failed task logs expose sensitive data

**Phase to address:**
Bootstrap phase - Establish task payload patterns before first worker. Add pre-commit hook to grep for common sensitive field names in task structs.

**Confidence:** HIGH (standard queue security practice)

---

### Pitfall 4: Templ Component Boundaries Cause Template Duplication

**What goes wrong:**
HTMX swaps fragments, but Templ components need full HTML context. Developers duplicate layout code in each endpoint's template to avoid "partial layout" bugs. Over time, header/nav/footer logic diverges across handlers.

**Why it happens:**
Initial approach: each HTMX endpoint returns full page fragment with layout. Refactoring to extract layout feels risky mid-development. No clear "partial vs full" template pattern.

**How to avoid:**
- Establish layout pattern early: `layout(header, content, footer)`
- Use `HX-Request` header to detect HTMX requests
- Return fragments for HTMX, full pages for direct access
- Create Templ helper: `RenderWithLayout(c, content, isHXRequest)`

**Example:**
```go
func HandleDashboard(c *gin.Context) {
    content := components.Dashboard(data)

    if c.GetHeader("HX-Request") == "true" {
        // HTMX swap - just content
        content.Render(c.Request.Context(), c.Writer)
    } else {
        // Full page load - with layout
        layout.Base(content).Render(c.Request.Context(), c.Writer)
    }
}
```

**Warning signs:**
- Same nav/header code in 5+ template files
- Layout changes require multi-file edits
- Inconsistent styling between pages
- HTMX swaps break layout (missing CSS/JS)

**Phase to address:**
Bootstrap phase - Create layout abstraction before second page. Document pattern in ADR.

**Confidence:** MEDIUM (Templ-specific pattern, limited production data)

---

### Pitfall 5: OAuth Token Refresh Race Conditions

**What goes wrong:**
User opens 3 tabs. Each tab makes API call. All 3 detect expired token simultaneously. All 3 trigger refresh. OAuth provider sees 3 refresh requests, invalidates token after first, remaining 2 fail. User logged out.

**Why it happens:**
No distributed lock on token refresh. Each request independently checks expiry and refreshes. Goth doesn't handle concurrency. Race window is small but hits production.

**How to avoid:**
- Use Redis lock with `SET NX EX` for token refresh
- Check token freshness after acquiring lock (another process may have refreshed)
- Use single-flight pattern (Go's `singleflight` package)
- Add jitter to expiry checks (refresh at 80-95% of lifetime randomly)
- Implement token refresh queue (only one worker can refresh)

**Example:**
```go
import "golang.org/x/sync/singleflight"

var refreshGroup singleflight.Group

func RefreshToken(userID int) (*Token, error) {
    key := fmt.Sprintf("refresh-%d", userID)

    val, err, _ := refreshGroup.Do(key, func() (interface{}, error) {
        // Check if still needed (another goroutine may have refreshed)
        token := db.GetToken(userID)
        if token.ExpiresAt.After(time.Now().Add(5*time.Minute)) {
            return token, nil // Fresh enough
        }

        // Actually refresh
        return oauth.Refresh(token.RefreshToken)
    })

    return val.(*Token), err
}
```

**Warning signs:**
- Random "token invalid" errors under load
- Logs show multiple refresh attempts for same user
- OAuth provider rate limits refresh endpoint
- Users report being logged out intermittently

**Phase to address:**
Authentication phase - Implement before enabling concurrent users. Critical for production readiness.

**Confidence:** HIGH (common OAuth pattern)

---

### Pitfall 6: Missing Database Transaction Rollback on HTMX Errors

**What goes wrong:**
Handler starts transaction, renders HTMX response, hits template error. Response sends 200 OK with partial HTML, transaction commits. User sees error UI but data is saved. Retry attempts fail with duplicate key errors.

**Why it happens:**
Templ `Render()` can error after HTTP headers sent. Developers check `err` but can't change status code. Transaction already committed in defer/middleware.

**How to avoid:**
- Render to buffer first, THEN write to response
- Commit transactions only after successful render
- Use middleware pattern: `tx.Rollback()` in defer unless explicitly committed
- Add integration tests that force template errors

**Example:**
```go
func HandleCreate(c *gin.Context) {
    tx := db.Begin()
    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
        }
    }()

    // Do database work
    item := CreateItem(tx, data)

    // Render to buffer
    var buf bytes.Buffer
    if err := components.Success(item).Render(c.Request.Context(), &buf); err != nil {
        tx.Rollback()
        c.String(500, "Render error")
        return
    }

    // Commit only after successful render
    tx.Commit()
    c.Data(200, "text/html", buf.Bytes())
}
```

**Warning signs:**
- Duplicate key errors on retry
- Partial data in database
- Error UI but database shows "success"
- Transaction logs don't match error logs

**Phase to address:**
Bootstrap phase - Establish transaction pattern before first write operation.

**Confidence:** MEDIUM (Templ-specific, general transaction pattern is HIGH)

---

### Pitfall 7: Asynq Worker Panics Lose Tasks Without Retry

**What goes wrong:**
Worker panics (nil pointer, type assertion). Asynq catches panic, marks task as failed. Default retry config may exhaust quickly (3 retries). Task lost. No alert. Silent failure.

**Why it happens:**
Go panics aren't exceptions. Asynq recovers but doesn't distinguish panic from error. Developers don't configure retry/dead queue monitoring. "It works in dev" (low volume, no edge cases).

**How to avoid:**
- Configure generous retry policy: `asynq.MaxRetry(10)` with exponential backoff
- Monitor dead letter queue size (alert if >0)
- Wrap handlers with panic recovery that logs stack traces
- Send panics to error tracking (Sentry/Bugsnag)
- Use `asynq.ErrorHandler` to distinguish panic from business error
- Set up dead queue processor for manual intervention

**Example:**
```go
srv := asynq.NewServer(
    asynq.RedisClientOpt{Addr: redisAddr},
    asynq.Config{
        Concurrency: 10,
        ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
            // Log to error tracking
            if isPanic(err) {
                sentry.CaptureException(err)
                // Alert on-call
            }
        }),
        RetryDelayFunc: func(n int, err error, task *asynq.Task) time.Duration {
            // Exponential backoff: 1m, 2m, 4m, 8m, ...
            return time.Duration(1<<uint(n)) * time.Minute
        },
    },
)
```

**Warning signs:**
- Jobs mysteriously don't complete
- No logs for some task executions
- Dead queue grows
- "It worked yesterday" without code changes

**Phase to address:**
Worker implementation phase - Configure before first production worker deployment.

**Confidence:** HIGH (standard async job queue pattern)

---

### Pitfall 8: HTMX Out-of-Band Swaps Accumulate Without Cleanup

**What goes wrong:**
Use `hx-swap-oob="true"` to update notifications/counters. Each response adds to DOM. After 50 status checks, 50 notification divs exist (all but one hidden). Memory leak. Browser slows. Inspector shows 1000s of duplicate IDs.

**Why it happens:**
OOB swaps default to `innerHTML` which appends. Developers use OOB for "fire and forget" updates. Don't test long-running sessions.

**How to avoid:**
- Always use `hx-swap-oob="outerHTML"` (replaces, not appends)
- Use specific ID selectors: `id="notification-123"` not `id="notification"`
- Add cleanup interval: `setInterval(() => cleanupOldNotifications(), 60000)`
- Limit OOB updates per response (max 3)
- Test with 100+ sequential updates

**Example:**
```templ
// BAD - accumulates
<div id="notification" hx-swap-oob="true">New message</div>

// GOOD - replaces
<div id="notification" hx-swap-oob="outerHTML">New message</div>

// BETTER - unique IDs
<div id="notification-{timestamp}" hx-swap-oob="beforeend:#notification-container">
    <span class="notification">New message</span>
</div>
```

**Warning signs:**
- Browser DevTools shows duplicate IDs
- Page gets slower after extended use
- Memory usage grows linearly with requests
- Multiple elements respond to same ID selector

**Phase to address:**
Bootstrap phase - Establish OOB pattern before notifications/polling.

**Confidence:** MEDIUM (HTMX-specific behavior)

---

### Pitfall 9: GORM Preloading Doesn't Work with `Select()` on Joined Fields

**What goes wrong:**
```go
db.Preload("Tokens").Select("id, email").Find(&users)
// Tokens is nil - Preload silently ignored
```
GORM can't preload associations when using `Select()` that excludes foreign keys. No error. No warning. Looks like data problem.

**Why it happens:**
`Select()` limits SQL columns. Preload needs foreign keys. GORM doesn't validate compatibility. Common when optimizing queries.

**How to avoid:**
- Always include foreign key columns in `Select()`: `Select("id, email, token_id")`
- Use `Omit()` instead of `Select()` when excluding few fields
- Test with empty database (preload failures often hidden by cached data)
- Enable GORM logger in dev: `db.Debug()` to see actual queries

**Example:**
```go
// BAD - Tokens won't load
db.Preload("Tokens").Select("id, email").Find(&users)

// GOOD - Include foreign key
db.Preload("Tokens").Select("id, email, token_id").Find(&users)

// BETTER - Use Omit for large structs
db.Preload("Tokens").Omit("large_binary_field").Find(&users)
```

**Warning signs:**
- Associations unexpectedly nil
- "Working" code breaks after "optimization"
- Different results between `Find` and `First`
- N+1 queries suddenly appear

**Phase to address:**
Database integration phase - Add linter rule, document in GORM patterns.

**Confidence:** HIGH (documented GORM behavior)

---

### Pitfall 10: Gin Context Not Propagated to Goroutines

**What goes wrong:**
```go
func Handler(c *gin.Context) {
    go func() {
        c.JSON(200, data) // Panic or wrong response
    }()
}
```
Gin's context is request-scoped. Goroutine outlives request. Context methods panic. Common with background tasks triggered by requests.

**Why it happens:**
Go's goroutines are easy. Developers treat Gin context like Go's context. Asynq integration tempts inline task enqueuing.

**How to avoid:**
- Never use `*gin.Context` in goroutines
- Extract data before goroutine: `userID := c.GetInt("user_id")`
- Use `c.Request.Context()` for cancellation
- Enqueue Asynq tasks, don't run inline
- Add linter: `go vet` with custom check

**Example:**
```go
// BAD
func HandleWebhook(c *gin.Context) {
    go processWebhook(c) // Context invalid
}

// GOOD
func HandleWebhook(c *gin.Context) {
    var payload WebhookPayload
    c.BindJSON(&payload)

    userID := c.GetInt("user_id")
    ctx := c.Request.Context()

    go func(ctx context.Context, uid int, p WebhookPayload) {
        // Use ctx for cancellation, uid for data
        processWebhook(ctx, uid, p)
    }(ctx, userID, payload)

    c.JSON(202, gin.H{"status": "accepted"})
}

// BEST - Use Asynq
func HandleWebhook(c *gin.Context) {
    var payload WebhookPayload
    c.BindJSON(&payload)

    task, _ := tasks.NewWebhookTask(payload)
    asynqClient.Enqueue(task)

    c.JSON(202, gin.H{"status": "queued"})
}
```

**Warning signs:**
- Random panics under load
- Context deadline exceeded errors
- Multiple responses for single request
- Race detector warnings

**Phase to address:**
Bootstrap phase - Enforce before first async operation.

**Confidence:** HIGH (documented Gin behavior)

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Skip transaction boundaries | Faster development | Data inconsistency, hard to debug | Never - always use transactions for writes |
| Direct SQL in handlers | Bypass GORM complexity | No type safety, SQL injection risk, hard to test | Complex queries GORM can't express - use sqlx |
| Inline task processing (no Asynq) | Simpler code | Timeouts, no retry, blocks requests | Truly fast operations (<100ms) with no failure scenarios |
| Store full OAuth token in cookie | Avoid database lookup | Security risk, size limits | Never - always use session ID |
| HTMX without fallback URLs | Less code | Breaks without JS, bad SEO | Internal dashboards with JS guarantee |
| Skip GORM migrations | Faster schema changes | No rollback, environment drift | Local dev only - never staging/prod |
| Cache without TTL | Simpler logic | Stale data, memory leak | Immutable data (user IDs, config loaded once) |

---

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| **GORM + HTMX** | Return JSON from endpoints | Use Templ templates, return HTML fragments |
| **Asynq + OAuth** | Pass tokens in task payload | Pass user ID, fetch token in handler |
| **Templ + Gin** | Call `c.JSON()` after `Render()` | Choose one response method - Templ OR JSON |
| **HTMX + Gin middleware** | CORS blocks `HX-Request` header | Add `HX-*` headers to CORS allowed headers |
| **GORM + Docker** | Use SQLite in dev, Postgres in prod | Match database engines (use Postgres in dev container) |
| **Asynq + Docker Compose** | Share Redis for cache + queue | Separate Redis instances (different eviction policies) |
| **Tailwind + Templ** | Classes don't apply | Run Tailwind build watching `.templ` files |
| **DaisyUI + HTMX** | JS-based components break | Use CSS-only DaisyUI components or reinitialize after swap |

---

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| GORM N+1 queries | Slow with more data | Use `Preload()`, monitor query count | >100 records |
| HTMX polling storm | CPU spikes, connection pool exhaustion | Implement backoff, use cache | >50 concurrent users |
| Asynq task backlog | Growing Redis memory, delayed jobs | Tune worker concurrency, add workers | >1000 jobs/hour |
| Templ compilation on request | First request slow after deploy | Pre-compile templates at build | Every deploy |
| OAuth token encryption overhead | Slow auth checks | Cache decrypted tokens in Redis (short TTL) | >100 requests/second |
| Missing database indexes | All queries slow | Index foreign keys and WHERE clauses | >10K rows |
| HTMX responses without streaming | Memory spikes on large pages | Use `Transfer-Encoding: chunked`, stream templates | Pages >100KB |

---

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| **OAuth state not validated** | CSRF on login flow | Always verify state parameter matches session |
| **Encrypted tokens use weak cipher** | Token compromise | Use AES-256-GCM, rotate keys, use GORM hook carefully |
| **HTMX endpoints skip CSRF** | CSRF attacks on mutations | Treat HTMX POST/DELETE like forms - require CSRF token |
| **Asynq tasks logged with payloads** | Sensitive data in logs | Redact payloads in logs, only log task type + ID |
| **Templ renders user input unescaped** | XSS attacks | Use Templ's built-in escaping, never `template.HTML()` user data |
| **Database credentials in environment** | Exposed in process list, logs | Use secret management (Kubernetes secrets, Vault) |
| **Redis has no password** | Unauthorized queue access | Enable Redis AUTH, use TLS |
| **Session cookies without SameSite** | CSRF via cookie | Set `SameSite=Lax` minimum, `Secure=true` in production |

---

## "Looks Done But Isn't" Checklist

- **OAuth Integration:** Often missing token refresh handling — verify refresh flow works when access token expires
- **HTMX Forms:** Often missing validation feedback — verify error states render correctly without full page reload
- **Asynq Workers:** Often missing dead letter queue monitoring — verify failed tasks are tracked and alertable
- **GORM Migrations:** Often missing rollback migrations — verify `down` migrations work for last 3 changes
- **Templ Components:** Often missing error states — verify templates handle nil/empty data gracefully
- **Docker Build:** Often missing multi-stage optimization — verify production image is <100MB
- **Database Indexes:** Often missing composite indexes — verify `EXPLAIN` shows index usage for common queries
- **HTMX Polling:** Often missing stop condition — verify polling stops after success/max attempts
- **Gin Middleware:** Often wrong order — verify auth runs before rate limiting, logging runs first
- **OAuth Encryption:** Often missing key rotation — verify app survives encryption key change

---

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| GORM Hooks N+1 | Bootstrap | Query count middleware shows <5 queries/request |
| HTMX Polling Thundering Herd | Bootstrap | Load test with 50 users shows backoff working |
| Asynq Sensitive Data | Bootstrap | Grep codebase for OAuth/token in task structs (none) |
| Templ Layout Duplication | Bootstrap | Count layout template imports (should be 1 base) |
| OAuth Token Refresh Race | Authentication Phase | Concurrent login test with 10 tabs succeeds |
| Missing Transaction Rollback | Bootstrap | Integration test forces template error, verifies rollback |
| Asynq Panic Loses Tasks | Worker Phase | Dead queue dashboard shows retry policy working |
| HTMX OOB Accumulation | Bootstrap | 100 sequential updates don't leak DOM elements |
| GORM Preload + Select | Database Phase | Test suite includes preload assertions |
| Gin Context in Goroutines | Bootstrap | Race detector enabled in CI, no warnings |

---

## Sources

- **GORM Documentation:** https://gorm.io/docs/ (hooks, transactions, preloading)
- **HTMX Documentation:** https://htmx.org/docs/ (polling, OOB swaps)
- **Asynq Documentation:** https://github.com/hibiken/asynq/wiki (task serialization, retries)
- **Gin Documentation:** https://gin-gonic.com/docs/ (context handling, middleware)
- **Templ Documentation:** https://templ.guide/ (component patterns)
- **Training Data:** Go best practices, OAuth security patterns, database transaction management

**Confidence Assessment:**
- GORM pitfalls: HIGH (well-documented behavior)
- HTMX pitfalls: MEDIUM-HIGH (documented, some inference on performance)
- Asynq pitfalls: HIGH (standard queue patterns)
- Templ pitfalls: MEDIUM (newer tool, less production data)
- OAuth pitfalls: HIGH (standard security patterns)
- Integration patterns: MEDIUM (combination-specific, some inference)

**Limitations:**
- Unable to verify with current 2026 sources (WebSearch unavailable)
- Templ confidence lower due to less community maturity vs established tools
- Performance numbers are estimates based on typical scenarios
- Security recommendations based on general OAuth/web security principles

---

*Pitfalls research for: Go web application with Templ + HTMX + GORM + Asynq + OAuth*
*Researched: 2026-02-10*
*Confidence: MEDIUM overall (HIGH for established patterns, MEDIUM for tool-specific combinations)*
