# Phase 7: Briefing History - Research

**Researched:** 2026-02-13
**Domain:** GORM pagination, HTMX dynamic lists, history UI patterns
**Confidence:** HIGH

## Summary

This phase adds a history/archive view for past briefings with 30-day retention. Research covers three primary domains: (1) GORM pagination and time-based filtering patterns, (2) HTMX click-to-load and infinite scroll strategies for rendering historical lists, and (3) UI/UX best practices for history views including date grouping and navigation patterns.

The existing codebase already has the infrastructure needed: `Briefing` model with timestamps, HTMX integration on the frontend, and the Liquid Glass design system with navbar components. The primary technical challenges are implementing efficient date-range queries, pagination for potentially large result sets, and presenting chronological data in a scannable format.

**Primary recommendation:** Use GORM's built-in `Limit/Offset` with time-based scopes for last-30-days filtering, implement HTMX click-to-load pattern for pagination, and present briefings in a reverse-chronological list grouped by date (not a calendar view). Leverage existing `BriefingCard` component for consistency. Do NOT implement actual archiving/deletion yet — "archived" means hidden from the history view but retained in database.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| GORM | Current (from go.mod) | Database queries with pagination | Already in use; provides `Limit`, `Offset`, and `Scopes` for reusable filtering |
| Templ | Current | Server-rendered HTML components | Already in use; supports inline Go expressions like `fmt.Sprintf` for date formatting |
| HTMX 2.0 | 2.0.0 | Dynamic content loading | Already in use; click-to-load pattern avoids full page reloads |
| PostgreSQL | Current | Database with timestamp support | Already in use; `CreatedAt` and `UpdatedAt` fields from `gorm.Model` |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| Go `time` package | stdlib | Date formatting and manipulation | For `time.Format()` with layout strings, `time.AddDate()` for 30-day calculations |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Simple pagination | Infinite scroll with `hx-trigger="revealed"` | Infinite scroll better for long lists but click-to-load gives user more control; choose click-to-load for clarity |
| Third-party pagination libraries | Hand-rolled `Limit/Offset` | Libraries like `github.com/morkid/paginate` add features but simple pagination doesn't justify dependency |
| Calendar grid view | List view with date headers | Calendar views require complex month/day calculations; list view is simpler and more mobile-friendly |

**Note:** Do NOT add soft delete yet. The requirement says "archived (not deleted)" but doesn't specify implementation. Keep records in the table; filter by date range only.

## Architecture Patterns

### Recommended Project Structure
```
internal/
├── briefings/
│   ├── handlers.go           # Add GetHistoryHandler, GetHistoryPageHandler
│   ├── templates.templ        # Add HistoryPage, HistoryList components
│   └── templates_templ.go    # Auto-generated
cmd/
└── server/
    └── main.go               # Add /history route
```

### Pattern 1: Scoped Time-Based Filtering
**What:** Define reusable GORM scope for "last 30 days" filtering
**When to use:** Any query that needs time-based filtering (can be reused in future phases)
**Example:**
```go
// In internal/briefings/queries.go (new file)
func Last30Days(db *gorm.DB) *gorm.DB {
    thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
    return db.Where("created_at >= ?", thirtyDaysAgo)
}

// Usage in handler
db.Scopes(Last30Days).
    Where("user_id = ?", userID).
    Order("created_at DESC").
    Limit(10).
    Offset(page * 10).
    Find(&briefings)
```

**Source:** [GORM Query Documentation](https://gorm.io/docs/query.html) — Scopes allow reusable query logic as `func(*gorm.DB) *gorm.DB`

### Pattern 2: HTMX Click-to-Load Pagination
**What:** Replaceable "Load More" element that fetches next page and swaps itself out
**When to use:** History lists, any paginated content where user controls loading
**Example:**
```html
<!-- Initial page (page 0) -->
<div id="history-list">
    <!-- Briefing cards here -->
    <div id="load-more-row">
        <button
            class="glass-btn glass-btn-ghost"
            hx-get="/api/history?page=1"
            hx-target="#load-more-row"
            hx-swap="outerHTML"
        >
            Load More
        </button>
    </div>
</div>

<!-- Server returns on click -->
<div>
    <!-- Next 10 briefing cards -->
    <div id="load-more-row">
        <button hx-get="/api/history?page=2" ...>Load More</button>
    </div>
</div>
```

**Source:** [HTMX Click to Load Example](https://htmx.org/examples/click-to-load/) — Each response includes new content plus updated button pointing to next page

### Pattern 3: Date Grouping in Templates
**What:** Group briefings by date (e.g., "February 13, 2026") in the UI without complex server-side grouping
**When to use:** Presenting chronological data that benefits from visual separation
**Example:**
```go
templ HistoryList(briefings []models.Briefing, page int, hasMore bool) {
    {{
        var lastDate string
    }}
    for _, briefing := range briefings {
        {{
            // Format date as "January 2, 2006"
            currentDate := briefing.CreatedAt.Format("January 2, 2006")
        }}
        if currentDate != lastDate {
            <div class="date-separator">
                <span class="date-label">{ currentDate }</span>
            </div>
            {{ lastDate = currentDate }}
        }
        @BriefingCard(briefing)
    }
    if hasMore {
        <div id="load-more-row">
            <button class="glass-btn glass-btn-ghost"
                hx-get={ fmt.Sprintf("/api/history?page=%d", page+1) }
                hx-target="#load-more-row"
                hx-swap="outerHTML">
                Load More
            </button>
        </div>
    }
}
```

**Source:** Templ [Statements documentation](https://templ.guide/syntax-and-usage/statements/) — Code blocks with `{{ }}` allow Go logic in templates

### Pattern 4: Navigation Link in Navbar
**What:** Add "History" link to existing navbar alongside "Logout"
**When to use:** Phase requires "Dashboard shows History navigation link"
**Example:**
```go
templ DashboardPage(...) {
    @Layout("Dashboard - First Sip") {
        <nav class="glass-navbar">
            <div class="navbar-inner">
                <div class="navbar-brand">
                    <img src="/static/img/logo.png" alt="" class="navbar-logo"/>
                    <span class="navbar-title">First Sip</span>
                </div>
                <div class="navbar-actions">
                    <a href="/history" class="navbar-link">History</a>  // NEW
                    <span class="navbar-user">{ name }</span>
                    <a href="/logout" class="glass-btn glass-btn-ghost glass-btn-sm">Logout</a>
                </div>
            </div>
        </nav>
        ...
    }
}
```

**Note:** Add `.navbar-link` CSS class to `liquid-glass.css` for consistent styling

### Anti-Patterns to Avoid
- **Don't preload all history:** Loading 30 days of briefings on initial page load is wasteful; use pagination
- **Don't use infinite scroll without indicator:** Click-to-load gives users control; auto-loading can be disorienting
- **Don't build a calendar grid yet:** Requirement is "browse past briefings" not "calendar view" — list view is simpler and mobile-friendly
- **Don't implement soft delete now:** Requirement says "archived (not deleted)" but doesn't specify mechanism; simple date filtering is sufficient

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Date pagination | Custom page calculation logic | GORM `Limit(10).Offset(page*10)` | Built-in, well-tested, handles edge cases |
| Date range filtering | Raw SQL with string interpolation | GORM `Where("created_at >= ?", date)` with Scopes | Parameterized queries prevent SQL injection; Scopes are reusable |
| Date formatting | Manual string manipulation | Go `time.Format("January 2, 2006")` | Go's reference-date layout system is idiomatic and readable |
| Load-more state | JavaScript pagination tracking | HTMX `hx-get` with page param in URL | Server controls pagination state; no client-side state to manage |

**Key insight:** HTMX eliminates need for client-side pagination state. Server returns HTML with next page's URL embedded in the "Load More" button.

## Common Pitfalls

### Pitfall 1: N+1 Queries on Relationships
**What goes wrong:** If future phases add user relationships or joins, naive iteration can trigger separate queries for each briefing
**Why it happens:** GORM lazy-loads associations by default
**How to avoid:** Use `Preload()` for associations when needed (not relevant for current phase but will matter if briefings gain relationships)
**Warning signs:** Slow history page load times, database query logs showing repetitive SELECT statements

### Pitfall 2: Timezone Confusion in Date Grouping
**What goes wrong:** User sees "February 13" but server groups by UTC date, so briefings appear under wrong day headers
**Why it happens:** `time.Now()` defaults to server timezone; `CreatedAt` is stored as UTC in PostgreSQL
**How to avoid:** Use `.UTC()` consistently or store user timezone preferences (future phase)
**Warning signs:** Date headers don't match user's local date

**Current decision:** Accept UTC for MVP; grouping by UTC date is acceptable for Phase 7

### Pitfall 3: Off-by-One Errors in "Last 30 Days"
**What goes wrong:** Using `>=` vs `>` with `time.Now().AddDate(0, 0, -30)` can include or exclude today incorrectly
**Why it happens:** Boundary conditions in date math are subtle
**How to avoid:** Use `>=` for inclusive start date; test with briefings exactly 30 days old
**Warning signs:** Briefings disappearing one day early or lingering one day late

### Pitfall 4: Empty State When No History Exists
**What goes wrong:** Blank page when user has no briefings in last 30 days
**Why it happens:** Template renders nothing if `len(briefings) == 0`
**How to avoid:** Add explicit empty state in template
**Warning signs:** User sees blank page, reports "history page is broken"

**Example:**
```go
if len(briefings) == 0 {
    <div class="empty-state">
        <p>No briefings in the last 30 days.</p>
        <a href="/dashboard" class="glass-btn glass-btn-primary">Go to Dashboard</a>
    </div>
}
```

### Pitfall 5: Forgetting to Filter by User ID
**What goes wrong:** History page shows all users' briefings (data leak)
**Why it happens:** Copy-pasting queries without adding `.Where("user_id = ?", userID)`
**How to avoid:** Always include user_id filter in every briefing query; use a scope for this too
**Warning signs:** QA finds other users' data in history view

**Example:**
```go
func ForUser(userID uint) func(*gorm.DB) *gorm.DB {
    return func(db *gorm.DB) *gorm.DB {
        return db.Where("user_id = ?", userID)
    }
}

// Usage
db.Scopes(ForUser(user.ID), Last30Days).Find(&briefings)
```

## Code Examples

Verified patterns from official sources and existing codebase:

### GORM Pagination with Time Filtering
```go
// Handler for /api/history?page=0
func GetHistoryPageHandler(db *gorm.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        // Get user ID from auth middleware
        userEmail, _ := c.Get("user_email")
        var user models.User
        if err := db.Where("email = ?", userEmail.(string)).First(&user).Error; err != nil {
            c.Status(http.StatusInternalServerError)
            return
        }

        // Parse page parameter
        page := 0
        if pageStr := c.Query("page"); pageStr != "" {
            page, _ = strconv.Atoi(pageStr)
        }

        // Query briefings with pagination
        const pageSize = 10
        var briefings []models.Briefing
        thirtyDaysAgo := time.Now().AddDate(0, 0, -30)

        result := db.Where("user_id = ?", user.ID).
            Where("created_at >= ?", thirtyDaysAgo).
            Order("created_at DESC").
            Limit(pageSize + 1).  // Request 1 extra to check if more exist
            Offset(page * pageSize).
            Find(&briefings)

        if result.Error != nil {
            c.Status(http.StatusInternalServerError)
            return
        }

        // Check if more pages exist
        hasMore := len(briefings) > pageSize
        if hasMore {
            briefings = briefings[:pageSize]
        }

        // Render HTML fragment
        c.Header("Content-Type", "text/html")
        HistoryList(briefings, page, hasMore).Render(c.Request.Context(), c.Writer)
    }
}
```

**Source:** [GORM Query Documentation](https://gorm.io/docs/query.html) — `Limit` and `Offset` for pagination

### Date Formatting in Templ
```go
templ HistoryList(briefings []models.Briefing, page int, hasMore bool) {
    <div id="history-list">
        if len(briefings) == 0 {
            <div class="empty-state">
                <p>No briefings in the last 30 days.</p>
            </div>
        } else {
            {{
                var lastDate string
            }}
            for _, briefing := range briefings {
                {{
                    currentDate := briefing.CreatedAt.Format("January 2, 2006")
                }}
                if currentDate != lastDate {
                    <div class="date-separator">
                        <span class="date-label">{ currentDate }</span>
                    </div>
                    {{ lastDate = currentDate }}
                }
                @BriefingCard(briefing)
            }
            if hasMore {
                <div id="load-more-row">
                    <button
                        class="glass-btn glass-btn-ghost"
                        hx-get={ fmt.Sprintf("/api/history?page=%d", page+1) }
                        hx-target="#load-more-row"
                        hx-swap="outerHTML"
                    >
                        Load More
                    </button>
                </div>
            }
        }
    </div>
}
```

**Source:** [Templ Statements](https://templ.guide/syntax-and-usage/statements/), [Go time.Format](https://pkg.go.dev/time) — Reference date layout "January 2, 2006"

### HTMX Click-to-Load Pattern
```html
<!-- Initial page load: /history -->
<main class="dashboard-content">
    <h1>Briefing History</h1>
    <div id="history-container">
        <!-- Server renders HistoryList with first 10 briefings -->
        <!-- Last element is Load More button targeting #load-more-row -->
    </div>
</main>

<!-- When Load More clicked, server returns: -->
<!-- 10 more briefing cards + new Load More button with page+1 -->
```

**Source:** [HTMX Click to Load](https://htmx.org/examples/click-to-load/) — Button's `hx-swap="outerHTML"` replaces itself with response

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Client-side pagination with JSON APIs | Server-rendered pagination with HTMX | ~2020-2023 | Simpler: no JSON parsing, no client state, progressive enhancement |
| Third-party pagination libraries | GORM built-in `Limit`/`Offset` | Stable since GORM v1 | Fewer dependencies; simple pagination doesn't need libraries |
| FuncMap for date formatting in html/template | Inline Go expressions in Templ | Templ released 2021 | Type-safe, no magic strings, compile-time checking |
| Soft delete with `DeletedAt` | Date-based filtering (for this use case) | N/A | Soft delete is for "undo"; time-based retention is a filter, not deletion |

**Deprecated/outdated:**
- Custom pagination libraries for simple offset pagination: GORM's built-in methods are sufficient
- JavaScript-driven "Load More": HTMX eliminates need for custom JS

## Open Questions

1. **Should "archived" be implemented with a separate flag or table?**
   - What we know: Requirement says "archived (not deleted)" but doesn't specify implementation
   - What's unclear: Whether to add `archived_at` field, move to separate table, or just filter by date
   - Recommendation: For Phase 7, "archived" = "older than 30 days" = hidden from UI but retained in DB. Do NOT add `archived_at` field yet; wait for explicit archival requirements in future phases.

2. **Should history show all statuses (pending/failed/completed)?**
   - What we know: Dashboard shows latest briefing regardless of status
   - What's unclear: Should history include failed briefings?
   - Recommendation: Show all statuses for transparency; users may want to see failed attempts. Filter in UI with badges.

3. **How many briefings per page?**
   - What we know: Standard pagination is 10-25 items per page
   - What's unclear: Briefing cards are large; 10 might be too few, 25 too many
   - Recommendation: Start with 10 per page; adjust in user testing if needed

4. **Should clicking a history briefing open in modal or inline?**
   - What we know: Dashboard briefing card is inline, click-to-mark-read
   - What's unclear: Whether history cards should behave the same
   - Recommendation: Consistent with dashboard; click marks read, no modal needed

## Sources

### Primary (HIGH confidence)
- [GORM Query Documentation](https://gorm.io/docs/query.html) — Limit, Offset, Scopes, time-based filtering
- [GORM Delete Documentation](https://gorm.io/docs/delete.html) — Soft delete with `gorm.DeletedAt`, `Unscoped()` usage
- [HTMX Click to Load Example](https://htmx.org/examples/click-to-load/) — Pagination pattern with `outerHTML` swap
- [Templ Statements Docs](https://templ.guide/syntax-and-usage/statements/) — Code blocks with `{{ }}` for Go logic
- [Go time package](https://pkg.go.dev/time) — `time.Format` with reference date layouts, `AddDate` for 30-day math

### Secondary (MEDIUM confidence)
- [GORM Pagination Best Practices](https://medium.com/@manueldoncelmartos/a-new-approach-to-pagination-in-gorm-188dcf16149e) — Clauses approach for advanced pagination (not needed here)
- [PostgreSQL Data Retention with pg_partman](https://www.crunchydata.com/blog/auto-archiving-and-data-retention-management-in-postgres-with-pg_partman) — Time-based partitioning for large-scale archival (future optimization)
- [Calendar UI Best Practices](https://www.eleken.co/blog-posts/calendar-ui) — List view vs calendar view for history (list view preferred for MVP)
- [HTMX Infinite Scroll Example](https://htmx.org/examples/infinite-scroll/) — `hx-trigger="revealed"` pattern (alternative to click-to-load)

### Tertiary (LOW confidence)
- [Third-party GORM pagination libraries](https://pkg.go.dev/github.com/morkid/paginate) — Not needed for simple pagination; mentioned for awareness

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — All libraries already in use; GORM and HTMX patterns well-documented
- Architecture: HIGH — Patterns verified against official docs and existing codebase
- Pitfalls: MEDIUM — Common issues documented in community sources; timezone handling requires testing

**Research date:** 2026-02-13
**Valid until:** 2026-03-15 (30 days; GORM and HTMX are stable, no major changes expected)
