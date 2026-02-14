# First Sip - Design System Rules

## Tech Stack

- **Backend:** Go 1.24, Gin web framework, GORM (PostgreSQL), Asynq (Redis job queue)
- **Templating:** [Templ](https://templ.guide/) — type-safe Go HTML templates (`.templ` files compile to `_templ.go`)
- **Frontend:** HTMX 2.0 for server-driven interactivity, no client-side framework
- **Styling:** Custom CSS design system (`static/css/liquid-glass.css`) + Tailwind CSS via CDN for utilities
- **Fonts:** Google Fonts — Bricolage Grotesque (display), Outfit (body)

## Design System: Liquid Glass

The visual identity is a **glass morphism** aesthetic with warm, coffee-inspired tones. All design tokens are CSS custom properties defined in `static/css/liquid-glass.css`.

### Color Palette

| Token                    | Value                          | Usage                    |
| ------------------------ | ------------------------------ | ------------------------ |
| `--bg-base`              | `#FEF5ED`                      | Page background (warm cream) |
| `--accent`               | `#D4915E`                      | Primary CTA, links, bullet markers |
| `--accent-hover`         | `#C07A48`                      | Hover state for accent   |
| `--accent-glow`          | `rgba(212, 145, 94, 0.3)`     | Button glow shadows      |
| `--text-primary`         | `#2A1F18`                      | Headings, body text (warm charcoal) |
| `--text-secondary`       | `rgba(42, 31, 24, 0.58)`      | Supporting text          |
| `--text-tertiary`        | `rgba(42, 31, 24, 0.35)`      | Placeholder, empty state |
| `--status-unread-text`   | `#C94040`                      | Unread badge, errors     |
| `--status-read-text`     | `#2D7A4A`                      | Read badge (botanical fern green) |
| `--navbar-bg`            | Dark espresso gradient          | Navbar background        |
| `--navbar-text`          | `#FEF5ED`                      | Navbar primary text (cream) |
| `--navbar-text-secondary`| `rgba(254, 245, 237, 0.6)`    | Navbar secondary text    |
| `--navbar-border`        | `rgba(212, 145, 94, 0.15)`    | Navbar bottom border (warm accent) |

### Glass Material Tokens

| Token                    | Value                          | Usage                    |
| ------------------------ | ------------------------------ | ------------------------ |
| `--glass-bg`             | Semi-transparent white gradient | Card backgrounds        |
| `--glass-bg-heavy`       | Heavier white gradient          | Login card, elevated panels |
| `--glass-border`         | `rgba(255,255,255,0.35)`       | Card borders             |
| `--glass-border-subtle`  | `rgba(255,255,255,0.18)`       | Subtle dividers          |
| `--glass-blur`           | `24px`                          | Backdrop blur radius     |
| `--glass-saturate`       | `180%`                          | Backdrop saturation      |

### Border Radius

| Token          | Value   | Usage                            |
| -------------- | ------- | -------------------------------- |
| `--radius-xl`  | `28px`  | Large containers                 |
| `--radius-lg`  | `20px`  | Cards (`.glass-card`)            |
| `--radius-md`  | `14px`  | Buttons, alerts, inner panels    |
| `--radius-sm`  | `10px`  | Small elements                   |

### Typography

| Token            | Font Family           | Weights    | Usage                       |
| ---------------- | --------------------- | ---------- | --------------------------- |
| `--font-display` | Bricolage Grotesque   | 400/600/700 | Headings, titles, branding |
| `--font-body`    | Outfit                | 300/400/500/600 | Body text, UI elements |

### Motion

| Token           | Value                              | Usage             |
| --------------- | ---------------------------------- | ----------------- |
| `--ease-spring` | `cubic-bezier(0.16, 1, 0.3, 1)`  | Interactive transitions, hover effects |
| `--ease-out`    | `cubic-bezier(0.33, 1, 0.68, 1)` | General easing    |

## Component Classes

All UI components use the `glass-` prefix. Always use these classes instead of creating new ones:

### Containers
- **`.glass-card`** — Primary container with backdrop blur, hover lift effect, and warm edge glow
- **`.glass-card-body`** — Inner padding wrapper for card content
- **`.glass-inner`** — Nested panel within a card (e.g., briefing sections)
- **`.glass-navbar`** — Sticky top navigation bar

### Buttons
- **`.glass-btn`** — Base button (always include)
- **`.glass-btn-primary`** — Warm brown CTA with accent glow shadow
- **`.glass-btn-ghost`** — Transparent secondary action button
- **`.glass-btn-google`** — OAuth provider button with glass effect
- **`.glass-btn-sm`** — Small size modifier (combine with other btn classes)

### Feedback
- **`.glass-badge`** — Pill-shaped status indicator (combine with variant)
- **`.glass-badge-unread`** — Red-tinted badge for unread/pending states
- **`.glass-badge-read`** — Green-tinted badge for completed/read states
- **`.glass-alert`** — Alert container (combine with variant)
- **`.glass-alert-error`** — Red-tinted error alert
- **`.glass-spinner`** — Rotating loading indicator (accent-colored)

### Content States
- **`.empty-state`** — Centered placeholder text when no content exists
- **`.content-error`** — Error text for content display failures
- **`.content-empty`** — Subtle text for missing optional content

## Template Architecture

### File Locations

| Path                                  | Purpose                        |
| ------------------------------------- | ------------------------------ |
| `internal/templates/layout.templ`     | Master HTML layout with `{ children... }` slot |
| `internal/templates/login.templ`      | Login page                     |
| `internal/templates/dashboard.templ`  | Dashboard page                 |
| `internal/briefings/templates.templ`  | Briefing card and content components |

### Templ Patterns

**Page templates** wrap content with the Layout component:
```go
templ MyPage() {
    @Layout("Page Title - First Sip") {
        // page content here
    }
}
```

**Component rendering in handlers** uses this pattern:
```go
func render(c *gin.Context, component templ.Component) {
    c.Header("Content-Type", "text/html")
    component.Render(c.Request.Context(), c.Writer)
}
```

**After editing `.templ` files**, regenerate Go code:
```bash
make templ-generate
```

### HTMX Integration

All dynamic interactions use HTMX attributes — no client-side JavaScript routing:
- `hx-post` / `hx-get` — Server endpoints returning HTML fragments
- `hx-target` — DOM element to update (use `#id` selectors)
- `hx-swap="outerHTML"` — Replace entire element (standard swap mode)
- `hx-trigger="every 2s"` — Polling for async status updates
- Fragments are self-contained Templ components (e.g., `BriefingCard`)

## Asset Management

### Static Files
- **CSS:** `static/css/liquid-glass.css`
- **Images:** `static/img/` (logo.png, coffeeicon.png)
- **Served at:** `/static/` via Gin's `r.Static("/static", "./static")`

### External CDN Resources
- Google Fonts: Bricolage Grotesque + Outfit (loaded in layout.templ `<head>`)
- Tailwind CSS: `https://cdn.tailwindcss.com` (utility supplement only)
- HTMX: `https://unpkg.com/htmx.org@2.0.0`

### Icons
- Inline SVG in templates (e.g., Google logo in login, error icon in briefing card)
- No icon library — add SVGs directly in `.templ` files

## Responsive Breakpoints

| Breakpoint          | Target       | Key Changes                    |
| ------------------- | ------------ | ------------------------------ |
| `min-width: 768px`  | Tablet+      | Larger padding, bigger fonts   |
| `max-width: 360px`  | Small phones | Reduced padding, smaller title |

Desktop-first approach: base styles target desktop, media queries adjust for smaller screens.

## Layout Structure

Every page has this DOM structure:
```html
<body>
    <!-- Animated mesh background (4 floating gradient orbs) -->
    <div class="mesh-bg" aria-hidden="true">
        <div class="orb orb-1"></div>
        <div class="orb orb-2"></div>
        <div class="orb orb-3"></div>
        <div class="orb orb-4"></div>
    </div>
    <!-- Page content injected via { children... } -->
</body>
```

Dashboard pages add:
```html
<nav class="glass-navbar">...</nav>
<main class="dashboard-content">...</main>
```

## Build & Development

```bash
make db-up              # Start Postgres + Redis + Asynqmon
make templ-generate     # Compile .templ -> _templ.go
make dev                # Run server with embedded worker
make build              # Production binary
make test               # Run tests with race detector
```

## Key Conventions

1. **Always use CSS custom properties** — never hardcode colors, radii, or font families
2. **Prefix all component classes with `glass-`** to maintain design system consistency
3. **Use Templ for all HTML** — no raw `html/template` or string concatenation
4. **Server-render everything** — HTMX handles interactivity, no client-side state management
5. **Keep animations subtle** — hover lifts (translateY -1px to -3px), spring easing, fadeInUp on page load
6. **Glass cards always need backdrop-filter** — use `.glass-card` class, don't recreate the effect manually
7. **Status colors follow the pattern**: red = unread/error/warning, green = read/success
8. **Max content width is `56rem`** — enforced by `.dashboard-content` and `.navbar-inner`
