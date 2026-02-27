# Phase 11: Tile-Based Dashboard - Research

**Researched:** 2026-02-21
**Domain:** CSS Grid tile layout, HTMX polling, SortableJS drag-and-drop, PostgreSQL window queries, Go/Templ component patterns
**Confidence:** HIGH (stack is well-established, codebase patterns are clear, all claims verified)

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### Tile content & density
- Plugin icon (from YAML metadata) displayed prominently on each tile
- Plugin name as tile heading
- 2-3 line briefing preview from a dedicated summary field (CrewAI generates summary alongside full briefing)
- Timing info (last run, next scheduled run) in hover-over tooltip on an info badge — not visible as text on tile
- Status indicators: subtle spinner icon when generating, error icon when last run failed (no text)
- Error state: show previous successful briefing content with error icon overlay — don't replace content
- No visible timestamp on tile face — all timing details in hover only

#### Tile interactions
- Click tile to expand in-place within the grid — other tiles reflow around it
- One tile expanded at a time — expanding a new tile auto-collapses the previous
- Collapse via close button in expanded view OR clicking outside the expanded content
- No quick actions on tiles (no run-now, no settings link) — those belong in Phase 12 Settings

#### Layout & sizing
- Plugin-defined tile sizes using grid units: 1x1, 2x1, 2x2 (declared in plugin YAML metadata)
- User-defined tile ordering with drag-and-drop (SortableJS or similar alongside HTMX)
- Order persists per user in database
- Dashboard header above tiles: time-aware greeting ("Good morning, Jim") + date
- Header is greeting only — no status summary line
- Mobile: single column, same user-defined order, tiles full-width stacked

#### Empty & loading states
- No plugins enabled: onboarding prompt ("Enable your first plugin to get started") with button linking to Settings
- Plugin enabled but no briefing yet: tile shows "Your first briefing is scheduled for [next-run time]"
- Briefing just completed: smooth fade transition for new content (no highlight/pulse)
- Auto-poll every 30 seconds via HTMX for live tile updates

### Claude's Discretion
- Initial page load pattern (skeleton tiles vs spinner)
- Expanded tile sizing behavior (full-width vs proportional expansion)
- Exact spacing, typography, and glass card styling within design system
- Drag-and-drop library choice
- Hover tooltip implementation approach

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope
</user_constraints>

---

## Summary

This phase replaces the current single-briefing dashboard with a CSS Grid tile layout showing all user-enabled plugins. The core technical challenge is multi-dimensional: CSS Grid with variable-size tiles and expand-in-place behavior, SortableJS drag-and-drop alongside HTMX, efficient PostgreSQL queries for latest plugin run per user-plugin pair, and several new database schema additions (summary field on PluginRun output, tile_size/icon on PluginMetadata, display_order on UserPluginConfig).

The existing codebase is well-structured for this extension. The `plugins` package already has `PluginMetadata`, `UserPluginConfig`, and `PluginRun` models. The migration pattern (numbered SQL files in `internal/database/migrations/`) is established. HTMX polling already works in `briefing.templ` with `hx-trigger="every 2s"`. The main work is: schema additions, two new Go view-model structs, a new dashboard handler, updated `dashboard.templ`, CSS additions to `liquid-glass.css`, and SortableJS initialization inline in the layout or dashboard template.

The recommended approach for initial page load is skeleton tiles (Claude's discretion): render tile outlines immediately from UserPluginConfig data, then the content fills in. This is superior to a full-page spinner because users see their configured plugins instantly and the skeleton communicates structure before content arrives.

**Primary recommendation:** Use CSS Grid with `grid-template-columns: repeat(auto-fit, minmax(280px, 1fr))`, `grid-column: span N` for tile sizes, SortableJS 1.15.7 for drag-and-drop, PostgreSQL `DISTINCT ON` for latest-run-per-plugin, HTMX `hx-trigger="every 30s"` for polling, and pure CSS `::after` + `data-tooltip` for timing tooltips.

---

## Standard Stack

### Core (all already in project)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| CSS Grid | Native browser | Tile layout, variable column spans | No-JS responsive layout; `auto-fit/minmax` handles reflow when tiles expand |
| HTMX | 2.0.0 (CDN, already loaded) | Polling, in-place swap, drag-end POST | Project standard; `hx-trigger="every 30s"` and `hx-swap="outerHTML"` cover all tile update needs |
| Templ | v0.3.977 (already in go.mod) | Type-safe HTML generation | Project standard; inline `style={ }` supports computed `grid-column: span N` |
| GORM + PostgreSQL | gorm v1.31.1 (already in go.mod) | DB queries | Project standard; raw SQL via `db.Raw()` for DISTINCT ON window query |

### Supporting (new additions)
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| SortableJS | 1.15.7 (CDN) | Drag-and-drop tile reordering | User tile ordering — the HTMX official example uses SortableJS exactly |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| SortableJS | dragula, @dnd-kit | SortableJS is the library called out in the official HTMX examples; no build step required via CDN |
| CSS tooltip (::after) | Tippy.js, Floating UI | Pure CSS is sufficient for a static info-badge tooltip; no JS dependency needed |
| DISTINCT ON query | ROW_NUMBER() window function | DISTINCT ON is idiomatic PostgreSQL and simpler for this exact use case; ROW_NUMBER requires a CTE or subquery |

**Installation (CDN additions to `layout.templ` `<head>`):**
```html
<script src="https://cdn.jsdelivr.net/npm/sortablejs@1.15.7/Sortable.min.js"></script>
```

No npm build step — SortableJS loads via CDN like HTMX.

---

## Architecture Patterns

### Recommended Project Structure (additions only)

```
internal/
├── dashboard/
│   └── handlers.go          # New: GET /dashboard, POST /api/tiles/order
├── templates/
│   └── dashboard.templ      # Replace: tile grid layout replacing single card
static/css/
└── liquid-glass.css         # Extend: add tile grid, skeleton, tooltip CSS
internal/database/migrations/
├── 000007_add_tile_fields.up.sql    # New: icon/tile_size to plugins, display_order to user_plugin_configs
└── 000007_add_tile_fields.down.sql
plugins/daily-news-digest/
└── plugin.yaml              # Extend: add icon and tile_size fields
```

The `internal/briefings/handlers.go` file stays unchanged — it handles the legacy single-briefing flow. The new `internal/dashboard/` package owns all tile dashboard logic.

### Pattern 1: CSS Grid with Variable Tile Sizes

**What:** A single CSS Grid container using `auto-fit/minmax`. Individual tiles use `style` to inject `grid-column: span N; grid-row: span N` computed from plugin YAML metadata.

**When to use:** Any time you need N-column tiles with auto-reflow. The grid handles all responsive wrapping.

**Example (CSS in liquid-glass.css):**
```css
/* Tile grid container */
.tile-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 1.25rem;
  align-items: start;        /* tiles don't stretch to row height */
}

/* Tile size modifiers via data attributes */
[data-tile-size="2x1"] {
  grid-column: span 2;
}
[data-tile-size="2x2"] {
  grid-column: span 2;
  grid-row: span 2;
}
/* 1x1 is default (no extra rules needed) */

/* Expanded tile takes full row */
.tile-card.tile-expanded {
  grid-column: 1 / -1;       /* span all columns */
}

/* Mobile: all tiles single column */
@media (max-width: 640px) {
  [data-tile-size="2x1"],
  [data-tile-size="2x2"] {
    grid-column: span 1;
    grid-row: span 1;
  }
}
```

**Example (Templ component):**
```go
// Source: templ.guide/syntax-and-usage/css-style-management/
templ TileCard(tile TileViewModel) {
    <div
        id={ fmt.Sprintf("tile-%d", tile.PluginID) }
        class="glass-card tile-card"
        data-tile-size={ tile.TileSize }
        data-plugin-id={ fmt.Sprintf("%d", tile.PluginID) }
    >
        @TileContent(tile)
    </div>
}
```

### Pattern 2: View-Model Struct for Dashboard Data

**What:** A single Go struct aggregating plugin config, latest run, and summary into one object passed to the template. Built by a single DB query + Go-side join.

**Why:** The template needs data from multiple tables (UserPluginConfig + Plugin + latest PluginRun). Build a flat view-model in the handler so the template stays logic-free.

```go
// internal/dashboard/viewmodel.go
type TileViewModel struct {
    PluginID      uint
    PluginName    string
    PluginIcon    string      // emoji or SVG ref from plugin YAML
    TileSize      string      // "1x1", "2x1", "2x2"
    DisplayOrder  int
    Enabled       bool

    // Latest run data (nil if no runs yet)
    LatestRunID      *string
    LatestRunStatus  string   // pending/processing/completed/failed
    LatestRunAt      *time.Time
    NextRunAt        *time.Time  // computed from cron + timezone
    BriefingSummary  string      // 2-3 line summary from PluginRun.Output
    HasContent       bool        // true if a completed run exists

    // For error overlay: last successful summary (even if current run failed)
    LastSuccessfulSummary string
}
```

### Pattern 3: Latest Plugin Run Per User — DISTINCT ON

**What:** A single PostgreSQL query using `DISTINCT ON (user_id, plugin_id)` to get the most recent PluginRun per user-plugin pair. Avoids N+1 (no per-tile query).

**When to use:** Dashboard page load, and per-tile HTMX polling updates.

**Example (SQL):**
```sql
-- Source: PostgreSQL DISTINCT ON documentation
-- Gets the single most recent plugin_run per (user_id, plugin_id) pair
SELECT DISTINCT ON (pr.user_id, pr.plugin_id)
    pr.id, pr.plugin_run_id, pr.user_id, pr.plugin_id,
    pr.status, pr.output, pr.error_message,
    pr.started_at, pr.completed_at, pr.created_at
FROM plugin_runs pr
WHERE pr.user_id = ? AND pr.deleted_at IS NULL
ORDER BY pr.user_id, pr.plugin_id, pr.created_at DESC
```

**Example (GORM raw scan):**
```go
// Source: gorm.io/docs/sql_builder.html
type LatestPluginRun struct {
    ID           uint
    PluginRunID  string
    PluginID     uint
    Status       string
    Output       datatypes.JSON
    ErrorMessage string
    CompletedAt  *time.Time
    CreatedAt    time.Time
}

var runs []LatestPluginRun
db.Raw(`
    SELECT DISTINCT ON (plugin_id)
        id, plugin_run_id, plugin_id, status, output,
        error_message, completed_at, created_at
    FROM plugin_runs
    WHERE user_id = ? AND deleted_at IS NULL
    ORDER BY plugin_id, created_at DESC
`, userID).Scan(&runs)
```

### Pattern 4: SortableJS + HTMX Drag-to-Reorder

**What:** SortableJS handles the drag interaction; on `end` event HTMX posts the new order to the server. The server persists `display_order` in `user_plugin_configs` and returns a 200 (no body needed).

**When to use:** When user drags tiles to reorder them.

**Example (HTML in dashboard.templ):**
```go
// Source: htmx.org/examples/sortable/
templ TileGrid(tiles []TileViewModel) {
    <div
        class="tile-grid sortable"
        hx-post="/api/tiles/order"
        hx-trigger="end"
        hx-swap="none"
    >
        for _, tile := range tiles {
            <div class="tile-wrapper" data-plugin-id={ fmt.Sprintf("%d", tile.PluginID) }>
                <input type="hidden" name="plugin_id" value={ fmt.Sprintf("%d", tile.PluginID) }/>
                @TileCard(tile)
            </div>
        }
    </div>
}
```

**Example (JavaScript initialization in dashboard.templ or layout.templ):**
```html
<!-- Source: htmx.org/examples/sortable/ -->
<script>
htmx.onLoad(function(content) {
    var sortables = content.querySelectorAll(".sortable");
    for (var i = 0; i < sortables.length; i++) {
        var sortable = sortables[i];
        var sortableInstance = new Sortable(sortable, {
            animation: 150,
            handle: ".tile-drag-handle",  // optional — tile card acts as handle
            onEnd: function(evt) {
                this.option("disabled", true);
            }
        });
        sortable.addEventListener("htmx:afterSwap", function() {
            sortableInstance.option("disabled", false);
        });
    }
});
</script>
```

**Server handler:**
```go
// POST /api/tiles/order
// Body: plugin_id[] in new order (form values)
func UpdateTileOrderHandler(db *gorm.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        userID := getAuthUserID(c, db)
        pluginIDs := c.PostFormArray("plugin_id")
        for i, idStr := range pluginIDs {
            id, _ := strconv.ParseUint(idStr, 10, 64)
            db.Model(&plugins.UserPluginConfig{}).
                Where("user_id = ? AND plugin_id = ?", userID, id).
                Update("display_order", i)
        }
        c.Status(http.StatusOK)
    }
}
```

### Pattern 5: HTMX Polling for Tile Status Updates

**What:** Each tile polls its own status endpoint every 30 seconds while a plugin run is pending/processing. When completed, the response replaces the tile content with new HTML.

**Example:**
```go
// Tile in "generating" state polls itself
templ TileGenerating(tile TileViewModel) {
    <div
        id={ fmt.Sprintf("tile-%d", tile.PluginID) }
        class="glass-card tile-card tile-generating"
        data-tile-size={ tile.TileSize }
        hx-get={ fmt.Sprintf("/api/tiles/%d", tile.PluginID) }
        hx-trigger="every 30s"
        hx-swap="outerHTML"
    >
        // Spinner + last known content (if any)
    </div>
}
```

The endpoint `/api/tiles/:pluginID` returns the full tile HTML (same component), so HTMX replaces the tile in the grid.

### Pattern 6: Fade-In Transition for New Content

**What:** When a tile's content changes (new briefing arrived), the new HTML fades in using the `htmx-added` CSS class.

**Example (CSS in liquid-glass.css):**
```css
/* Source: htmx.org/examples/animations/ */
.tile-card.htmx-added {
  opacity: 0;
}
.tile-card {
  opacity: 1;
  transition: opacity 0.6s var(--ease-out);
}
```

**HTMX swap with settle:**
```html
hx-swap="outerHTML settle:0.6s"
```

### Pattern 7: Time-Aware Greeting in Go

**What:** Compute "Good morning/afternoon/evening, Name" server-side using `time.LoadLocation` and the user's stored IANA timezone.

**Example (Go in dashboard handler):**
```go
// Source: go.dev/src/time package
func timeAwareGreeting(name, timezone string) string {
    loc, err := time.LoadLocation(timezone)
    if err != nil {
        loc = time.UTC
    }
    hour := time.Now().In(loc).Hour()
    switch {
    case hour < 12:
        return "Good morning, " + name
    case hour < 17:
        return "Good afternoon, " + name
    default:
        return "Good evening, " + name
    }
}
```

### Pattern 8: Pure CSS Timing Tooltip

**What:** An info badge (circle icon) on each tile shows timing details on hover via a CSS `::after` tooltip using `data-tooltip` attribute.

**Example (CSS):**
```css
/* Source: CSS tooltip pattern, MDN */
.tile-info-badge {
  position: relative;
  display: inline-flex;
  align-items: center;
  cursor: default;
}

.tile-info-badge[data-tooltip]::after {
  content: attr(data-tooltip);
  position: absolute;
  bottom: calc(100% + 8px);
  left: 50%;
  transform: translateX(-50%);
  background: rgba(42, 31, 24, 0.92);
  color: var(--navbar-text);
  font-size: 0.75rem;
  white-space: nowrap;
  padding: 0.4rem 0.75rem;
  border-radius: var(--radius-sm);
  pointer-events: none;
  opacity: 0;
  transition: opacity 0.2s ease;
  z-index: 10;
}

.tile-info-badge[data-tooltip]:hover::after {
  opacity: 1;
}
```

**Templ usage:**
```go
<span
    class="tile-info-badge"
    data-tooltip={ fmt.Sprintf("Last run: %s · Next: %s", lastRun, nextRun) }
>
    // SVG info icon
</span>
```

### Pattern 9: Skeleton Loading on Initial Page Load

**What:** Render skeleton tile outlines immediately from `UserPluginConfig` data (no content needed), then fetch actual tile content via `hx-trigger="load"`. This shows layout structure instantly.

**Recommendation:** Use skeleton tiles rather than a full-page spinner. The skeleton tiles communicate which plugins are configured, even before briefing content loads.

**Example:**
```go
templ TileSkeleton(tile TileViewModel) {
    <div
        id={ fmt.Sprintf("tile-%d", tile.PluginID) }
        class="glass-card tile-card tile-skeleton"
        data-tile-size={ tile.TileSize }
        hx-get={ fmt.Sprintf("/api/tiles/%d", tile.PluginID) }
        hx-trigger="load"
        hx-swap="outerHTML"
    >
        <div class="skeleton-icon"></div>
        <div class="skeleton-line skeleton-line-title"></div>
        <div class="skeleton-line"></div>
        <div class="skeleton-line skeleton-line-short"></div>
    </div>
}
```

### Anti-Patterns to Avoid

- **Per-tile DB queries in a loop:** Always use the DISTINCT ON batch query to load all plugin runs in one query; never query each tile separately.
- **Storing tile_size as an integer or enum in DB:** Store as a string ("1x1", "2x1", "2x2") matching the YAML so the UI can apply `data-tile-size` directly without a translation step.
- **Using `grid-template-rows: repeat(auto-fill, ...)` for expand-in-place:** The `grid-column: 1 / -1` trick for expanded tiles only works when `grid-auto-flow: dense` is NOT set. Do not set `grid-auto-flow: dense` on the tile grid or expanded tiles will not reflow correctly.
- **Initializing SortableJS outside `htmx.onLoad`:** HTMX replaces DOM elements, so SortableJS must re-initialize after each swap. Always wrap initialization in `htmx.onLoad`.
- **Storing plugin icon as a file path in the DB Plugin model:** The icon field is metadata that lives in YAML; sync it to the DB `Plugin` table during `InitPlugins` (existing upsert pattern already handles this).

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Drag-to-reorder | Custom mouse event handlers | SortableJS 1.15.7 | Touch support, ghost class, scroll handling, disabled state during HTMX swap — all handled |
| Latest-run-per-plugin query | N+1 loop or complex CTE | PostgreSQL `DISTINCT ON` | Single pass, no subquery needed, idiomatic Postgres pattern |
| Hover timing tooltips | JavaScript tooltip library | Pure CSS `::after` + `data-tooltip` | No JS dependency; works for static content; existing design tokens apply |
| Content fade-in | Manual `requestAnimationFrame` | HTMX `htmx-added` CSS class + `settle:Ns` | Built into HTMX swap lifecycle |
| Time-zone greeting | Third-party library | `time.LoadLocation` + `time.Now().In(loc).Hour()` | Go stdlib is sufficient; no additional dependency |

**Key insight:** This phase is primarily a layout and data-pipeline problem, not a JavaScript problem. CSS Grid + SortableJS + HTMX cover all interactivity without custom JavaScript event wiring.

---

## Common Pitfalls

### Pitfall 1: `grid-auto-flow: dense` Breaks Expand-in-Place

**What goes wrong:** If `grid-auto-flow: dense` is set, CSS Grid backfills empty spaces with smaller tiles when a large tile expands to `grid-column: 1 / -1`. Tiles visually jump to unexpected positions.

**Why it happens:** Dense packing is greedy — it reorders tiles to minimize whitespace, ignoring DOM order.

**How to avoid:** Do not set `grid-auto-flow: dense` on `.tile-grid`. Use the default `row` flow. Accept that there may be gaps on the last row; this is expected behavior.

**Warning signs:** After expanding a tile, other tiles visually rearrange in unexpected order.

### Pitfall 2: SortableJS Breaks After HTMX Swap

**What goes wrong:** After HTMX swaps tile content (e.g., a polling update replaces a tile), the SortableJS instance bound to the old DOM element becomes stale. Drag stops working.

**Why it happens:** HTMX replaces DOM nodes; SortableJS holds references to the old nodes.

**How to avoid:** Use `htmx.onLoad` for SortableJS initialization — it runs on every HTMX content swap, re-attaching Sortable to refreshed elements. The official HTMX example shows this exact pattern.

**Warning signs:** Dragging stops responding after the first HTMX tile update.

### Pitfall 3: PluginMetadata KnownFields Validation Rejects New YAML Fields

**What goes wrong:** `LoadPluginMetadata` uses `decoder.KnownFields(true)` which rejects any YAML key not declared in `PluginMetadata`. Adding `icon` and `tile_size` to `plugin.yaml` without adding them to the struct causes startup to fail for all plugins.

**Why it happens:** `decoder.KnownFields(true)` is intentional (catches typos), but it means the struct is the source of truth.

**How to avoid:** Add `Icon string` and `TileSize string` to `PluginMetadata` struct in `internal/plugins/metadata.go` before updating any `plugin.yaml` files. Update `syncPluginToDB` to persist these new fields.

**Warning signs:** Application logs "failed to parse plugin metadata: yaml: unmarshal errors" at startup.

### Pitfall 4: DISTINCT ON Requires Matching ORDER BY

**What goes wrong:** PostgreSQL requires that `ORDER BY` includes all columns listed in `DISTINCT ON` before any other sort columns. Omitting `plugin_id` from `ORDER BY` before `created_at DESC` causes a syntax error.

**Why it happens:** This is a PostgreSQL language rule. DISTINCT ON picks the first row per group per the ORDER BY, so the grouping columns must sort first.

**How to avoid:** Always write: `ORDER BY plugin_id, created_at DESC` (the DISTINCT ON columns first, then the tie-breaker).

**Warning signs:** `ERROR: SELECT DISTINCT ON expressions must match initial ORDER BY expressions`.

### Pitfall 5: display_order NULL Handling During First Load

**What goes wrong:** When a user first enables a plugin, `display_order` is NULL. Sorting by `display_order ASC` puts all NULLs at the end (in PostgreSQL, NULLs sort last by default with ASC), but some users may have mixed NULL and non-NULL values.

**How to avoid:** Use `ORDER BY display_order ASC NULLS LAST` in the dashboard query. Initialize `display_order` to the current count of enabled plugins for that user when they first enable a plugin.

**Warning signs:** New plugins always appear at end regardless of enabled order.

### Pitfall 6: Templ CSS Sanitization Strips grid-column Values

**What goes wrong:** Using `style={ fmt.Sprintf("grid-column: span %d", cols) }` may be sanitized by Templ if it detects unexpected patterns.

**Why it happens:** Templ sanitizes dynamic CSS for security. Standard CSS values like `span 2` are permitted, but complex expressions may be blocked.

**How to avoid:** Use `data-tile-size` HTML attribute + CSS `[data-tile-size="2x1"] { grid-column: span 2; }` instead of inline `style`. This completely avoids dynamic CSS injection and is more maintainable.

---

## Code Examples

### Full Dashboard Query (Handler)

```go
// Source: GORM sql_builder.html + PostgreSQL DISTINCT ON
func getDashboardTiles(db *gorm.DB, userID uint) ([]TileViewModel, error) {
    // Step 1: Get all enabled user plugin configs with plugin metadata
    type UserPluginRow struct {
        PluginID     uint
        PluginName   string
        PluginIcon   string
        TileSize     string
        DisplayOrder *int
        CronExpr     string
        Timezone     string
    }
    var configs []UserPluginRow
    db.Raw(`
        SELECT upc.plugin_id, p.name as plugin_name, p.icon as plugin_icon,
               p.tile_size, upc.display_order, upc.cron_expression, upc.timezone
        FROM user_plugin_configs upc
        JOIN plugins p ON p.id = upc.plugin_id
        WHERE upc.user_id = ? AND upc.enabled = true AND upc.deleted_at IS NULL
        ORDER BY upc.display_order ASC NULLS LAST, p.name ASC
    `, userID).Scan(&configs)

    // Step 2: Get latest plugin run per plugin for this user (single query)
    type LatestRun struct {
        PluginID    uint
        Status      string
        Output      []byte // JSONB
        ErrorMsg    string
        CompletedAt *time.Time
        CreatedAt   time.Time
    }
    var runs []LatestRun
    db.Raw(`
        SELECT DISTINCT ON (plugin_id)
            plugin_id, status, output, error_message, completed_at, created_at
        FROM plugin_runs
        WHERE user_id = ? AND deleted_at IS NULL
        ORDER BY plugin_id, created_at DESC
    `, userID).Scan(&runs)

    // Step 3: Build run map for O(1) lookup
    runMap := make(map[uint]LatestRun, len(runs))
    for _, r := range runs {
        runMap[r.PluginID] = r
    }

    // Step 4: Assemble TileViewModels
    tiles := make([]TileViewModel, 0, len(configs))
    for _, cfg := range configs {
        tile := TileViewModel{
            PluginID:    cfg.PluginID,
            PluginName:  cfg.PluginName,
            PluginIcon:  cfg.PluginIcon,
            TileSize:    cfg.TileSize,
        }
        if run, ok := runMap[cfg.PluginID]; ok {
            tile.LatestRunStatus = run.Status
            tile.LatestRunAt = &run.CreatedAt
            // Parse summary from run.Output JSON
            tile.BriefingSummary = extractSummary(run.Output)
        }
        // Compute next run from cron expression
        tile.NextRunAt = computeNextRun(cfg.CronExpr, cfg.Timezone)
        tiles = append(tiles, tile)
    }
    return tiles, nil
}
```

### Tile Template (Collapsed State)

```go
// Templ component — uses data attribute for grid sizing to avoid CSS sanitization
templ TileCardCollapsed(tile TileViewModel) {
    <div
        id={ fmt.Sprintf("tile-%d", tile.PluginID) }
        class="glass-card tile-card"
        data-tile-size={ tile.TileSize }
        data-plugin-id={ fmt.Sprintf("%d", tile.PluginID) }
        if tile.LatestRunStatus == "processing" || tile.LatestRunStatus == "pending" {
            hx-get={ fmt.Sprintf("/api/tiles/%d", tile.PluginID) }
            hx-trigger="every 30s"
            hx-swap="outerHTML settle:0.6s"
        }
    >
        <div class="tile-header">
            <span class="tile-icon">{ tile.PluginIcon }</span>
            <span class="tile-name">{ tile.PluginName }</span>
            <span class="tile-status-icon">
                if tile.LatestRunStatus == "processing" || tile.LatestRunStatus == "pending" {
                    <div class="glass-spinner glass-spinner-sm"></div>
                } else if tile.LatestRunStatus == "failed" {
                    // Error SVG icon (inline)
                }
            </span>
            <span
                class="tile-info-badge"
                data-tooltip={ formatTimingTooltip(tile) }
            >
                // Info SVG icon (inline)
            </span>
        </div>
        <div class="tile-summary">
            if tile.BriefingSummary != "" {
                <p class="tile-summary-text">{ tile.BriefingSummary }</p>
            } else if tile.NextRunAt != nil {
                <p class="tile-waiting">
                    Your first briefing is scheduled for { formatRelativeTime(*tile.NextRunAt) }
                </p>
            }
        </div>
    </div>
}
```

### SortableJS Order Endpoint

```go
// POST /api/tiles/order — no response body needed
func UpdateTileOrderHandler(db *gorm.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        userID := mustGetUserID(c, db)
        pluginIDs := c.PostFormArray("plugin_id")
        for i, idStr := range pluginIDs {
            pluginID, err := strconv.ParseUint(idStr, 10, 64)
            if err != nil {
                continue
            }
            db.Model(&plugins.UserPluginConfig{}).
                Where("user_id = ? AND plugin_id = ? AND deleted_at IS NULL",
                    userID, pluginID).
                Update("display_order", i)
        }
        c.Status(http.StatusOK)
    }
}
```

---

## Schema Changes Required

These are the database and model additions needed for this phase:

### Migration 000007: Add tile fields to plugins, display_order to user_plugin_configs

```sql
-- 000007_add_tile_fields.up.sql
ALTER TABLE plugins
    ADD COLUMN icon VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN tile_size VARCHAR(10) NOT NULL DEFAULT '1x1';

ALTER TABLE user_plugin_configs
    ADD COLUMN display_order INTEGER;

-- Index for dashboard query ordering
CREATE INDEX idx_user_plugin_configs_order
    ON user_plugin_configs(user_id, display_order)
    WHERE deleted_at IS NULL AND enabled = true;
```

```sql
-- 000007_add_tile_fields.down.sql
DROP INDEX IF EXISTS idx_user_plugin_configs_order;
ALTER TABLE plugins DROP COLUMN IF EXISTS icon;
ALTER TABLE plugins DROP COLUMN IF EXISTS tile_size;
ALTER TABLE user_plugin_configs DROP COLUMN IF EXISTS display_order;
```

### PluginRun Output — Summary Field

The `PluginRun.Output` column is already JSONB. The CrewAI sidecar must be updated to include a `summary` key alongside the full briefing content:

```json
{
  "summary": "2-3 sentence preview...",
  "content": "Full briefing markdown..."
}
```

The Go side reads `summary` from the JSONB `Output` field — no schema change needed, just a Go struct for parsing:

```go
type PluginRunOutput struct {
    Summary string `json:"summary"`
    Content string `json:"content"`
}
```

### PluginMetadata Struct (internal/plugins/metadata.go)

```go
type PluginMetadata struct {
    Name               string                 `yaml:"name"`
    Description        string                 `yaml:"description"`
    Owner              string                 `yaml:"owner"`
    Version            string                 `yaml:"version"`
    SchemaVersion      string                 `yaml:"schema_version"`
    Capabilities       []string               `yaml:"capabilities"`
    DefaultConfig      map[string]interface{} `yaml:"default_config"`
    SettingsSchemaPath string                 `yaml:"settings_schema_path"`
    Icon               string                 `yaml:"icon"`      // NEW: emoji or SVG ref
    TileSize           string                 `yaml:"tile_size"` // NEW: "1x1", "2x1", "2x2"
}
```

### Plugin DB Model (internal/plugins/models.go)

```go
type Plugin struct {
    // ... existing fields ...
    Icon     string `gorm:"column:icon;not null;default:''"`      // NEW
    TileSize string `gorm:"column:tile_size;not null;default:'1x1'"` // NEW
}
```

### plugin.yaml (daily-news-digest)

```yaml
name: daily-news-digest
description: Generates a personalized daily news digest
icon: "📰"         # NEW
tile_size: "2x1"   # NEW — 2 columns wide, 1 row tall
# ... rest of existing fields ...
```

---

## State of the Art

| Old Approach | Current Approach | Impact |
|--------------|------------------|--------|
| Single latest briefing on dashboard | CSS Grid tiles per plugin | Each plugin gets independent card with its own status |
| Full-page spinner on load | Skeleton tiles with `hx-trigger="load"` | Users see configured plugins immediately |
| `DashboardPage(name, email, *Briefing)` | `DashboardPage(name, greeting, []TileViewModel)` | Template receives pre-built view models |
| No drag ordering | SortableJS + `display_order` column | User-controlled tile layout persisted to DB |

**Deprecated in this phase:**
- `templates.DashboardPage(name, email, latestBriefingPtr)` — signature changes; old signature must not be used after this phase
- Dashboard route handler in `main.go` inline lambda — move to `internal/dashboard/handlers.go` for consistency with briefings package pattern

---

## Open Questions

1. **CrewAI sidecar output format**
   - What we know: `PluginRun.Output` is JSONB; current format is opaque (the `output` field in `streams.PluginResult` is a string)
   - What's unclear: Does the sidecar currently store a `summary` key, or only raw content? How is `Output` JSONB structured today?
   - Recommendation: Inspect a real `PluginRun.output` record in the dev DB. If no summary field exists, add it to the sidecar's output format as part of this phase.

2. **Expand-in-place — full-width vs proportional**
   - What we know: `grid-column: 1 / -1` gives full-width expansion; CSS Grid handles reflow automatically
   - What's unclear: Should a 2x1 tile expand to full-width or to a larger proportional size (e.g., 3x2)?
   - Recommendation: Use `grid-column: 1 / -1` (full-width) for the expanded state. It's the simplest to implement with CSS Grid and provides the most reading space for full briefing content.

3. **Tile polling — all tiles vs generating-only tiles**
   - What we know: 30s poll is the locked decision; question is whether to poll from a dashboard-level endpoint or per-tile
   - What's unclear: Per-tile polling (each tile polls its own endpoint) avoids sending unnecessary data when tiles are idle. Dashboard-level polling (one endpoint returns all tiles) is simpler but sends a full re-render on every poll.
   - Recommendation: Per-tile polling on tiles that are `pending/processing` status only. Idle tiles (status=completed) do not poll. This is the pattern the existing `BriefingCard` uses (stops polling after completion).

---

## Sources

### Primary (HIGH confidence)
- `internal/plugins/metadata.go` — `PluginMetadata` struct with `KnownFields(true)` constraint
- `internal/plugins/models.go` — `Plugin`, `UserPluginConfig`, `PluginRun` DB models
- `internal/database/migrations/` — Migration naming convention (000007 is next)
- `internal/templates/briefing.templ` — Existing HTMX polling pattern (`hx-trigger="every 2s"`, `hx-swap="outerHTML"`)
- `internal/templates/dashboard.templ` — Current dashboard structure being replaced
- `static/css/liquid-glass.css` — CSS custom properties and class names to extend
- `go.mod` — Confirmed: no SortableJS; HTMX 2.0.0 via CDN; templ v0.3.977
- [htmx.org/examples/sortable/](https://htmx.org/examples/sortable/) — Official SortableJS + HTMX integration pattern
- [htmx.org/examples/animations/](https://htmx.org/examples/animations/) — `htmx-added` / `htmx-swapping` CSS classes for fade transitions
- [templ.guide/syntax-and-usage/css-style-management/](https://templ.guide/syntax-and-usage/css-style-management/) — Dynamic inline styles via `map[string]string` and `templ.SafeCSS`
- [github.com/SortableJS/Sortable/releases](https://github.com/SortableJS/Sortable/releases) — Latest version: 1.15.7

### Secondary (MEDIUM confidence)
- [gorm.io/docs/advanced_query.html](https://gorm.io/docs/advanced_query.html) — GORM `Table("(?) as u", subquery)` and `Raw().Scan()` patterns
- [gorm.io/docs/sql_builder.html](https://gorm.io/docs/sql_builder.html) — Raw SQL execution with bind variables
- PostgreSQL DISTINCT ON documentation — verified via multiple sources (neon.com PostgreSQL docs, official docs URL pattern)

### Tertiary (LOW confidence, flag for validation)
- Open question 1 (sidecar output format) — cannot verify without inspecting live DB; flagged above

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all libraries are in use or well-documented with current official sources
- Architecture: HIGH — codebase patterns are clear; GORM raw scan, Templ inline styles, HTMX polling are all verified
- Schema changes: HIGH — existing migration pattern is unambiguous; new columns are straightforward
- Pitfalls: HIGH — KnownFields trap and SortableJS/HTMX lifecycle trap both directly derived from codebase inspection
- Open questions: LOW — sidecar output format cannot be determined without DB inspection

**Research date:** 2026-02-21
**Valid until:** 2026-03-21 (stable libraries; SortableJS and HTMX rarely break APIs)
