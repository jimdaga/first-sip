# Phase 04: Briefing Generation (Mock) - Research

**Researched:** 2026-02-12
**Domain:** HTMX interactive UI patterns, Asynq task handlers, HTTP webhook clients, GORM JSON data
**Confidence:** HIGH

## Summary

Phase 4 implements the complete briefing generation flow: button click → API endpoint → task enqueue → worker processing → database update → polling UI. The stack is well-established with mature patterns.

**Key findings:**
- HTMX polling with `hx-trigger="every 2s"` provides simple status updates without WebSockets
- Asynq distinguishes retryable vs non-retryable errors via `asynq.SkipRetry` wrapper
- GORM's `datatypes.JSON` stores briefing content with type-safe access via `JSONType[T]`
- DaisyUI skeleton components provide built-in loading states
- Go `net/http` client needs explicit timeouts (never use `http.DefaultClient`)
- Stub mode can use conditional logic to return mock data instead of making HTTP calls

**Primary recommendation:** Use HTMX polling for status updates (simpler than SSE/WebSockets), implement idempotent task handlers with proper error classification, and structure briefing content as typed JSON for maintainability.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| HTMX | 2.0.0 | Interactive UI updates | Already loaded; polling pattern is industry standard for async job status |
| Asynq | latest | Background task processing | Already integrated; proven pattern for retry logic and error handling |
| GORM datatypes | latest | JSON field management | Native GORM support for JSONB with type-safe marshaling |
| DaisyUI | 4.x | UI components | Already loaded; built-in skeleton/loading states |
| Templ | latest | Component-based HTML | Already integrated; strongly-typed template composition |
| Gin | latest | HTTP routing | Already integrated; flexible response types (JSON/HTML) |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `net/http` stdlib | Go 1.21+ | HTTP client for webhooks | Standard library sufficient for simple HTTP calls |
| `context` stdlib | Go 1.21+ | Request cancellation, timeouts | Always use for DB queries and HTTP requests |
| `encoding/json` stdlib | Go 1.21+ | JSON marshaling for mock data | Standard library handles all JSON needs |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| HTMX polling | Server-Sent Events (SSE) | SSE requires persistent connections, more complex server setup |
| HTMX polling | WebSockets | WebSockets overkill for unidirectional status updates |
| `net/http` | `github.com/hashicorp/go-retryablehttp` | Only needed if retry logic required (not needed for stub mode) |
| HTMX `hx-swap="innerHTML"` | `hx-swap="outerHTML"` | `outerHTML` replaces entire element (use when updating attributes/classes) |

**Installation:**
```bash
# All dependencies already installed in prior phases
# HTMX, DaisyUI loaded via CDN in layout.templ
# Asynq installed in Phase 3
# GORM datatypes part of GORM installation
```

## Architecture Patterns

### Recommended Project Structure
```
internal/
├── briefings/           # NEW - Briefing domain logic
│   ├── handlers.go      # Gin routes: POST /api/briefings, GET /api/briefings/:id/status
│   ├── service.go       # Business logic: CreateBriefing, GetBriefingStatus
│   └── templates.go     # Templ components: BriefingCard, BriefingStatus
├── webhook/             # NEW - n8n webhook client (stub mode)
│   ├── client.go        # HTTP client with X-N8N-SECRET header
│   └── types.go         # Request/response types for n8n API
├── worker/
│   └── handlers/        # NEW - Separate handler logic
│       └── briefing.go  # handleGenerateBriefing implementation
└── models/
    └── briefing.go      # Existing - add Content struct definition
```

### Pattern 1: HTMX Polling for Status Updates

**What:** Poll an endpoint until terminal state (completed/failed), then stop polling

**When to use:** Async operations where status changes server-side (background jobs, external API calls)

**Example:**
```html
<!-- Initial state: polling every 2 seconds -->
<div id="briefing-status"
     hx-get="/api/briefings/123/status"
     hx-trigger="every 2s"
     hx-swap="outerHTML">
  <span class="loading loading-spinner"></span> Generating...
</div>

<!-- Server returns this when completed (no hx-trigger = polling stops) -->
<div id="briefing-status">
  <span class="text-success">✓</span> Completed
</div>
```

**Key insight:** Stop polling by omitting `hx-trigger` in terminal state response. Alternative: return HTTP 286 to explicitly stop polling.

**Source:** [HTMX hx-trigger documentation](https://htmx.org/attributes/hx-trigger/)

### Pattern 2: Asynq Task Handler with Error Classification

**What:** Distinguish retryable errors (network failures) from non-retryable errors (invalid data)

**When to use:** All background task handlers

**Example:**
```go
// Source: https://pkg.go.dev/github.com/hibiken/asynq
func handleGenerateBriefing(logger *slog.Logger, db *gorm.DB) func(context.Context, *asynq.Task) error {
    return func(ctx context.Context, task *asynq.Task) error {
        var payload struct {
            BriefingID uint `json:"briefing_id"`
        }

        // Non-retryable: invalid JSON payload
        if err := json.Unmarshal(task.Payload(), &payload); err != nil {
            return fmt.Errorf("invalid payload: %w", asynq.SkipRetry)
        }

        // Fetch briefing
        var briefing models.Briefing
        if err := db.WithContext(ctx).First(&briefing, payload.BriefingID).Error; err != nil {
            if errors.Is(err, gorm.ErrRecordNotFound) {
                return fmt.Errorf("briefing not found: %w", asynq.SkipRetry) // Non-retryable
            }
            return err // Retryable (DB connection issue)
        }

        // Update status to processing
        db.Model(&briefing).Update("status", models.BriefingStatusProcessing)

        // Call webhook (stub mode = return mock data)
        content, err := fetchBriefingContent(ctx, briefing.UserID)
        if err != nil {
            // Update to failed state
            db.Model(&briefing).Updates(map[string]interface{}{
                "status": models.BriefingStatusFailed,
                "error_message": err.Error(),
            })
            return err // Retryable
        }

        // Update to completed
        db.Model(&briefing).Updates(map[string]interface{}{
            "status": models.BriefingStatusCompleted,
            "content": datatypes.NewJSONType(content),
            "generated_at": time.Now(),
        })

        return nil
    }
}
```

**Key insight:** Wrap errors with `asynq.SkipRetry` for unrecoverable failures (validation errors, 404s). Let other errors retry automatically.

**Source:** [Asynq best practices](https://pkg.go.dev/github.com/hibiken/asynq)

### Pattern 3: GORM datatypes.JSON with Type Safety

**What:** Store structured JSON in Postgres JSONB column with Go struct types

**When to use:** Complex nested data that doesn't deserve its own tables (briefing content sections)

**Example:**
```go
// Source: https://pkg.go.dev/gorm.io/datatypes
type BriefingContent struct {
    News    []NewsItem    `json:"news"`
    Weather WeatherInfo   `json:"weather"`
    Work    WorkSummary   `json:"work"`
}

type NewsItem struct {
    Title   string `json:"title"`
    Summary string `json:"summary"`
    URL     string `json:"url"`
}

type WeatherInfo struct {
    Location    string `json:"location"`
    Temperature int    `json:"temperature"`
    Condition   string `json:"condition"`
}

type WorkSummary struct {
    TodayEvents  []string `json:"today_events"`
    TomorrowTasks []string `json:"tomorrow_tasks"`
}

// In models/briefing.go
type Briefing struct {
    gorm.Model
    UserID      uint                                    `gorm:"not null;index"`
    User        User                                    `gorm:"constraint:OnDelete:CASCADE;"`
    Content     datatypes.JSONType[BriefingContent]     `gorm:"type:jsonb"`
    Status      string                                  `gorm:"not null;default:'pending';index"`
    ErrorMessage string                                 `gorm:"column:error_message;type:text"`
    GeneratedAt *time.Time
    ReadAt      *time.Time
}

// Usage
briefing.Content = datatypes.NewJSONType(BriefingContent{
    News: []NewsItem{
        {Title: "Mock News", Summary: "This is mock content", URL: "https://example.com"},
    },
    Weather: WeatherInfo{Location: "San Francisco", Temperature: 65, Condition: "Sunny"},
    Work: WorkSummary{
        TodayEvents: []string{"Team standup at 10am", "Client meeting at 2pm"},
        TomorrowTasks: []string{"Review PR #123", "Deploy to staging"},
    },
})

// Access typed data
data := briefing.Content.Data()
fmt.Println(data.Weather.Temperature) // Type-safe access
```

**Key insight:** `datatypes.JSONType[T]` provides type-safe marshaling/unmarshaling. Use it instead of `datatypes.JSON` (raw bytes) for structured content.

**Source:** [GORM datatypes documentation](https://pkg.go.dev/gorm.io/datatypes)

### Pattern 4: Stub Mode via Conditional Logic

**What:** Return mock data when in stub mode without making external HTTP calls

**When to use:** Early development phases, testing, environments without external dependencies

**Example:**
```go
// internal/webhook/client.go
type Client struct {
    baseURL    string
    secret     string
    httpClient *http.Client
    stubMode   bool
}

func NewClient(baseURL, secret string, stubMode bool) *Client {
    return &Client{
        baseURL: baseURL,
        secret:  secret,
        stubMode: stubMode,
        httpClient: &http.Client{
            Timeout: 30 * time.Second, // ALWAYS set timeout
        },
    }
}

func (c *Client) GenerateBriefing(ctx context.Context, userID uint) (*BriefingContent, error) {
    // Stub mode: return mock data immediately
    if c.stubMode {
        return &BriefingContent{
            News: []NewsItem{
                {Title: "Mock: Market Update", Summary: "Tech stocks rally...", URL: "https://example.com"},
            },
            Weather: WeatherInfo{Location: "San Francisco", Temperature: 65, Condition: "Sunny"},
            Work: WorkSummary{
                TodayEvents: []string{"Team standup at 10am"},
                TomorrowTasks: []string{"Review code"},
            },
        }, nil
    }

    // Production mode: make actual HTTP request
    req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/generate", nil)
    if err != nil {
        return nil, err
    }

    req.Header.Set("X-N8N-SECRET", c.secret)
    req.Header.Set("Content-Type", "application/json")

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    // ... parse response ...
}
```

**Key insight:** Stub mode controlled by environment variable. Same interface, different implementation path.

**Source:** Standard Go testing patterns

### Pattern 5: Templ Component Composition

**What:** Break UI into reusable components with typed parameters

**When to use:** Repeated UI patterns, conditional rendering

**Example:**
```templ
// internal/briefings/templates.go
package briefings

import "github.com/jimdaga/first-sip/internal/models"

// BriefingCard renders a single briefing with status
templ BriefingCard(briefing models.Briefing) {
    <div class="card bg-base-100 shadow-xl">
        <div class="card-body">
            <h3 class="card-title">Daily Briefing</h3>
            @BriefingStatus(briefing)
            if briefing.Status == models.BriefingStatusCompleted {
                @BriefingContent(briefing.Content.Data())
            }
        </div>
    </div>
}

// BriefingStatus renders status with polling if pending
templ BriefingStatus(briefing models.Briefing) {
    if briefing.Status == models.BriefingStatusPending || briefing.Status == models.BriefingStatusProcessing {
        <div
            id={ "briefing-status-" + fmt.Sprint(briefing.ID) }
            hx-get={ "/api/briefings/" + fmt.Sprint(briefing.ID) + "/status" }
            hx-trigger="every 2s"
            hx-swap="outerHTML">
            <span class="loading loading-spinner loading-sm"></span>
            <span class="ml-2">{ briefing.Status }...</span>
        </div>
    } else if briefing.Status == models.BriefingStatusCompleted {
        <div id={ "briefing-status-" + fmt.Sprint(briefing.ID) } class="text-success">
            <span>✓</span> Completed
        </div>
    } else if briefing.Status == models.BriefingStatusFailed {
        <div id={ "briefing-status-" + fmt.Sprint(briefing.ID) } class="text-error">
            <span>✗</span> Failed: { briefing.ErrorMessage }
        </div>
    }
}

// BriefingContent renders the actual briefing sections
templ BriefingContent(content BriefingContent) {
    <div class="space-y-4 mt-4">
        <section>
            <h4 class="font-bold text-lg">News</h4>
            for _, item := range content.News {
                <div class="p-2 border-l-4 border-blue-500 my-2">
                    <a href={ templ.URL(item.URL) } class="link">{item.Title}</a>
                    <p class="text-sm text-gray-600">{item.Summary}</p>
                </div>
            }
        </section>

        <section>
            <h4 class="font-bold text-lg">Weather</h4>
            <p>{content.Weather.Location}: {fmt.Sprint(content.Weather.Temperature)}°F, {content.Weather.Condition}</p>
        </section>

        <section>
            <h4 class="font-bold text-lg">Work</h4>
            <div>
                <h5 class="font-semibold">Today's Events:</h5>
                <ul class="list-disc list-inside">
                    for _, event := range content.Work.TodayEvents {
                        <li>{event}</li>
                    }
                </ul>
            </div>
        </section>
    </div>
}
```

**Key insight:** Components compose naturally. Status polling component re-renders itself via HTMX until terminal state reached.

**Source:** [Templ template composition](https://templ.guide/syntax-and-usage/template-composition/)

### Anti-Patterns to Avoid

- **Don't use `http.DefaultClient`:** It has no timeout, can hang indefinitely. Always create custom `http.Client` with `Timeout` field.
- **Don't poll faster than necessary:** 2-5 second intervals sufficient for background jobs. Faster polling wastes resources.
- **Don't update individual fields separately:** Use `Updates(map[string]interface{})` for atomic multi-field updates.
- **Don't return partial HTML in polling response:** Return complete replacement element with correct `id` for `hx-swap="outerHTML"`.
- **Don't retry unrecoverable errors:** Wrap validation errors, 404s, bad requests with `asynq.SkipRetry` to prevent wasted retry cycles.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Background job retries | Custom retry logic with goroutines | Asynq task queue | Edge cases: exponential backoff, dead letter queue, task deduplication, graceful shutdown |
| HTTP request retries | Manual retry loops | `hashicorp/go-retryablehttp` | Edge cases: jitter, max attempts, retryable status codes, connection pooling |
| JSON schema validation | Custom validation functions | Struct tags + validation library | Edge cases: nested validation, custom validators, error messages |
| Real-time updates | Custom WebSocket server | HTMX polling (for simple status) | Edge cases: connection management, reconnection, broadcast, message ordering |
| Database transactions | Manual BEGIN/COMMIT/ROLLBACK | GORM's `Transaction` method | Edge cases: nested transactions, savepoints, deadlock detection |

**Key insight:** Async job processing has surprising complexity (retry storms, connection leaks, graceful shutdown). Asynq handles this. Don't reinvent.

## Common Pitfalls

### Pitfall 1: Polling Continues After Terminal State

**What goes wrong:** HTMX continues polling even after briefing completes because response still contains `hx-trigger="every 2s"`

**Why it happens:** Developer returns same polling element structure for all states

**How to avoid:**
- **Option 1:** Remove `hx-trigger` attribute from terminal state responses
- **Option 2:** Return HTTP 286 status code to explicitly stop polling

**Warning signs:** Network tab shows continued requests after status shows "Completed"

**Example fix:**
```templ
// WRONG - keeps polling
templ BriefingStatus(briefing models.Briefing) {
    <div hx-get={...} hx-trigger="every 2s" hx-swap="outerHTML">
        if briefing.Status == "completed" {
            Completed!
        } else {
            Pending...
        }
    </div>
}

// RIGHT - stops polling on completion
templ BriefingStatus(briefing models.Briefing) {
    if briefing.Status == "pending" {
        <div hx-get={...} hx-trigger="every 2s" hx-swap="outerHTML">
            Pending...
        </div>
    } else {
        <div>Completed!</div>
    }
}
```

**Source:** [HTMX polling patterns](https://hamy.xyz/blog/2024-07_htmx-polling-example)

### Pitfall 2: Task Handler Retries Invalid Data

**What goes wrong:** Worker retries task 3 times even though payload is malformed JSON, wasting resources and delaying failure visibility

**Why it happens:** All errors treated as retryable by default in Asynq

**How to avoid:** Wrap non-retryable errors (validation, 404s, 400s) with `fmt.Errorf("...: %w", asynq.SkipRetry)`

**Warning signs:** Task logs show same JSON unmarshal error across multiple retry attempts

**Example fix:**
```go
// WRONG - retries on bad JSON
if err := json.Unmarshal(task.Payload(), &payload); err != nil {
    return err // Will retry 3 times
}

// RIGHT - skips retry
if err := json.Unmarshal(task.Payload(), &payload); err != nil {
    return fmt.Errorf("invalid payload: %w", asynq.SkipRetry)
}
```

**Source:** [Asynq error handling best practices](https://pkg.go.dev/github.com/hibiken/asynq)

### Pitfall 3: Database Updates Outside Transaction

**What goes wrong:** Status updates to "processing" succeeds, but worker crashes before updating to "completed". Briefing stuck in "processing" state forever.

**Why it happens:** Multiple database updates not wrapped in transaction

**How to avoid:** Use `db.Transaction()` for multi-step updates, or ensure final state update includes all changes

**Warning signs:** Orphaned records in "processing" state after worker restarts

**Example fix:**
```go
// WRONG - multiple separate updates
db.Model(&briefing).Update("status", "processing")
content, err := fetchContent(ctx)
if err != nil {
    db.Model(&briefing).Updates(map[string]interface{}{
        "status": "failed",
        "error_message": err.Error(),
    })
    return err
}
db.Model(&briefing).Updates(map[string]interface{}{
    "status": "completed",
    "content": content,
})

// RIGHT - single final update (optimistic approach)
// Only update status at the end
content, err := fetchContent(ctx)
if err != nil {
    db.Model(&briefing).Updates(map[string]interface{}{
        "status": "failed",
        "error_message": err.Error(),
    })
    return err
}
db.Model(&briefing).Updates(map[string]interface{}{
    "status": "completed",
    "content": content,
    "generated_at": time.Now(),
})
```

**Alternative:** Use GORM transactions for complex multi-table updates, but single-record updates can rely on task retry for consistency.

**Source:** [GORM transaction best practices](https://gorm.io/docs/transactions.html)

### Pitfall 4: HTTP Client Without Timeout

**What goes wrong:** Webhook call to n8n hangs indefinitely if n8n is down, blocking worker goroutine and potentially exhausting worker pool

**Why it happens:** `http.DefaultClient` has no timeout configured

**How to avoid:** Always create custom `http.Client` with explicit `Timeout` field

**Warning signs:** Worker stops processing new tasks, goroutines accumulate, no error logs

**Example fix:**
```go
// WRONG - can hang forever
resp, err := http.Post(url, "application/json", body)

// RIGHT - 30 second timeout
client := &http.Client{
    Timeout: 30 * time.Second,
}
resp, err := client.Post(url, "application/json", body)
```

**Source:** [Go HTTP client timeout best practices](https://medium.com/@nate510/don-t-use-go-s-default-http-client-4804cb19f779)

### Pitfall 5: HTMX Swap Target Mismatch

**What goes wrong:** Status update returns HTML with `id="status-123"` but HTMX can't find element to swap because page has `id="briefing-status-123"`

**Why it happens:** Inconsistent ID naming between initial render and HTMX response

**How to avoid:** Use consistent ID generation. Templ components with shared ID logic help enforce this.

**Warning signs:** Network shows 200 response but UI doesn't update

**Example fix:**
```templ
// WRONG - IDs don't match
// Initial render
<div id="status-{briefing.ID}">...</div>

// HTMX response
<div id="briefing-status-{briefing.ID}">...</div>

// RIGHT - consistent ID
// Both use same ID
<div id={ "briefing-status-" + fmt.Sprint(briefing.ID) }>...</div>
```

**Source:** [HTMX swap documentation](https://htmx.org/attributes/hx-swap/)

## Code Examples

Verified patterns from official sources and project conventions:

### POST Endpoint to Trigger Briefing

```go
// internal/briefings/handlers.go
func CreateBriefingHandler(db *gorm.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        userID, exists := c.Get("user_id")
        if !exists {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
            return
        }

        // Create briefing record in pending state
        briefing := models.Briefing{
            UserID: userID.(uint),
            Status: models.BriefingStatusPending,
        }

        if err := db.Create(&briefing).Error; err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create briefing"})
            return
        }

        // Enqueue background task
        if err := worker.EnqueueGenerateBriefing(briefing.ID); err != nil {
            // Task failed to enqueue, update status
            db.Model(&briefing).Updates(map[string]interface{}{
                "status": models.BriefingStatusFailed,
                "error_message": "Failed to enqueue task",
            })
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to enqueue task"})
            return
        }

        // Return HTMX-compatible HTML fragment
        c.Header("Content-Type", "text/html")
        templates.BriefingCard(briefing).Render(c.Request.Context(), c.Writer)
    }
}
```

### Status Polling Endpoint

```go
// internal/briefings/handlers.go
func GetBriefingStatusHandler(db *gorm.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        briefingID := c.Param("id")

        var briefing models.Briefing
        if err := db.First(&briefing, briefingID).Error; err != nil {
            c.String(http.StatusNotFound, "Not found")
            return
        }

        // Return status fragment (HTMX will swap this in)
        c.Header("Content-Type", "text/html")
        templates.BriefingStatus(briefing).Render(c.Request.Context(), c.Writer)
    }
}
```

### Worker Task Handler Implementation

```go
// internal/worker/handlers/briefing.go
package handlers

import (
    "context"
    "encoding/json"
    "fmt"
    "log/slog"
    "time"

    "github.com/hibiken/asynq"
    "gorm.io/datatypes"
    "gorm.io/gorm"

    "github.com/jimdaga/first-sip/internal/models"
    "github.com/jimdaga/first-sip/internal/webhook"
)

func HandleGenerateBriefing(logger *slog.Logger, db *gorm.DB, webhookClient *webhook.Client) func(context.Context, *asynq.Task) error {
    return func(ctx context.Context, task *asynq.Task) error {
        var payload struct {
            BriefingID uint `json:"briefing_id"`
        }

        // Non-retryable: invalid payload
        if err := json.Unmarshal(task.Payload(), &payload); err != nil {
            logger.Error("Invalid task payload", "error", err)
            return fmt.Errorf("json unmarshal failed: %w", asynq.SkipRetry)
        }

        logger.Info("Generating briefing", "briefing_id", payload.BriefingID)

        // Fetch briefing
        var briefing models.Briefing
        if err := db.WithContext(ctx).First(&briefing, payload.BriefingID).Error; err != nil {
            if errors.Is(err, gorm.ErrRecordNotFound) {
                logger.Error("Briefing not found", "briefing_id", payload.BriefingID)
                return fmt.Errorf("briefing not found: %w", asynq.SkipRetry)
            }
            return fmt.Errorf("db query failed: %w", err)
        }

        // Update to processing (optional - shows progress)
        db.Model(&briefing).Update("status", models.BriefingStatusProcessing)

        // Call webhook client (stub mode returns mock data)
        content, err := webhookClient.GenerateBriefing(ctx, briefing.UserID)
        if err != nil {
            logger.Error("Webhook call failed", "error", err, "briefing_id", briefing.ID)

            // Mark as failed
            db.Model(&briefing).Updates(map[string]interface{}{
                "status": models.BriefingStatusFailed,
                "error_message": err.Error(),
            })

            return fmt.Errorf("webhook failed: %w", err) // Retryable
        }

        // Success - update to completed with content
        now := time.Now()
        updates := map[string]interface{}{
            "status": models.BriefingStatusCompleted,
            "content": datatypes.NewJSONType(*content),
            "generated_at": &now,
        }

        if err := db.Model(&briefing).Updates(updates).Error; err != nil {
            logger.Error("Failed to update briefing", "error", err, "briefing_id", briefing.ID)
            return fmt.Errorf("db update failed: %w", err) // Retryable
        }

        logger.Info("Briefing generated successfully", "briefing_id", briefing.ID)
        return nil
    }
}
```

### Webhook Client with Stub Mode

```go
// internal/webhook/client.go
package webhook

import (
    "context"
    "fmt"
    "net/http"
    "time"
)

type Client struct {
    baseURL    string
    secret     string
    httpClient *http.Client
    stubMode   bool
}

func NewClient(baseURL, secret string, stubMode bool) *Client {
    return &Client{
        baseURL: baseURL,
        secret:  secret,
        stubMode: stubMode,
        httpClient: &http.Client{
            Timeout: 30 * time.Second,
        },
    }
}

func (c *Client) GenerateBriefing(ctx context.Context, userID uint) (*BriefingContent, error) {
    if c.stubMode {
        // Return mock data in stub mode
        return &BriefingContent{
            News: []NewsItem{
                {
                    Title:   "Mock: Tech Stocks Rally on AI News",
                    Summary: "Major technology companies saw gains today following announcements about new AI capabilities.",
                    URL:     "https://example.com/news/1",
                },
                {
                    Title:   "Mock: Climate Summit Begins",
                    Summary: "World leaders gather to discuss climate change initiatives and carbon reduction targets.",
                    URL:     "https://example.com/news/2",
                },
            },
            Weather: WeatherInfo{
                Location:    "San Francisco",
                Temperature: 65,
                Condition:   "Partly Cloudy",
            },
            Work: WorkSummary{
                TodayEvents: []string{
                    "Team standup at 10:00 AM",
                    "Client presentation at 2:00 PM",
                    "Code review session at 4:00 PM",
                },
                TomorrowTasks: []string{
                    "Review PR #123",
                    "Deploy to staging environment",
                    "Update project documentation",
                },
            },
        }, nil
    }

    // Production mode: make actual HTTP request to n8n webhook
    req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/generate", nil)
    if err != nil {
        return nil, fmt.Errorf("create request: %w", err)
    }

    req.Header.Set("X-N8N-SECRET", c.secret)
    req.Header.Set("Content-Type", "application/json")

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("http request: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
    }

    var content BriefingContent
    if err := json.NewDecoder(resp.Body).Decode(&content); err != nil {
        return nil, fmt.Errorf("decode response: %w", err)
    }

    return &content, nil
}
```

### DaisyUI Loading States

```html
<!-- Skeleton loader while generating -->
<div class="flex flex-col gap-4">
  <div class="skeleton h-8 w-full"></div>
  <div class="skeleton h-4 w-3/4"></div>
  <div class="skeleton h-4 w-full"></div>
  <div class="skeleton h-32 w-full"></div>
</div>

<!-- Spinner with text -->
<div class="flex items-center gap-2">
  <span class="loading loading-spinner loading-sm"></span>
  <span>Generating your briefing...</span>
</div>

<!-- Progress indicator (optional) -->
<progress class="progress progress-primary w-56" value="70" max="100"></progress>
```

**Source:** [DaisyUI skeleton component](https://daisyui.com/components/skeleton/)

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| WebSockets for all async updates | HTMX polling for simple status | 2023-2024 | Simpler server, no connection management |
| `datatypes.JSON` (raw bytes) | `datatypes.JSONType[T]` (typed) | GORM v1.25+ (2024) | Type safety, better autocomplete |
| Manual error classification | `asynq.SkipRetry` wrapper | Asynq best practices 2023+ | Clear retry semantics |
| `http.DefaultClient` | Custom client with timeout | Go best practices (ongoing) | Prevents indefinite hangs |
| Separate polling library | HTMX native `hx-trigger="every"` | HTMX 2.0 (2024) | One less dependency |

**Deprecated/outdated:**
- **`hx-sse` extension:** Use native polling or standard SSE, extension less maintained
- **`gorm:"type:json"` without datatypes:** Use `datatypes.JSON` or `datatypes.JSONType[T]` for proper JSONB support
- **Manual AJAX for HTMX responses:** Let HTMX handle requests, don't mix jQuery/fetch

## Open Questions

1. **Polling interval optimization**
   - What we know: 2-5 seconds is typical for background job status
   - What's unclear: Impact on Redis/database load with multiple concurrent users
   - Recommendation: Start with 2 seconds, monitor query metrics, increase if load issues

2. **Error message user visibility**
   - What we know: `ErrorMessage` field stores technical errors
   - What's unclear: Should users see raw errors or friendly messages?
   - Recommendation: Store technical error in DB, display friendly "Generation failed. Please try again." to users, log details for debugging

3. **Briefing retention policy**
   - What we know: Briefings accumulate in database
   - What's unclear: When/how to clean up old briefings
   - Recommendation: Add `DeletedAt` (soft delete), implement cleanup job in Phase 5+

4. **Concurrent briefing generation**
   - What we know: User could click "Generate" multiple times
   - What's unclear: Should we prevent duplicate pending briefings?
   - Recommendation: Add unique constraint on `(user_id, status)` where status is 'pending', or check before creating

## Sources

### Primary (HIGH confidence)
- [HTMX hx-trigger attribute](https://htmx.org/attributes/hx-trigger/) - Polling syntax and patterns
- [Asynq package documentation](https://pkg.go.dev/github.com/hibiken/asynq) - Error handling, task handlers, retry semantics
- [GORM datatypes package](https://pkg.go.dev/gorm.io/datatypes) - JSON type usage and examples
- [GORM transactions documentation](https://gorm.io/docs/transactions.html) - Transaction patterns
- [DaisyUI skeleton component](https://daisyui.com/components/skeleton/) - Loading state classes
- [Templ template composition](https://templ.guide/syntax-and-usage/template-composition/) - Component patterns

### Secondary (MEDIUM confidence)
- [How to Build Simple Web Polling with HTMX](https://hamy.xyz/blog/2024-07_htmx-polling-example) - Polling until terminal state pattern
- [Go Testing Excellence: Table-Driven Tests and Mocking](https://dasroot.net/posts/2026/01/go-testing-excellence-table-driven-tests-mocking/) - 2026 testing approaches
- [How to Set HTTP Client Timeouts in Go (2026)](https://oneuptime.com/blog/post/2026-01-23-go-http-timeouts/view) - Current timeout best practices
- [How to Implement Retry Logic in Go (2026)](https://oneuptime.com/blog/post/2026-01-07-go-retry-exponential-backoff/view) - Retry patterns
- [n8n webhook credentials documentation](https://docs.n8n.io/integrations/builtin/credentials/webhook/) - Authentication methods
- [n8n header authentication community discussion](https://community.n8n.io/t/header-authentication-in-webhook/19511) - Custom header setup

### Tertiary (LOW confidence)
- [WebSearch: HTMX swap attribute](https://htmx.org/attributes/hx-swap/) - innerHTML vs outerHTML behavior
- [WebSearch: Go mock HTTP responses](https://medium.com/zus-health/mocking-outbound-http-requests-in-go-youre-probably-doing-it-wrong-60373a38d2aa) - Testing patterns
- [WebSearch: Gin with HTMX tutorial](https://www.hackingwithgo.nl/2024/03/11/how-to-mix-magic-a-fun-dive-into-go-gin-gonic-gorm-and-htmx-part-3-the-api/) - Response pattern examples

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All libraries already integrated, well-documented patterns
- Architecture: HIGH - HTMX polling, Asynq handlers, GORM JSON verified via official docs
- Pitfalls: HIGH - Common errors documented in issue trackers and best practice guides

**Research date:** 2026-02-12
**Valid until:** ~30 days (stable ecosystem, unlikely to change rapidly)
