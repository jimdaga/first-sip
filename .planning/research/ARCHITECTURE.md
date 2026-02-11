# Architecture Patterns

**Domain:** Go Web Application with Server-Side Rendering + Background Jobs
**Researched:** 2026-02-10
**Confidence:** MEDIUM-HIGH (standard patterns, training data through January 2025)

## Recommended Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                          User Browser                           │
│  ┌──────────────┐   ┌──────────────┐   ┌──────────────┐       │
│  │  HTMX        │   │  Tailwind    │   │  DaisyUI     │       │
│  │  (CDN)       │   │  CSS (CDN)   │   │  (CDN)       │       │
│  └──────────────┘   └──────────────┘   └──────────────┘       │
└───────────────────────────────┬─────────────────────────────────┘
                                │ HTTP (hx-get, hx-post)
┌───────────────────────────────▼─────────────────────────────────┐
│                       Gin HTTP Server                           │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  Middleware Stack                                          │ │
│  │  [Logging] → [Recovery] → [CORS] → [Sessions] → [Auth]   │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐           │
│  │   Auth      │  │  Briefing   │  │   Static    │           │
│  │  Handlers   │  │  Handlers   │  │   Assets    │           │
│  │  (Goth)     │  │  (Templ)    │  │             │           │
│  └─────────────┘  └─────────────┘  └─────────────┘           │
│         │                 │                                     │
│         ▼                 ▼                                     │
│  ┌─────────────┐  ┌─────────────┐                             │
│  │  Session    │  │  Briefing   │                             │
│  │  Service    │  │  Service    │◄──────┐                     │
│  └─────────────┘  └─────────────┘       │                     │
│         │                 │              │                     │
└─────────┼─────────────────┼──────────────┼─────────────────────┘
          │                 │              │
          ▼                 ▼              │
    ┌──────────┐      ┌──────────┐        │
    │  Redis   │      │ Postgres │        │
    │ (Sessions│      │ (GORM)   │        │
    │ + Queue) │      │          │        │
    └──────────┘      └──────────┘        │
          │                                │
          ▼                                │
┌─────────────────────────────────────────┼─────────────────────┐
│              Asynq Worker Process       │                     │
│  ┌────────────────────────────────────┐ │                     │
│  │  Task Handlers                     │ │                     │
│  │  - GenerateBriefingTask           │ │                     │
│  │  - SendEmailTask                  │ │                     │
│  │  - CleanupOldBriefingsTask        │ │                     │
│  └────────────────────────────────────┘ │                     │
│                 │                        │                     │
│                 ▼                        │                     │
│  ┌────────────────────────────────────┐ │                     │
│  │  n8n Client                        │ │                     │
│  │  - HTTP calls to n8n webhooks     │─┼─────────────────────┘
│  │  - Aggregate responses            │ │
│  └────────────────────────────────────┘ │
│                 │                        │
│                 ▼                        │
│         ┌──────────────┐                │
│         │  Postgres    │                │
│         │  (Store      │                │
│         │  Briefing)   │                │
│         └──────────────┘                │
└─────────────────────────────────────────┘
          │
          ▼
    ┌──────────────┐
    │     n8n      │
    │  Workflow    │
    │  Engine      │
    │  (External)  │
    └──────────────┘
```

## Component Boundaries

| Component | Responsibility | Communicates With | Notes |
|-----------|---------------|-------------------|-------|
| **Gin HTTP Server** | Route requests, middleware, render Templ | Browser, Redis (sessions), Postgres (GORM), Asynq (client) | Single process, stateless except sessions |
| **Templ Templates** | Type-safe HTML generation | Gin handlers (called by handlers) | Compile-time, generates Go code |
| **HTMX (Browser)** | Client-side HTTP requests, DOM updates | Gin server (AJAX) | CDN-hosted, no build step |
| **Goth Auth Service** | OAuth flow management | Google OAuth, Redis (sessions) | Middleware + handlers |
| **Asynq Client** | Enqueue background jobs | Redis (queue) | Embedded in Gin server process |
| **Asynq Worker** | Execute background jobs | Redis (queue), n8n (HTTP), Postgres (GORM) | Separate process, horizontally scalable |
| **GORM Models** | Database ORM | Postgres | Shared by server + worker |
| **Redis** | Session storage, job queue | Gin (sessions), Asynq (queue) | Single instance for MVP, cluster later |
| **Postgres** | Persistent data storage | GORM (server + worker) | Users, briefings, job metadata |
| **n8n** | External workflow orchestration | Asynq worker (HTTP webhooks) | Self-hosted or cloud, external to app |

## Data Flow Patterns

### Pattern 1: OAuth Authentication
```
User clicks "Login with Google"
  → Gin handler: /auth/google
  → Goth redirects to Google OAuth
  → User approves
  → Google redirects to /auth/google/callback
  → Goth validates, extracts user info
  → Store session in Redis (gorilla/sessions)
  → Redirect to /dashboard
  → Subsequent requests include session cookie
  → Middleware validates session from Redis
```

### Pattern 2: Briefing Generation
```
User clicks "Generate Briefing"
  → HTMX POST to /briefing/generate
  → Gin handler validates auth (session middleware)
  → Create Job metadata in Postgres (user_id, status=pending)
  → Enqueue Asynq task (GenerateBriefingTask)
  → Return 202 Accepted with hx-trigger to start polling
  → HTMX polls GET /briefing/status every 2 seconds

Meanwhile (async):
  → Asynq worker dequeues task
  → Call n8n webhook(s) with HTTP client
  → Aggregate responses (news, weather, etc.)
  → Parse JSON responses
  → Store briefing in Postgres (GORM)
  → Update job status to "complete"

Polling:
  → GET /briefing/status returns job status
  → When status=complete, hx-trigger swaps content
  → GET /briefing/latest renders Templ component
  → HTMX swaps HTML into DOM
```

### Pattern 3: Scheduled Generation
```
Asynq scheduler (configured at startup)
  → Enqueues GenerateBriefingTask at cron schedule (7am daily)
  → Worker executes same flow as manual generation
  → Stores briefing in Postgres
  → (Optional) Enqueues SendEmailTask if user has email enabled
```

## Patterns to Follow

### Pattern 1: Handler → Service → Repository
**What:** Separate concerns across layers
**When:** Always, for testability and clarity
**Example:**
```go
// Handler (HTTP layer)
func (h *BriefingHandler) Generate(c *gin.Context) {
    userID := c.GetString("user_id") // from auth middleware

    job, err := h.briefingService.EnqueueGeneration(userID)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    c.JSON(202, gin.H{"job_id": job.ID})
}

// Service (business logic)
func (s *BriefingService) EnqueueGeneration(userID string) (*Job, error) {
    job := s.jobRepo.Create(userID, "pending")

    task := asynq.NewTask("briefing:generate", []byte(job.ID))
    s.asynqClient.Enqueue(task)

    return job, nil
}

// Repository (data access)
func (r *JobRepository) Create(userID, status string) *Job {
    job := &Job{UserID: userID, Status: status}
    r.db.Create(job)
    return job
}
```

### Pattern 2: Templ Component Composition
**What:** Nested components for reusability
**When:** Layout + page + partials
**Example:**
```templ
// layout.templ
templ Layout(title string) {
    <!DOCTYPE html>
    <html>
    <head><title>{title}</title></head>
    <body>
        @Header()
        { children... }
        @Footer()
    </body>
    </html>
}

// briefing.templ
templ BriefingPage(briefing *Briefing) {
    @Layout("Your Briefing") {
        <div class="container">
            @BriefingCard(briefing)
        </div>
    }
}

// Render in handler
func (h *Handler) Show(c *gin.Context) {
    briefing := h.service.GetLatest(c.GetString("user_id"))
    c.Header("Content-Type", "text/html")
    BriefingPage(briefing).Render(c.Request.Context(), c.Writer)
}
```

### Pattern 3: HTMX Partial Rendering
**What:** Return HTML fragments, not full pages
**When:** HTMX requests (check HX-Request header)
**Example:**
```go
func (h *Handler) GetStatus(c *gin.Context) {
    jobID := c.Param("id")
    job := h.service.GetJob(jobID)

    if c.GetHeader("HX-Request") == "true" {
        // Return partial for HTMX swap
        StatusPartial(job).Render(c.Request.Context(), c.Writer)
    } else {
        // Return full page for direct access
        StatusPage(job).Render(c.Request.Context(), c.Writer)
    }
}
```

### Pattern 4: Asynq Task Serialization
**What:** JSON payloads for task data
**When:** Enqueueing background jobs
**Example:**
```go
type GenerateBriefingPayload struct {
    UserID    string `json:"user_id"`
    JobID     string `json:"job_id"`
    Sources   []string `json:"sources"`
}

// Enqueue
func Enqueue(client *asynq.Client, payload GenerateBriefingPayload) {
    data, _ := json.Marshal(payload)
    task := asynq.NewTask("briefing:generate", data)
    client.Enqueue(task)
}

// Handler
func HandleGenerateBriefing(ctx context.Context, t *asynq.Task) error {
    var payload GenerateBriefingPayload
    json.Unmarshal(t.Payload(), &payload)

    // Execute task...
}
```

## Anti-Patterns to Avoid

### Anti-Pattern 1: Business Logic in Handlers
**What:** Handlers contain database queries, external API calls, complex logic
**Why bad:** Untestable, violates SRP, hard to reuse
**Instead:** Handlers only route, validate, call services
```go
// BAD
func (h *Handler) Generate(c *gin.Context) {
    db.Where("user_id = ?", c.GetString("user_id")).First(&user)
    resp, _ := http.Get("https://n8n.example.com/webhook")
    // ... 50 lines of logic ...
}

// GOOD
func (h *Handler) Generate(c *gin.Context) {
    job, err := h.service.EnqueueGeneration(c.GetString("user_id"))
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    c.JSON(202, gin.H{"job_id": job.ID})
}
```

### Anti-Pattern 2: Shared Mutable State
**What:** Global variables, package-level state
**Why bad:** Race conditions, hard to test, breaks horizontal scaling
**Instead:** Dependency injection, immutable config
```go
// BAD
var db *gorm.DB // global

func Init() {
    db = connectDB()
}

// GOOD
type Service struct {
    db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
    return &Service{db: db}
}
```

### Anti-Pattern 3: GORM N+1 Queries
**What:** Lazy loading causes query per relation
**Why bad:** Kills performance, database load
**Instead:** Eager loading with Preload
```go
// BAD (N+1)
briefings := []Briefing{}
db.Find(&briefings) // 1 query
for _, b := range briefings {
    db.Model(&b).Association("Sections").Find(&b.Sections) // N queries
}

// GOOD (2 queries)
db.Preload("Sections").Find(&briefings)
```

### Anti-Pattern 4: Long-Running Requests
**What:** Call n8n webhooks directly in HTTP handler
**Why bad:** Request timeout, blocks server, no retry
**Instead:** Background job via Asynq
```go
// BAD
func (h *Handler) Generate(c *gin.Context) {
    resp, _ := http.Get("https://n8n.example.com/webhook") // blocks 30+ seconds
    c.JSON(200, resp)
}

// GOOD
func (h *Handler) Generate(c *gin.Context) {
    task := asynq.NewTask("briefing:generate", payload)
    h.asynqClient.Enqueue(task) // returns immediately
    c.JSON(202, gin.H{"status": "pending"})
}
```

## Scalability Considerations

| Concern | At 1 User | At 100 Users | At 10K Users |
|---------|-----------|--------------|--------------|
| **HTTP Server** | Single Gin process | Same, stateless scales horizontally | Multiple instances behind load balancer (k8s) |
| **Database** | Single Postgres instance | Connection pooling (max 25) | Read replicas for queries, write to primary |
| **Redis** | Single instance | Same, Redis handles 10K+ ops/sec | Redis Cluster or Sentinel for HA |
| **Asynq Workers** | 1 worker, 10 concurrency | 2-3 workers, 20 concurrency each | Worker pool, queue partitioning by priority |
| **Sessions** | Redis-backed, no issue | Same | Consider TTL cleanup, session size limits |
| **Static Assets** | Gin serves | Same | CDN (CloudFlare), nginx for static files |
| **n8n** | Single instance | Same | Multiple n8n instances, webhook routing |

## Deployment Architecture

### Development (Local)
```
Docker Compose:
- app (hot reload with Air)
- postgres
- redis
- n8n (optional, can use cloud)

Single binary: server + worker (combined)
```

### Production (Kubernetes)
```
Deployments:
- first-sip-server (Gin HTTP server, 2+ replicas)
- first-sip-worker (Asynq worker, 2+ replicas)

StatefulSets:
- postgres (or use managed RDS/Cloud SQL)
- redis (or use managed ElastiCache/Memorystore)

Services:
- first-sip-server (LoadBalancer/Ingress)
- postgres (ClusterIP)
- redis (ClusterIP)

ConfigMaps: env vars
Secrets: DATABASE_URL, REDIS_URL, GOOGLE_CLIENT_SECRET, SESSION_SECRET
```

## Security Boundaries

| Boundary | Protection | Implementation |
|----------|-----------|----------------|
| **External → Server** | TLS, rate limiting | Traefik/Ingress with cert-manager, Gin rate-limit middleware |
| **Server → Database** | TLS, credentials | Postgres TLS mode, secrets management |
| **Server → Redis** | TLS, password | Redis AUTH, TLS connection |
| **Worker → n8n** | API key, HTTPS | n8n webhook authentication, validate responses |
| **Session → User** | Secure cookies | HttpOnly, Secure, SameSite=Strict flags |
| **User → User** | Authorization middleware | Check userID from session matches resource owner |

## Observability

| Layer | Tool | Metrics |
|-------|------|---------|
| **HTTP** | Gin middleware + zerolog | Request duration, status codes, error rates |
| **Database** | GORM logger | Query duration, slow queries (>100ms) |
| **Jobs** | Asynqmon UI | Queue length, retry count, failure rate |
| **Redis** | Redis INFO | Memory usage, connection count |
| **Application** | zerolog JSON logs | Structured logs to stdout (k8s captures) |

## Testing Strategy by Layer

| Layer | Tool | What to Test |
|-------|------|--------------|
| **Handlers** | httptest + testify | HTTP status codes, header validation, auth checks |
| **Services** | testify mocks | Business logic, error handling, service contracts |
| **Repositories** | testcontainers-go | Database queries, GORM relations, migrations |
| **Workers** | asynqtest | Task execution, retry logic, idempotency |
| **Templ** | templ_test | HTML output, XSS prevention, attribute rendering |
| **Integration** | Docker Compose + go test | End-to-end flows (login → generate → view) |

## Directory Structure

```
/cmd/
  /server/main.go          - HTTP server entrypoint
  /worker/main.go          - Asynq worker entrypoint

/internal/
  /auth/
    handler.go             - Goth OAuth handlers
    middleware.go          - Session auth middleware
    service.go             - Auth business logic
  /briefing/
    handler.go             - Briefing HTTP handlers
    service.go             - Briefing business logic
    repository.go          - GORM database access
  /models/
    user.go                - GORM User model
    briefing.go            - GORM Briefing model
    job.go                 - GORM Job model
  /workers/
    briefing_task.go       - Asynq generate briefing handler
    email_task.go          - Asynq send email handler
  /templates/              - Templ components
    layout.templ
    briefing.templ
    auth.templ
  /middleware/
    logging.go
    recovery.go
    session.go
  /n8n/
    client.go              - n8n webhook HTTP client

/pkg/
  /config/
    config.go              - Load env vars, validate
  /logger/
    logger.go              - zerolog setup

/migrations/               - SQL migration files
/static/                   - CSS, JS, images (minimal)
/deployments/              - Helm charts, k8s manifests (already exist)
```

## Configuration Management

| Environment | Source | Notes |
|-------------|--------|-------|
| **Development** | .env file (godotenv) | Not committed to git |
| **Testing** | .env.test | Overrides for test database |
| **Production** | Kubernetes ConfigMap + Secrets | Mounted as env vars |

**Required Config:**
```bash
# Server
PORT=8080
GIN_MODE=release

# Database
DATABASE_URL=postgres://...

# Redis
REDIS_URL=redis://...

# OAuth
GOOGLE_CLIENT_ID=...
GOOGLE_CLIENT_SECRET=...
GOOGLE_CALLBACK_URL=https://app.example.com/auth/google/callback

# Session
SESSION_SECRET=... # 32+ random bytes

# n8n
N8N_WEBHOOK_BASE_URL=https://n8n.example.com

# Asynq
ASYNQ_CONCURRENCY=10
```

## Process Lifecycle

### Server Process
```
main()
  → Load config
  → Connect to Postgres (GORM)
  → Connect to Redis
  → Initialize Goth providers
  → Initialize Asynq client
  → Setup Gin router
  → Register middleware
  → Register handlers
  → Start HTTP server (blocking)
  → On SIGTERM: graceful shutdown (30s timeout)
```

### Worker Process
```
main()
  → Load config
  → Connect to Postgres (GORM)
  → Connect to Redis
  → Initialize Asynq server
  → Register task handlers
  → Start Asynq server (blocking)
  → On SIGTERM: graceful shutdown (finish in-flight tasks)
```

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Gin + HTMX patterns | HIGH | Well-established, training data recent |
| GORM + Postgres | HIGH | Mature, stable patterns |
| Asynq architecture | MEDIUM | Training data includes best practices, but version-specific features unknown |
| Templ integration | MEDIUM | Newer library, patterns still emerging |
| n8n integration | LOW | External system, depends on n8n's API stability |
| Kubernetes deployment | HIGH | Standard patterns, existing Helm charts |

## Sources

Based on:
- Go web service architecture patterns (training data)
- Gin framework best practices (training data)
- Background job queue patterns (training data)
- Server-side rendering with HTMX patterns (training data)
- Microservice deployment patterns (training data)
- Existing project structure (read from codebase)

**Unable to verify:**
- Latest Asynq monitoring best practices
- Templ + HTMX integration patterns (newer stack)
- n8n webhook API stability and versioning

---
*Architecture research for: Daily Briefing Go Web App*
*Researched: 2026-02-10*
*Confidence: MEDIUM-HIGH (standard patterns, external verification unavailable)*
