# Phase 11: Tile-Based Dashboard - Context

**Gathered:** 2026-02-21
**Status:** Ready for planning

<domain>
## Phase Boundary

CSS Grid tile layout displays all enabled plugins with latest briefing and status. Each plugin renders as a tile showing plugin name, icon, briefing summary, and status. Tiles support expand-in-place to view full briefing. Dashboard auto-polls for updates. Plugin settings management (enable/disable, configuration) belongs to Phase 12.

</domain>

<decisions>
## Implementation Decisions

### Tile content & density
- Plugin icon (from YAML metadata) displayed prominently on each tile
- Plugin name as tile heading
- 2-3 line briefing preview from a dedicated summary field (CrewAI generates summary alongside full briefing)
- Timing info (last run, next scheduled run) in hover-over tooltip on an info badge — not visible as text on tile
- Status indicators: subtle spinner icon when generating, error icon when last run failed (no text)
- Error state: show previous successful briefing content with error icon overlay — don't replace content
- No visible timestamp on tile face — all timing details in hover only

### Tile interactions
- Click tile to expand in-place within the grid — other tiles reflow around it
- One tile expanded at a time — expanding a new tile auto-collapses the previous
- Collapse via close button in expanded view OR clicking outside the expanded content
- No quick actions on tiles (no run-now, no settings link) — those belong in Phase 12 Settings

### Layout & sizing
- Plugin-defined tile sizes using grid units: 1x1, 2x1, 2x2 (declared in plugin YAML metadata)
- User-defined tile ordering with drag-and-drop (SortableJS or similar alongside HTMX)
- Order persists per user in database
- Dashboard header above tiles: time-aware greeting ("Good morning, Jim") + date
- Header is greeting only — no status summary line
- Mobile: single column, same user-defined order, tiles full-width stacked

### Empty & loading states
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

</decisions>

<specifics>
## Specific Ideas

- CrewAI workflow needs to generate a short summary field alongside the full briefing content (update crew task pipeline)
- Plugin YAML metadata needs new fields: `icon` (emoji or SVG reference), `tile_size` (grid units like "1x1", "2x1", "2x2")
- User tile order stored in database (new field or table for display_order per user per plugin)
- Time-aware greeting uses user's timezone from their profile

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 11-tile-based-dashboard*
*Context gathered: 2026-02-21*
