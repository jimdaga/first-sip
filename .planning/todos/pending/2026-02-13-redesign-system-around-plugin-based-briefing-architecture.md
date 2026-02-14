---
created: 2026-02-13T14:34:07.011Z
title: Redesign system around plugin-based briefing architecture
area: planning
files: []
---

## Problem

The current system treats briefings as a monolithic feature with a single generation flow, manual "Generate Summary" action, and global scheduling. This limits extensibility — adding new briefing types requires modifying core code rather than plugging in independent modules. The architecture needs to support multiple independent briefing types, per-user configuration, tier-based constraints, and a tile-based dashboard.

## Solution

Major architectural redesign with 6 pillars:

### 1. Core Plugin Model
- Each briefing type = independent plugin
- Plugin generates its own briefing records in the database
- Plugin owns its own backend AI workflows (bundled within the plugin) — **using CrewAI (NOT n8n)**
- AI workflows implemented as CrewAI agents/tasks in Python, triggered by Go scheduler via internal HTTP (FastAPI sidecar)
- CrewAI chosen for: easiest workflow authoring, large community, code-first (workflows are versionable Python code)
- YAML metadata file per plugin: name, description, owner, version, required capabilities, default configuration
- Settings schema/template defining configurable options (schedule time, frequency, plugin-specific inputs)

### 2. User Configuration
- Users enable/disable plugins individually
- Per-plugin user settings: schedule time, frequency, plugin-specific inputs
- Settings UI dynamically generated from the plugin's settings schema

### 3. Execution Model
- Remove manual "Generate Summary" action entirely
- Plugins execute only on their configured schedule
- Centralized scheduler triggers enabled plugins per user configuration
- Generation frequency may be constrained by account tier

### 4. Homepage Redesign (Tile-Based UI)
- Tile-based layout replacing current dashboard
- Each enabled plugin = its own tile showing: latest briefing, plugin name, status (last run, next run)
- Tile visibility configurable per plugin
- Plugins may optionally provide a dedicated detail page (via sidebar or tile click)

### 5. Account Tier Constraints (Future)
- Free tier: limited enabled plugins (e.g., 3), limited generation frequency
- Paid tiers: higher plugin limits (e.g., 10+), more frequent scheduled runs
- Architecture must support enforcing constraints at the platform level

### 6. Plugin Management Dashboard
- Settings > "Plugin Dashboard" listing all available plugins
- Shows: enabled/disabled status, version, short description
- Toggle to enable/disable, link to deeper configuration page
- Basic status info (last run, errors, etc.)
