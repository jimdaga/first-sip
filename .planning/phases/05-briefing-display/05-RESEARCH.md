# Phase 5: Briefing Display - Research

**Researched:** 2026-02-12
**Domain:** Mobile-responsive dashboard UI with DaisyUI, Tailwind CSS, HTMX, and Go Templ
**Confidence:** HIGH

## Summary

Phase 5 extends the existing briefing card (built in Phase 4) to display briefing content in distinct, mobile-friendly sections with read/unread state management. The codebase already has the foundational pieces: a working BriefingCard component with News/Weather/Work sections, DaisyUI styling via CDN, HTMX for interactivity, and Templ templates. This phase focuses on visual organization improvements, responsive layout enhancements, and adding click-to-mark-read functionality.

The existing implementation already displays briefings in distinct sections (News/Weather/Work) and uses DaisyUI components consistently. The primary work is: (1) enhancing mobile responsiveness with Tailwind's mobile-first patterns, (2) adding visual indicators for read/unread state using DaisyUI badges, and (3) implementing click-to-mark-read using HTMX's hx-post pattern.

**Primary recommendation:** Enhance the existing BriefingCard component with mobile-responsive container patterns, add badge-based read/unread indicators, and implement click-to-mark-read via HTMX hx-post targeting individual briefing cards.

## Standard Stack

### Core (Already in Use)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| DaisyUI | 4.x (CDN) | Component library | CSS-only, no JS overhead, mobile-friendly by default |
| Tailwind CSS | Latest (CDN) | Utility-first CSS | Mobile-first breakpoint system, zero media query boilerplate |
| HTMX | 2.0.0 | HTML-based interactivity | Hypermedia-driven state updates, 14KB, no build step |
| Templ | 0.3.977 | Type-safe Go templates | Compile-time checked, native Go integration |

### Supporting (Already in Use)
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| Gin | 1.11.0 | HTTP router | Serving HTML fragments for HTMX |
| GORM | Latest | ORM | Updating Briefing.ReadAt field |

### No New Dependencies Required
Phase 5 requires NO new libraries. All features can be built with the existing stack.

**Installation:**
N/A - All dependencies already installed

## Architecture Patterns

### Recommended Enhancement Pattern for BriefingCard
```
internal/
├── briefings/
│   ├── templates.templ          # Update BriefingCard with responsive classes
│   └── handlers.go              # Add MarkBriefingReadHandler
```

### Pattern 1: Mobile-First Responsive Layout
**What:** Start with mobile layout as base, add responsive prefixes only where layout changes
**When to use:** Always - Tailwind's mobile-first approach
**Example:**
```templ
// Current: Basic card
<div class="card bg-base-100 shadow-xl">
  <div class="card-body">
    <!-- content -->
  </div>
</div>

// Enhanced: Mobile-first responsive
<div class="container mx-auto p-4 md:p-8 max-w-4xl">
  <div class="card bg-base-100 shadow-xl">
    <div class="card-body p-4 md:p-6">
      <div class="space-y-4 md:space-y-6">
        <!-- sections -->
      </div>
    </div>
  </div>
</div>
```
**Source:** [Tailwind Responsive Design](https://tailwindcss.com/docs/responsive-design), [DaisyUI Best Practices](https://www.builder.io/blog/daisyui-best-practices-ai)

### Pattern 2: Click-to-Mark-Read with HTMX
**What:** Use hx-post to update read state, return updated card HTML
**When to use:** State updates triggered by user clicks
**Example:**
```templ
templ BriefingCard(briefing models.Briefing) {
  <div
    id={ fmt.Sprintf("briefing-%d", briefing.ID) }
    class="card bg-base-100 shadow-xl cursor-pointer hover:shadow-2xl transition-shadow"
    hx-post={ fmt.Sprintf("/api/briefings/%d/read", briefing.ID) }
    hx-target={ fmt.Sprintf("#briefing-%d", briefing.ID) }
    hx-swap="outerHTML"
  >
    <!-- card content -->
  </div>
}
```
**Source:** [HTMX Click-to-Load Pattern](https://htmx.org/examples/click-to-load/), [HTMX hx-post examples](https://htmx.org/attributes/hx-post/)

### Pattern 3: Read/Unread Badge Indicator
**What:** Use DaisyUI badge with color classes for status
**When to use:** Visual status indicators
**Example:**
```templ
<div class="flex items-center justify-between mb-4">
  <h2 class="card-title">Daily Briefing</h2>
  if briefing.ReadAt == nil {
    <span class="badge badge-error">Unread</span>
  } else {
    <span class="badge badge-success">Read</span>
  }
</div>
```
**Source:** [DaisyUI Badge Component](https://daisyui.com/components/badge/)

### Pattern 4: Text Truncation for Mobile
**What:** Use Tailwind truncate/line-clamp for long text on small screens
**When to use:** News summaries, long event titles
**Example:**
```templ
<p class="text-sm text-gray-600 mt-1 line-clamp-2 md:line-clamp-none">
  { item.Summary }
</p>
```
**Source:** [Tailwind Text Overflow](https://tailwindcss.com/docs/text-overflow)

### Anti-Patterns to Avoid
- **Full-page postbacks**: Don't redirect or reload entire page on mark-read - use HTMX selective replacement
- **Client-side state management**: Don't add Alpine.js or other JS frameworks - let server manage read/unread state
- **Breaking mobile-first**: Don't write `sm:` prefix for mobile - unprefixed utilities are mobile, prefixed are larger screens
- **outerHTML on wrong target**: Use `hx-target="this"` or specific ID, not body, to avoid replacing unrelated content

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Responsive breakpoints | Custom media queries | Tailwind's sm:/md:/lg: prefixes | Consistent, mobile-first, zero boilerplate |
| Status badges | Custom CSS for colored dots/labels | DaisyUI badge-success/badge-error | Accessible, themed, works with dark mode |
| Click-to-update state | JavaScript event listeners + fetch | HTMX hx-post + hx-target | Declarative, works without JS frameworks |
| Loading states during reads | Custom spinners + JS | HTMX hx-indicator (if needed) | Built-in, no custom code |
| Mobile card spacing | Custom container queries | Tailwind space-y-{n} + responsive variants | Standard, predictable, maintainable |

**Key insight:** The existing stack (DaisyUI + Tailwind + HTMX) handles ALL Phase 5 requirements without custom CSS, custom JavaScript, or new dependencies.

## Common Pitfalls

### Pitfall 1: Polling Continues on Read State Updates
**What goes wrong:** If BriefingCard has `hx-trigger="every 2s"` in completed state, adding `hx-post` for mark-read could create conflicting triggers
**Why it happens:** Mixing polling triggers with click triggers without understanding HTMX trigger hierarchy
**How to avoid:** Ensure completed briefings have NO `hx-trigger="every 2s"` (Phase 4 already does this correctly). Click events with `hx-post` work independently.
**Warning signs:** Browser console shows continuous polling even after briefing is completed and read

### Pitfall 2: Mobile Viewport Not Set
**What goes wrong:** Responsive breakpoints don't activate on mobile devices
**Why it happens:** Missing viewport meta tag in HTML head
**How to avoid:** Verify `internal/templates/layout.templ` has `<meta name="viewport" content="width=device-width, initial-scale=1"/>` (already present in current codebase)
**Warning signs:** Mobile browser shows desktop layout zoomed out

### Pitfall 3: Using sm: for Mobile Styles
**What goes wrong:** Styles don't appear on mobile devices
**Why it happens:** Misunderstanding Tailwind's mobile-first approach - `sm:` means "640px and up", not "small screens"
**How to avoid:** Use unprefixed utilities for mobile, add `md:`/`lg:` for larger screens
**Warning signs:** Layout looks broken on mobile but fine on desktop

### Pitfall 4: Replacing Wrong Target with hx-swap
**What goes wrong:** Entire page or wrong element gets replaced when marking read
**Why it happens:** Using `hx-target="body"` or no target (defaults to triggering element's parent)
**How to avoid:** Use `hx-target="this"` for self-updating elements, or target specific ID: `hx-target="#briefing-123"`
**Warning signs:** Dashboard navigation disappears after marking briefing as read

### Pitfall 5: Not Handling Already-Read Clicks
**What goes wrong:** User clicks a read briefing, backend returns error or duplicate update
**Why it happens:** Handler doesn't check if briefing is already read before updating
**How to avoid:** In MarkBriefingReadHandler, check if `briefing.ReadAt != nil` - if already read, just return the current card without DB update (idempotent)
**Warning signs:** Console errors on repeated clicks, or ReadAt timestamp keeps changing

### Pitfall 6: Large Content Causes Mobile Overflow
**What goes wrong:** Long news summaries or many work items cause horizontal scroll on mobile
**Why it happens:** No text truncation or max-width constraints
**How to avoid:** Add `max-w-full overflow-hidden` to card containers, use `line-clamp-N` or `truncate` on text
**Warning signs:** Horizontal scrollbar appears on mobile, content extends beyond screen width

## Code Examples

Verified patterns from official sources and current codebase:

### Mobile-Responsive Container (Enhance Dashboard)
```templ
// Source: internal/templates/dashboard.templ (current)
// Enhancement: Add responsive padding and max-width
templ DashboardPage(name string, email string, latestBriefing *models.Briefing) {
  @Layout("Dashboard - First Sip") {
    <!-- navbar unchanged -->
    <div class="container mx-auto p-4 md:p-8 max-w-4xl">
      <h2 class="text-2xl md:text-3xl font-bold mb-4 md:mb-6">Welcome, { name }</h2>
      <div class="mb-4">
        <button class="btn btn-primary w-full md:w-auto" hx-post="/api/briefings" hx-target="#briefing-area" hx-swap="outerHTML">
          Generate Daily Summary
        </button>
      </div>
      if latestBriefing != nil {
        @briefings.BriefingCard(*latestBriefing)
      } else {
        <div id="briefing-area" class="text-gray-500 text-center md:text-left">
          No briefings yet. Click Generate to create your first one.
        </div>
      }
    </div>
  }
}
```

### BriefingCard with Read/Unread Badge and Click-to-Mark-Read
```templ
// Source: internal/briefings/templates.templ (enhancement)
templ BriefingCard(briefing models.Briefing) {
  <div id="briefing-area">
    if briefing.Status == models.BriefingStatusPending || briefing.Status == models.BriefingStatusProcessing {
      <!-- Polling state unchanged from Phase 4 -->
      <div class="card bg-base-100 shadow-xl" hx-get={ fmt.Sprintf("/api/briefings/%d/status", briefing.ID) } hx-trigger="every 2s" hx-swap="outerHTML">
        <div class="card-body">
          <div class="flex items-center gap-4">
            <span class="loading loading-spinner loading-md"></span>
            <span>Generating your briefing...</span>
          </div>
        </div>
      </div>
    } else if briefing.Status == models.BriefingStatusCompleted {
      <!-- Enhanced: Add click-to-mark-read, badge, responsive spacing -->
      <div
        class="card bg-base-100 shadow-xl cursor-pointer hover:shadow-2xl transition-shadow"
        hx-post={ fmt.Sprintf("/api/briefings/%d/read", briefing.ID) }
        hx-target="#briefing-area"
        hx-swap="outerHTML"
      >
        <div class="card-body p-4 md:p-6">
          <div class="flex items-center justify-between mb-4">
            <h2 class="card-title text-lg md:text-xl">Daily Briefing</h2>
            if briefing.ReadAt == nil {
              <span class="badge badge-error">Unread</span>
            } else {
              <span class="badge badge-success">Read</span>
            }
          </div>
          @BriefingContentView(briefing)
        </div>
      </div>
    } else if briefing.Status == models.BriefingStatusFailed {
      <!-- Failed state unchanged from Phase 4 -->
    }
  </div>
}
```

### BriefingContentView with Mobile-Responsive Sections
```templ
// Source: internal/briefings/templates.templ (enhancement)
templ BriefingContentView(briefing models.Briefing) {
  if len(briefing.Content) > 0 {
    @renderContent(briefing.Content)
  } else {
    <div class="text-gray-500 mt-4">No content available</div>
  }
}

templ renderContent(contentJSON []byte) {
  {{
    var content webhook.BriefingContent
    err := json.Unmarshal(contentJSON, &content)
    if err != nil {
      contentValid := false
      _ = contentValid
    } else {
      contentValid := true
      _ = contentValid
    }
  }}
  if err != nil {
    <div class="text-error mt-4">Unable to display briefing content</div>
  } else {
    <div class="space-y-4 md:space-y-6 mt-4">
      <!-- News Section -->
      if len(content.News) > 0 {
        <div class="bg-base-200 p-3 md:p-4 rounded-lg">
          <h4 class="text-base md:text-lg font-semibold mb-2 md:mb-3">📰 News</h4>
          <div class="space-y-3">
            for _, item := range content.News {
              <div class="border-l-4 border-primary pl-3 md:pl-4">
                <a href={ templ.URL(item.URL) } target="_blank" class="font-medium text-primary hover:underline text-sm md:text-base">
                  { item.Title }
                </a>
                <p class="text-xs md:text-sm text-gray-600 mt-1 line-clamp-2 md:line-clamp-3">{ item.Summary }</p>
              </div>
            }
          </div>
        </div>
      }
      <!-- Weather Section -->
      <div class="bg-base-200 p-3 md:p-4 rounded-lg">
        <h4 class="text-base md:text-lg font-semibold mb-2">🌤️ Weather</h4>
        <div class="flex items-center gap-2 text-sm md:text-base">
          <span class="font-medium">{ content.Weather.Location }</span>
          <span>•</span>
          <span>{ fmt.Sprintf("%d°", content.Weather.Temperature) }</span>
          <span>•</span>
          <span>{ content.Weather.Condition }</span>
        </div>
      </div>
      <!-- Work Section -->
      <div class="bg-base-200 p-3 md:p-4 rounded-lg">
        <h4 class="text-base md:text-lg font-semibold mb-2">💼 Work</h4>
        if len(content.Work.TodayEvents) > 0 {
          <h5 class="text-sm md:text-base font-semibold mb-1">Today</h5>
          <ul class="list-disc list-inside space-y-1 text-xs md:text-sm">
            for _, event := range content.Work.TodayEvents {
              <li class="truncate md:whitespace-normal">{ event }</li>
            }
          </ul>
        } else {
          <p class="text-gray-500 text-sm">No events scheduled</p>
        }
        if len(content.Work.TomorrowTasks) > 0 {
          <h5 class="text-sm md:text-base font-semibold mt-3 mb-1">Tomorrow</h5>
          <ul class="list-disc list-inside space-y-1 text-xs md:text-sm">
            for _, task := range content.Work.TomorrowTasks {
              <li class="truncate md:whitespace-normal">{ task }</li>
            }
          </ul>
        }
      </div>
    </div>
  }
}
```

### MarkBriefingReadHandler (New Handler)
```go
// Source: internal/briefings/handlers.go (new function)
// Handler to mark briefing as read
func MarkBriefingReadHandler(db *gorm.DB) gin.HandlerFunc {
  return func(c *gin.Context) {
    // Parse briefing ID from URL parameter
    briefingID := c.Param("id")

    // Query briefing
    var briefing models.Briefing
    if err := db.First(&briefing, briefingID).Error; err != nil {
      c.Status(http.StatusNotFound)
      return
    }

    // Idempotent: Only update if not already read
    if briefing.ReadAt == nil {
      now := time.Now()
      if err := db.Model(&briefing).Update("read_at", now).Error; err != nil {
        c.Header("Content-Type", "text/html")
        c.String(http.StatusInternalServerError, `<div class="alert alert-error">Failed to mark as read</div>`)
        return
      }
      briefing.ReadAt = &now
    }

    // Return updated briefing card
    c.Header("Content-Type", "text/html")
    BriefingCard(briefing).Render(c.Request.Context(), c.Writer)
  }
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| jQuery + AJAX for state updates | HTMX declarative attributes | HTMX 2.0 (2024) | Simpler code, no JS frameworks needed |
| Custom CSS components | Utility-first (Tailwind) + semantic components (DaisyUI) | Tailwind v3+ (2021+) | Faster development, smaller CSS bundles |
| React/Vue for UI state | Server-driven HTML with templ + HTMX | Go templ (2023+) | Type-safe templates, no client-side framework |
| CSS media queries | Tailwind responsive prefixes | Established | Mobile-first by default, no custom breakpoints |

**Deprecated/outdated:**
- DaisyUI v1 class names: v4 is current (as of 2024), but CDN link in codebase uses v4 - no migration needed
- HTMX 1.x patterns: v2.0 (2024) has same core patterns, improved Web Components support (not relevant here)

## Open Questions

1. **Should we show multiple briefings in a list view, or just the latest?**
   - What we know: Current implementation shows only the latest briefing (`Order("created_at DESC").First(&latestBriefing)`)
   - What's unclear: Requirements say "each briefing" should show read/unread, but UI only shows one at a time
   - Recommendation: Start with single-briefing view (Phase 5 scope), defer multi-briefing list to future phase. Success criteria 5 ("User can mark briefing as read by clicking it") can be satisfied with single briefing.

2. **Should marking as read be an explicit button, or implicit on card click?**
   - What we know: Requirements say "User can mark briefing as read by clicking it" (BDISP-03)
   - What's unclear: "Clicking it" could mean clicking anywhere on card, or clicking a specific button
   - Recommendation: Use whole-card click with `cursor-pointer` and hover effect (simpler UX, follows mobile app patterns). Entire card is the clickable area.

3. **Should read briefings be visually de-emphasized (greyed out)?**
   - What we know: Requirements specify read/unread indicator must be visible
   - What's unclear: Whether read briefings should have different visual treatment beyond badge
   - Recommendation: Use badge only for Phase 5. Visual de-emphasis (opacity change, grey background) could be added in future phase if users request it. Keep it simple.

## Sources

### Primary (HIGH confidence)
- Tailwind CSS Official Docs: [Responsive Design](https://tailwindcss.com/docs/responsive-design), [Padding](https://tailwindcss.com/docs/padding), [Text Overflow](https://tailwindcss.com/docs/text-overflow)
- DaisyUI Official Docs: [Card Component](https://daisyui.com/components/card/), [Badge Component](https://daisyui.com/components/badge/), [Layout & Typography](https://daisyui.com/docs/layout-and-typography/)
- HTMX Official Docs: [hx-post](https://htmx.org/attributes/hx-post/), [hx-swap](https://htmx.org/attributes/hx-swap/), [hx-target](https://htmx.org/attributes/hx-target/), [Click-to-Load Example](https://htmx.org/examples/click-to-load/)
- Templ Official Docs: [Introduction](https://templ.guide/), [HTMX Integration](https://templ.guide/server-side-rendering/htmx/)

### Secondary (MEDIUM confidence)
- [DaisyUI Best Practices](https://www.builder.io/blog/daisyui-best-practices-ai) - Anti-patterns to avoid
- [Tailwind Mobile-First Best Practices](https://medium.com/@rameshkannanyt0078/best-practices-for-mobile-responsiveness-with-tailwind-css-5b37e910b91c)
- [HTMX with Go Templ Guide](https://callistaenterprise.se/blogg/teknik/2024/01/08/htmx-with-go-templ/)
- [Building Reactive UIs with Go, Templ, and HTMX](https://medium.com/@iamsiddharths/building-reactive-uis-with-go-templ-and-htmx-a-simpler-path-beyond-spas-17e7dad2c7a2)

### Tertiary (LOW confidence)
- WebSearch results on HTMX patterns - verified against official docs

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All libraries already in use, official docs verified
- Architecture: HIGH - Patterns verified in current codebase and official docs
- Pitfalls: MEDIUM-HIGH - Common patterns from official docs and community guides

**Research date:** 2026-02-12
**Valid until:** ~30 days (stack is stable, DaisyUI and Tailwind have slow release cycles)
