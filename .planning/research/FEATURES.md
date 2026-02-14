# Feature Research

**Domain:** Plugin-Based Briefing Architecture for Personal Dashboard Apps
**Researched:** 2026-02-13
**Confidence:** MEDIUM (based on training data through January 2025, WebSearch unavailable)

## Context

This research covers v1.1 plugin architecture features ONLY. v1.0 already delivered: Google OAuth, briefing generation, HTMX status polling, briefing history, read/unread tracking, and automated daily scheduling.

**v1.1 Additions:**
- Plugin-based briefing architecture
- CrewAI workflow integration (replacing n8n)
- Tile-based dashboard UI
- Per-user per-plugin scheduling
- Dynamic settings UI generation
- Account tier scaffolding

## Feature Landscape by Domain

### 1. Plugin Framework

#### Table Stakes (Users Expect These)

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Enable/disable plugins | "Turn off what I don't need" | LOW | Boolean toggle per user-plugin. Existing pattern: read/unread tracking. |
| Plugin marketplace/directory | "What plugins are available?" | LOW-MEDIUM | List view of all plugins. Metadata from plugin.yaml files. |
| Per-plugin configuration | Each plugin has different needs | MEDIUM | Dynamic forms from JSON schema. Similar to WordPress plugin settings. |
| Plugin status visibility | "Is it working? When's next run?" | LOW | Display last_run_at, next_run_at timestamps. |
| Safe plugin failures | One plugin breaks ≠ everything breaks | MEDIUM | Isolated execution contexts. Asynq handles retry logic. |
| Plugin updates | Plugins improve over time | MEDIUM-HIGH | Version field in metadata. Migration strategy for settings schema changes. |
| Clear plugin permissions | "What data does this access?" | LOW | Metadata field: required_capabilities. Display before enabling. |

#### Differentiators (Competitive Advantage)

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Code-based plugins (not GUI) | Version-controlled, testable workflows | LOW | Plugin = directory with YAML + Go handlers. Differentiates from no-code tools that trap logic. |
| CrewAI integration | Best-in-class AI agent workflows | HIGH | Python sidecar via FastAPI. CrewAI handles multi-agent orchestration. |
| Plugin-owns-workflow model | No central workflow engine to configure | MEDIUM | Each plugin bundles its own CrewAI agents. Reduces coupling. |
| Transparent scheduling | Exact cron visibility, no "magic" | LOW | Users see exact schedule string. Can override per plugin. |
| JSON Schema settings | Self-documenting configuration | MEDIUM | Schema = validation + UI generation + documentation in one. |

#### Anti-Features (Commonly Requested, Often Problematic)

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| Visual plugin builder | "No-code plugin creation!" | Scope creep. Becomes drag-and-drop workflow engine. Hard to version control. | Provide plugin templates in GitHub repo. Documentation for creating plugins. Power users write code. |
| Plugin dependencies | "My plugin needs another plugin's data" | Complexity explosion. Circular dependencies. Version hell. | Plugins are isolated. If shared data needed, refactor into separate service/model. |
| Real-time plugin execution | "Run plugin on demand instantly" | Defeats scheduled batching purpose. Webhook abuse. Rate limiting nightmare. | Schedule-only execution. Manual trigger only for debugging (admin feature). |
| Plugin marketplace install | "One-click install from web UI" | Security nightmare. Arbitrary code execution. Need sandboxing, code review. | Pre-installed plugins only. Enable/disable existing plugins. External plugins = manual deployment. |
| Inter-plugin messaging | "Plugins communicate with each other" | Tight coupling. Event bus complexity. Debugging nightmares. | Plugins write to same database. Read shared data via models. No direct plugin-to-plugin communication. |

### 2. CrewAI Workflow Integration

#### Table Stakes (Users Expect These)

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Reliable workflow execution | "My briefing must arrive" | MEDIUM | Asynq retry policy. Dead letter queue for persistent failures. |
| Workflow status tracking | "Is it processing or stuck?" | LOW | Store execution state in plugin_executions table. |
| Error visibility | "Why did it fail?" | MEDIUM | Surface CrewAI error messages in UI. Not just "failed". |
| Workflow timeouts | Long-running tasks don't block forever | LOW | Asynq task timeout configuration per plugin. |
| Workflow isolation | One user's workflow doesn't affect others | MEDIUM | Separate Python processes per execution or semaphore limiting concurrency. |

#### Differentiators (Competitive Advantage)

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Multi-agent orchestration | CrewAI's hierarchical/sequential task model | HIGH | Researcher → Writer → Reviewer agent chains. Differentiates from single-LLM tools. |
| Workflow-as-code | Python code > GUI workflows | MEDIUM | Version control workflows with git. Test with pytest. No vendor lock-in. |
| Agent memory/context | Agents learn from past briefings | HIGH | CrewAI's memory features. Requires storage backend (Redis or embeddings DB). |
| Custom tools per plugin | Plugins define their own agent tools | MEDIUM | Plugin provides Python functions to CrewAI. Example: GitHub API tool for repo digest. |

#### Anti-Features (Commonly Requested, Often Problematic)

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| GUI workflow editor | "Visual workflow design" | CrewAI is code-first. GUI abstraction loses power. Sync issues between GUI and code. | Provide well-documented Python templates. Example workflows in docs. |
| Workflow sharing marketplace | "Import workflows from community" | Security (arbitrary code). Licensing. Support burden. | GitHub repo of example plugins. Users fork and customize. |
| Real-time workflow debugging | "Step through agent execution" | CrewAI agents run async. Stepping breaks LLM context. | Structured logging to stdout. Post-mortem analysis of execution logs. |

### 3. Tile-Based Dashboard

#### Table Stakes (Users Expect These)

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Responsive grid layout | Works on mobile + desktop | LOW | CSS Grid with auto-fit. Already have responsive design system. |
| Tile for each enabled plugin | "See all my briefings" | LOW | Iterate over user's enabled plugins. Render PluginTile component. |
| Latest briefing preview | "What's new?" without clicking | MEDIUM | Show first N characters or summary field. |
| Visual status indicators | Quick scan of health | LOW | Color-coded badges: green=success, red=error, yellow=pending. |
| Tile click to detail view | "Read more" interaction | LOW | HTMX link to /plugins/:id/briefing. Swap main content area. |
| Empty state | No plugins enabled = clear CTA | LOW | "Enable your first plugin" message with link to plugin directory. |

#### Differentiators (Competitive Advantage)

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Customizable tile order | Drag-and-drop reordering | MEDIUM-HIGH | Requires JS library (SortableJS) or HTMX extension. Persist order in user_plugin_settings. |
| Tile size variants | Important plugins bigger | MEDIUM | CSS Grid span classes. Plugin metadata: preferred_size (small/medium/large). |
| Tile hide/show | Enabled but not displayed on dashboard | LOW | Separate visible flag from enabled. Use case: background plugins (e.g., data sync). |
| Quick actions on tile | Archive, regenerate, share from tile | MEDIUM | HTMX action buttons in tile footer. Inline actions without navigation. |
| Tile data refresh | Update tiles without page reload | LOW | HTMX polling on dashboard. hx-trigger="every 30s" on .plugin-grid. |

#### Anti-Features (Commonly Requested, Often Problematic)

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| Infinite tile customization | "Let me change colors, borders, fonts per tile" | Maintenance explosion. 99% use defaults. CSS conflicts. | Themes at app level (light/dark). No per-tile styling. |
| Real-time live tiles | "Auto-updating tile content" | WebSocket complexity. Battery drain. Briefings are daily, not real-time. | Polling at reasonable interval (30-60s). Or manual refresh button. |
| Tile widgets | "Add charts, graphs, mini-apps to tiles" | Scope creep. Each plugin becomes mini-app. Performance. | Tiles show text summary only. Detail page has rich content. |
| Dashboard layouts | "Multiple dashboard views/tabs" | Complexity. Most users never use. | Single dashboard. Tile order + visibility = enough customization. |

### 4. Per-User Per-Plugin Scheduling

#### Table Stakes (Users Expect These)

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| User-specific schedule time | "6 AM Pacific not 6 AM UTC" | MEDIUM | Store timezone + cron in user_plugin_settings. Already have BriefingTimezone pattern. |
| Per-plugin schedule override | "Daily news at 6 AM, weekly digest on Sundays" | LOW | Plugin has default_schedule, user can override. |
| Timezone support | International users | LOW | Use time.LoadLocation(user.Timezone). Existing pattern. |
| Schedule preview | "When is next run?" | LOW | Calculate next_run_at from cron + timezone. Display on tile. |
| Pause/resume scheduling | "Vacation mode" | LOW | enabled flag already handles this. Toggle off = paused. |

#### Differentiators (Competitive Advantage)

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Cron expression flexibility | "Every weekday at 7 AM" not just daily | LOW | Full cron syntax. More flexible than fixed intervals. |
| Schedule history | "Did it actually run at 6 AM?" | LOW | plugin_executions table with started_at timestamp. |
| Smart retry scheduling | Failed run retries before next scheduled run | MEDIUM | Asynq retry with exponential backoff. Reschedule on success. |
| Schedule templates | "Morning routine" = 6 AM, "Evening digest" = 6 PM | LOW | Predefined cron strings users can select. |

#### Anti-Features (Commonly Requested, Often Problematic)

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| Real-time triggers | "Run when email arrives" or "Run when stock price changes" | Event-driven complexity. WebSocket subscriptions. Not a briefing anymore. | Scheduled checks. Plugin can check conditions and skip if not met. |
| Complex conditional scheduling | "Run only if weather is sunny AND I have meetings" | Workflow engine complexity. Hard to debug. Most users won't use. | Simple cron schedules. Plugin logic handles conditional content. |
| Dynamic schedule learning | "AI learns when I want briefings" | ML complexity. Creepy if wrong. Hard to explain. | User sets schedule explicitly. Can change anytime. |

### 5. Dynamic Settings UI

#### Table Stakes (Users Expect These)

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Form generation from schema | "Configure without code" | MEDIUM | JSON Schema → Templ form fields. Pattern in STACK.md. |
| Validation on submit | "Can't save invalid settings" | MEDIUM | kaptinlin/jsonschema validation. Return errors to form. |
| Default values | "Works out of the box" | LOW | Schema default field. Pre-fill form. |
| Help text / descriptions | "What does this setting do?" | LOW | Schema description field. Render below input. |
| Required vs optional fields | Visual clarity | LOW | Schema required array. Render * for required. |

#### Differentiators (Competitive Advantage)

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Schema extension fields | x-component, x-placeholder for rich UI | MEDIUM | kaptinlin/jsonschema preserves extra fields. Use for UI hints. |
| Conditional fields | "Show API key field only if auth_type=api" | HIGH | JSON Schema if/then/else. Complex to render in Templ. |
| Settings versioning | "Revert to previous settings" | MEDIUM | Store settings_history JSONB array with timestamps. |
| Import/export settings | "Copy settings between plugins or users" | LOW | Download JSON, upload to restore. Useful for template sharing. |

#### Anti-Features (Commonly Requested, Often Problematic)

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| WYSIWYG settings editor | "Visual form builder" | Overkill. Schema is the source of truth. | Schema is code. Edit schema file, auto-generates form. |
| Per-field custom validation logic | "Password must match regex AND not be in breach DB" | Can't express in JSON Schema. Requires custom Go code per field. | JSON Schema handles regex. Custom validation in handler for complex cases. |
| Settings A/B testing | "Test different settings for performance" | Statistical complexity. Most users have one setting set. | Manual settings change. Observe results. |

### 6. Account Tier System

#### Table Stakes (Users Expect These)

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Clear tier limits | "What do I get in free vs paid?" | LOW | Display limits in UI. Store in account_tiers table or config. |
| Enforcement of limits | Can't enable 10 plugins on free tier (max 3) | MEDIUM | Check limit before enabling plugin. Return error message. |
| Upgrade CTA | "Unlock more plugins" | LOW | Show upgrade banner when hitting limits. |
| Tier visibility | "What tier am I on?" | LOW | Display in profile/settings. Badge in navbar optional. |

#### Differentiators (Competitive Advantage)

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Feature flags per tier | Not just limits, but feature access | MEDIUM | example: Free=basic plugins, Pro=AI plugins, Enterprise=custom plugins. |
| Soft limits with grace period | "You're over limit, downgrade or upgrade within 7 days" | MEDIUM | Allow temporary over-limit. Background job checks and notifies. |
| Usage analytics per tier | "You've used 2 of 3 plugins" | LOW | Count enabled plugins. Show progress bar. |
| Trial period | "Try Pro for 14 days" | MEDIUM | Temporary tier upgrade. Background job downgrades on expiry. |

#### Anti-Features (Commonly Requested, Often Problematic)

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| Pay-per-plugin pricing | "Only pay for what I use" | Billing complexity. Cognitive load. Most want unlimited. | Flat tier pricing. Pro = up to 10 plugins. |
| Dynamic tier pricing | "Price based on usage" | Unpredictable costs. User anxiety. Billing edge cases. | Fixed tiers. Overage = upgrade prompt, not surprise charges. |
| Granular feature gating | "Free users can use plugin but only 2x/day" | Confusing limits. Hard to communicate. Implementation complexity. | Tier = plugin count limit. All plugins work same once enabled. |
| Multi-user team tiers | "Shared account for team" | Multi-tenant complexity. Permission system. Seat management. | v2+ feature. v1 = individual accounts only. |

## Feature Dependencies

```
Plugin Framework
    └──requires──> Plugin Metadata (YAML parsing)
    └──requires──> Plugin Registry (database models)
    └──requires──> Dynamic Settings UI
    └──requires──> Per-User Scheduling

CrewAI Integration
    └──requires──> Plugin Framework (workflow belongs to plugin)
    └──requires──> Python FastAPI Sidecar
    └──requires──> Webhook Handlers (sidecar → Go app)

Tile Dashboard
    └──requires──> Plugin Framework (tiles display plugins)
    └──requires──> Briefing Display (existing v1.0)
    └──enhances──> Real-time Status (existing HTMX polling)

Per-User Scheduling
    └──requires──> Asynq Infrastructure (existing v1.0)
    └──requires──> User Model (existing v1.0)
    └──enhances──> Global Scheduling (existing v1.0 pattern)

Dynamic Settings UI
    └──requires──> JSON Schema Validation
    └──requires──> Templ Templates (existing v1.0)
    └──requires──> HTMX Form Handling (existing v1.0)

Account Tiers
    └──requires──> Plugin Framework (limits plugin count)
    └──requires──> User Model (tier field)
    └──conflicts──> Per-Plugin Paid Features (anti-feature: too complex)
```

### Dependency Notes

- **Plugin Framework is foundational:** All other features depend on it. Must be Phase 1.
- **CrewAI requires plugin framework:** Can't integrate workflows without plugin structure.
- **Tile Dashboard requires plugin framework:** Need plugin list to render tiles.
- **Dynamic Settings UI enhances Plugin Framework:** Settings management is part of plugin lifecycle.
- **Account Tiers should be last:** Scaffolding only. Can add limits later without breaking existing features.

## MVP Definition (v1.1)

### Launch With (Core Plugin System)

- [ ] **Plugin metadata loading** — Parse plugin.yaml from filesystem (goccy/go-yaml)
- [ ] **Plugin registry** — Database models (plugins, user_plugin_settings tables)
- [ ] **Plugin enable/disable** — Toggle per user-plugin
- [ ] **Plugin directory page** — List all available plugins with descriptions
- [ ] **Basic tile dashboard** — Grid layout displaying enabled plugins
- [ ] **Per-plugin scheduling** — User can set cron schedule per plugin (Asynq integration)
- [ ] **Dynamic settings form** — Generate form from JSON schema, validate on save
- [ ] **Webhook handler pattern** — Go handler to receive CrewAI workflow results
- [ ] **CrewAI sidecar stub** — FastAPI app that can execute sample Python workflow
- [ ] **Single example plugin** — "Daily News Digest" plugin fully implemented to validate architecture

### Add After Validation (v1.2)

- [ ] **Plugin update mechanism** — Version field, migration strategy for schema changes
- [ ] **Schedule templates** — "Morning routine" / "Evening digest" presets
- [ ] **Tile reordering** — Drag-and-drop or up/down arrows to reorder tiles
- [ ] **Tile visibility toggle** — Hide tile from dashboard without disabling plugin
- [ ] **Plugin execution history** — Archive of past runs with timestamps and status
- [ ] **Settings import/export** — Download/upload JSON for backup or sharing
- [ ] **2-3 additional plugins** — Weather briefing, GitHub activity digest, calendar preview

### Future Consideration (v2+)

- [ ] **Multi-agent CrewAI workflows** — Complex agent chains (researcher → writer → reviewer)
- [ ] **Agent memory** — CrewAI memory backend for context across briefings
- [ ] **Account tier enforcement** — Limit plugin count by tier
- [ ] **Custom plugin development guide** — Documentation + templates for users to create plugins
- [ ] **Plugin permissions system** — Granular capability requirements
- [ ] **Tile size variants** — Small/medium/large tiles
- [ ] **Plugin marketplace** — External plugin repository (big security effort)

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority | Notes |
|---------|------------|---------------------|----------|-------|
| Plugin enable/disable | HIGH | LOW | P1 | Core interaction model |
| Plugin metadata loading | HIGH | LOW | P1 | Foundation for everything |
| Tile dashboard | HIGH | MEDIUM | P1 | Primary UI for v1.1 |
| Per-plugin scheduling | HIGH | MEDIUM | P1 | Differentiator from v1.0 global schedule |
| Dynamic settings form | HIGH | MEDIUM | P1 | Required for plugin configuration |
| Webhook handlers | HIGH | MEDIUM | P1 | How CrewAI sends results |
| CrewAI sidecar integration | HIGH | HIGH | P1 | Core architectural shift |
| Plugin directory UI | MEDIUM | LOW | P1 | Discovery mechanism |
| Single example plugin | HIGH | HIGH | P1 | Validates entire architecture |
| Plugin execution history | MEDIUM | MEDIUM | P2 | Debugging and transparency |
| Schedule templates | LOW | LOW | P2 | Nice UX improvement |
| Tile reordering | MEDIUM | MEDIUM | P2 | Power user feature |
| Settings import/export | LOW | LOW | P2 | Advanced use case |
| Account tier scaffolding | LOW | LOW | P2 | Future-proofing only |
| Multi-agent workflows | MEDIUM | HIGH | P3 | Complex, validate simple agents first |
| Agent memory | MEDIUM | HIGH | P3 | Requires embedding storage |
| Custom plugin guide | MEDIUM | MEDIUM | P3 | After 3-5 built-in plugins exist |
| Plugin marketplace | LOW | VERY HIGH | P3 | Security + support burden |

**Priority key:**
- P1: Must have for v1.1 launch (plugin architecture works end-to-end)
- P2: Should have for v1.2 (polish and power features)
- P3: Nice to have for v2+ (advanced capabilities)

## Competitor Feature Analysis

| Feature | WordPress Plugins | Zapier | n8n | Our Approach |
|---------|-------------------|--------|-----|--------------|
| **Plugin Architecture** | PHP files in /wp-content/plugins/ | SaaS integrations, no user code | Self-hosted workflows (GUI + code) | **Go packages + YAML metadata + CrewAI Python** |
| **Plugin Discovery** | Plugin directory search, one-click install | App directory, OAuth connect | Template library, community sharing | **Pre-installed plugins, enable/disable only** |
| **Configuration UI** | PHP generates HTML forms | OAuth + form fields per integration | GUI node configuration | **JSON Schema → Templ forms** |
| **Execution Model** | Hooks/filters in PHP lifecycle | Cloud queue, visual workflow | Self-hosted queue, n8n worker | **Asynq queue + CrewAI Python sidecar** |
| **Scheduling** | WP-Cron (pseudo-cron, on page load) | Zapier scheduler node | Cron node in workflows | **Per-user Asynq cron schedules** |
| **User Settings** | Per-plugin settings pages | Connection-level settings | Global credentials + node configs | **Per-user per-plugin JSONB settings** |
| **Extensibility** | Anyone can write PHP plugin | Zapier team controls integrations | Users write custom nodes (TS) | **Plugins are code (Go/Python), version controlled** |
| **Multi-tenancy** | Single WP install = one site | Native multi-user SaaS | Single n8n = shared workflows | **Database isolation per user, shared plugins** |

**Our Differentiation:**
1. **Code-first plugins** — Version controlled, testable (WordPress is code-first but messy, Zapier is GUI-only)
2. **AI-native workflows** — CrewAI for multi-agent patterns (competitors don't have native AI orchestration)
3. **Self-hosted + user-owned** — No vendor lock-in (like n8n) but simpler than n8n's GUI complexity
4. **Briefing-specific** — Not general automation tool, optimized for daily digest use case

## User Workflows

### Workflow 1: Enable a Plugin

```
User logs in → Dashboard shows tile grid
User clicks "Manage Plugins" link in navbar
→ Plugin Directory page loads (GET /plugins)
  Displays all available plugins (from plugin registry)
  Each plugin card shows: name, description, enabled status
User clicks "Enable" on "Daily News Digest" plugin
→ HTMX POST /plugins/:id/enable
  Server checks account tier limits (free = max 3 plugins)
  If allowed: create user_plugin_settings record (enabled=true)
  Render settings form from plugin's JSON schema
User fills form (news sources, preferred topics)
User submits form
→ HTMX POST /plugins/:id/settings
  Server validates settings with jsonschema
  Save to user_plugin_settings.settings (JSONB)
  Schedule next execution with Asynq (default cron from plugin.yaml)
  Redirect to dashboard
Dashboard reloads with new tile for "Daily News Digest"
```

### Workflow 2: Configure Plugin Schedule

```
User on Dashboard sees "Daily News Digest" tile
User clicks "Configure" button on tile
→ HTMX GET /plugins/:id/settings
  Loads settings form modal (overlay or sidebar)
  Form shows current settings + schedule field
User changes schedule from "0 6 * * *" (6 AM daily) to "0 7 * * 1-5" (7 AM weekdays)
User saves
→ HTMX POST /plugins/:id/settings
  Validate cron expression
  Update user_plugin_settings.schedule
  Recalculate next_run_at timestamp
  Cancel old Asynq scheduled task
  Enqueue new task with updated schedule
  Return updated tile with "Next run: Mon 7:00 AM PST"
Tile updates in place (HTMX swap)
```

### Workflow 3: View Plugin Briefing

```
Plugin executes on schedule (background)
→ Asynq dequeues task
→ Worker calls CrewAI sidecar (POST http://localhost:8001/plugins/news-digest/execute)
→ FastAPI executes CrewAI workflow (agents fetch news, summarize, format)
→ Sidecar returns JSON briefing data
→ Worker posts to Go app webhook (POST /webhooks/plugin/news-digest)
  Verify HMAC signature
  Create Briefing record in database (user_id, plugin_id, content, created_at)
  Update user_plugin_settings.last_run_at
  Schedule next execution
User refreshes dashboard (or HTMX polling updates tiles)
Tile shows "New briefing" badge
User clicks tile
→ HTMX GET /plugins/:id/briefing/latest
  Load full briefing content (Templ component)
  Swap into main content area (or modal)
  Mark briefing as read
User reads briefing, clicks "Back to Dashboard"
Tile updates to show "Last read: 2 min ago"
```

### Workflow 4: Disable a Plugin

```
User on Dashboard wants to remove clutter
User hovers over tile, clicks "..." menu
User clicks "Disable Plugin"
→ HTMX POST /plugins/:id/disable
  Confirmation modal: "Stop scheduling Daily News Digest?"
User confirms
→ Server updates user_plugin_settings.enabled = false
  Cancel Asynq scheduled tasks for this user+plugin
  Keep historical briefings (don't delete)
  Return empty tile or removed tile (HTMX swap)
Tile disappears from dashboard
Plugin still in directory, can re-enable later
```

## Domain Patterns Observed

### Successful Patterns

- **Plugin isolation:** Each plugin owns its data, schedule, and workflow. No shared state.
- **Metadata-driven UI:** JSON Schema generates forms automatically. Reduces code duplication.
- **Schedule transparency:** Show exact cron expression. Users understand "0 6 * * *" better than "daily at 6 AM" (or provide both).
- **Graceful degradation:** Plugin failure doesn't break dashboard. Show error on tile, other plugins work.
- **Settings versioning:** Keep history of settings changes. Useful for debugging "it worked yesterday".
- **Tile-based dashboards:** Flexible, scannable, responsive. Beats list views for visual information.
- **Enable ≠ Visible:** Some plugins run in background (data sync). Not all need tiles.

### Failed Patterns

- **Plugin dependencies:** "Plugin B needs Plugin A's output" creates coupling hell. Avoid.
- **GUI workflow builders:** Visual programming is appealing but complex. Code is better for power users.
- **Real-time everything:** Briefings are batched. Real-time adds complexity without value.
- **Unlimited customization:** 80% of users use defaults. Don't build for the 1% edge case.
- **Plugin marketplace:** Security and support burden. Pre-installed plugins are safer.
- **Complex scheduling:** "Run on sunny Tuesdays" is cute but not useful. Simple cron wins.

## Known Gaps & Research Needed

**WebSearch unavailable, so these need validation:**

1. **CrewAI 2026 best practices** — Training data through Jan 2025. CrewAI evolving fast. Need to verify:
   - Latest API patterns for multi-agent workflows
   - Memory backend options (Redis vs vector DB)
   - FastAPI sidecar integration examples
   - LangChain vs CrewAI for briefing use case

2. **JSON Schema extension fields** — kaptinlin/jsonschema claims to preserve x-* fields, but need to test:
   - Can we reliably use x-component for UI hints?
   - Do extension fields survive validation round-trip?
   - Are there standard extensions for form generation?

3. **Asynq dynamic scheduling patterns** — Need to verify:
   - Can Asynq handle hundreds of per-user schedules efficiently?
   - Does queue pattern matching (`plugin:*:user:*`) work as expected?
   - What's the performance impact of dynamic queue creation?

4. **HTMX form validation UX** — Pattern for displaying jsonschema validation errors:
   - Inline field errors vs top-of-form error list?
   - HTMX extension for form errors?
   - Templ pattern for error state rendering?

5. **Account tier enforcement patterns** — Common SaaS patterns for:
   - Soft limits (warning) vs hard limits (block action)
   - Grace periods for downgraded users
   - Upgrade flow integration with Stripe/payment

**Recommendation:** Schedule research spikes during implementation phases for items 1-4. Item 5 can defer to v1.2+ (account tiers are scaffolding only for v1.1).

## Sources

**Unable to access WebSearch for 2026 verification.** Findings based on:

**Plugin Architecture Patterns:**
- WordPress plugin ecosystem (training data)
- Zapier integration model (training data)
- n8n self-hosted workflows (training data)
- Grafana plugin architecture (training data)
- VS Code extension model (training data)

**Tile Dashboard Patterns:**
- Windows Metro/Live Tiles UX patterns (training data)
- Grafana dashboard panels (training data)
- Notion database views (training data)
- Trello board layouts (training data)

**Scheduling Systems:**
- Kubernetes CronJob patterns (training data)
- Asynq documentation (verified in STACK.md research)
- APScheduler (Python) patterns (training data)
- Quartz scheduler (Java) patterns (training data)

**Dynamic Settings UI:**
- JSON Schema Form libraries (React JSON Schema Form, training data)
- Backstage plugin configuration (training data)
- Home Assistant integration config (training data)

**Account Tier Systems:**
- Stripe subscription models (training data)
- GitHub Free/Pro/Enterprise tiers (training data)
- Notion Free/Plus/Business tiers (training data)

**Confidence level: MEDIUM**
- High confidence on plugin architecture patterns (stable, well-documented domain)
- Medium confidence on CrewAI integration (newer library, evolving API)
- High confidence on tile dashboards (CSS Grid is mature)
- Medium confidence on dynamic settings (custom implementation, not library-based)
- Medium confidence on account tiers (standard SaaS patterns, but not validated for 2026)

**Recommendation:** Given WebSearch unavailability, validate CrewAI patterns with official docs during implementation. Other patterns are stable and reliable from training data.

---
*Feature research for: Plugin-Based Briefing Architecture*
*Researched: 2026-02-13*
*Confidence: MEDIUM (training data through January 2025, WebSearch unavailable)*
