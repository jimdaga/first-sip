# Phase 12: Dynamic Settings UI - Context

**Gathered:** 2026-02-22
**Status:** Ready for planning

<domain>
## Phase Boundary

Settings page with plugin management, auto-generated forms from JSON Schema, and validation. Users can enable/disable plugins, configure each plugin via dynamically generated forms, and see plugin status. Manual form coding is not allowed — forms must generate from JSON Schema. Account tiers and payment are separate phases.

</domain>

<decisions>
## Implementation Decisions

### Page layout & plugin list
- Collapsed accordion layout — each plugin is a row that expands to reveal settings form and status
- Collapsed row shows: plugin name, enable/disable toggle, and a colored status indicator dot (green = healthy, yellow = warning, red = errors)
- Enable/disable toggle works instantly via HTMX (no save button for toggle)
- Enabling a plugin with required settings auto-expands the accordion to show the settings form

### Form presentation
- Help text shown via tooltip on hover/icon next to field labels — keeps forms compact
- Save button per plugin form within the expanded accordion

### Claude's Discretion (Form presentation)
- Field label sourcing strategy (schema 'title' vs humanized key — pick best approach based on what schemas provide)
- Boolean field rendering (toggle switch vs checkbox — pick what fits glass design system)
- Enum field rendering (dropdown vs radio buttons — pick based on option count)

### Validation & error feedback
- Validate on blur (individual fields) and on submit (full form) — both server-side via HTMX
- Inline errors only, no top-of-form error summary
- On successful save, the Save button briefly changes to "Saved ✓" then returns to normal
- Form preserves user's input on validation errors (re-render with submitted values, not defaults)

### Plugin status display
- Detailed status section inside the expanded accordion (alongside settings form)
- Shows: last run time, next run time, and recent error list (last 3-5 errors with timestamps and messages)
- "Run Now" button in expanded view to manually trigger a plugin run

</decisions>

<specifics>
## Specific Ideas

- Status indicator dot in collapsed row provides at-a-glance health without expanding
- Auto-expand on enable for plugins with required settings prevents silent misconfiguration
- "Saved ✓" inline confirmation keeps feedback close to the action without disruptive toast notifications
- Recent error list (not just count) lets users diagnose issues without leaving settings

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 12-dynamic-settings-ui*
*Context gathered: 2026-02-22*
