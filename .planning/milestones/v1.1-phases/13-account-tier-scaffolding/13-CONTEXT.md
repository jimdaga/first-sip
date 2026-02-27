# Phase 13: Account Tier Scaffolding - Context

**Gathered:** 2026-02-23
**Status:** Ready for planning

<domain>
## Phase Boundary

Tier-based constraint enforcement for plugin count and frequency limits. Scaffolding only — no payment integration, no billing, no Stripe. Creates the AccountTier model, associates users with tiers, and enforces limits server-side and in the UI.

</domain>

<decisions>
## Implementation Decisions

### Tier Definitions
- Two tiers: **free** and **pro**, seeded in the database
- Free tier: max **3 enabled plugins**, fastest frequency **once daily**
- Pro tier: **unlimited plugins**, fastest frequency **every 2 hours**
- All new users default to free tier on registration
- Downgrade logic (pro → free) is NOT handled in this phase — scaffolding only

### Enforcement Behavior
- **Hard block** when free user tries to enable a 4th plugin — must disable another or upgrade
- Enforcement happens **both UI + server**: UI prevents the action, server validates as backup
- Frequency options faster than once daily are **visible but locked with a "Pro" badge** in the cron picker — drives discovery of pro tier
- No grace periods, no soft warnings, no override mechanisms

### Upgrade Messaging
- **Tone:** Value-focused — "You're using 3/3 plugins. Pro users get unlimited" — show what they're missing
- **Upgrade CTA** links to a "Pro is coming soon" page with email capture (no payment system yet)
- **Placement:** Upgrade messaging lives in the **plugin settings section only** — no dashboard-wide banners
- Plugin counter ("2/3 plugins enabled") always visible in plugin settings area

### Limit Feedback UX
- Plugin counter shows at all times (e.g., "2/3 plugins enabled")
- Counter uses **normal color until 3/3**, then shifts to warning/accent color — subtle, not traffic-light
- At limit (3/3): enable buttons on other plugins are **disabled with a hover tooltip** explaining the limit
- Enable button is NOT replaced with upgrade CTA — stays as disabled button with tooltip

### Claude's Discretion
- Exact feedback mechanism when blocked (inline message vs toast vs other) — pick what fits existing UI patterns
- "Coming soon" page design and layout
- Tooltip wording and placement
- Exact color shift at 3/3 within the design system tokens

</decisions>

<specifics>
## Specific Ideas

- Pro badge on locked frequency options should feel like a gentle upsell, not aggressive marketing
- The plugin counter should integrate naturally into the existing plugin settings UI, not feel bolted on
- "Coming soon" page should capture email for notification when pro launches

</specifics>

<deferred>
## Deferred Ideas

- Payment integration / Stripe — separate phase
- Downgrade handling (what happens when pro user reverts to free) — future phase
- Usage analytics / tier metrics dashboard — future phase

</deferred>

---

*Phase: 13-account-tier-scaffolding*
*Context gathered: 2026-02-23*
