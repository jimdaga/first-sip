# Feature Research

**Domain:** Daily Briefing / Personal Dashboard / AI Summary Apps
**Researched:** 2026-02-10
**Confidence:** MEDIUM (based on training data through Jan 2025, unable to verify with WebSearch)

## Feature Landscape

### Table Stakes (Users Expect These)

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Single sign-on (OAuth) | Users won't create another password | LOW | Google/Microsoft most common. Already planned. |
| Personalization/source selection | "My briefing" not "a briefing" | MEDIUM | Which news sources, which data feeds. Core value prop. |
| Daily scheduled generation | Set-and-forget automation | MEDIUM | Cron/background job. n8n workflows already planned. |
| Mobile-responsive UI | 60%+ consume on mobile | LOW | DaisyUI handles this. Templ templates must be responsive. |
| Read/unread tracking | Don't show me old stuff | LOW | Session state or DB flag per briefing |
| Delivery mechanism | Must get TO user | MEDIUM | Email most common, but web dashboard acceptable for MVP |
| Fast load time | Briefings compete with attention | MEDIUM | HTMX polling already planned. Pre-generate, don't compute on-demand. |
| Clear section organization | Weather/news/calendar distinct | LOW | Template structure. Helps scanning. |
| Graceful source failures | One API down ≠ no briefing | MEDIUM | n8n workflow error handling critical |
| Time-to-briefing indicator | "Generating..." vs "Ready" | LOW | Status polling already planned with HTMX |

### Differentiators (Competitive Advantage)

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| n8n workflow customization | Power users build their own sources | HIGH | Expose n8n interface? Or pre-built templates? Huge differentiator if done well. |
| AI summarization quality | Better summaries = more value | HIGH | Model choice, prompt engineering, context preservation. Not all summaries equal. |
| Cross-source synthesis | "3 sources mention Ukraine" | HIGH | LLM can detect themes across sources. Very differentiating. |
| Briefing versioning/history | "What did I see last Tuesday?" | MEDIUM | Archive briefings, search history. Competes with email delivery. |
| Custom source integration | "Add my GitHub notifications" | HIGH | n8n makes this possible. API flexibility is differentiator. |
| Contextual follow-up | "Tell me more about X" | HIGH | Interactive AI chat on briefing items. Moves beyond static summary. |
| Smart scheduling | "Brief me when I'm ready" | HIGH | Learn user patterns, don't spam. Requires ML/behavior tracking. |
| Collaborative briefings | Team dashboards, shared sources | MEDIUM | Multi-tenant, permissions. Not MVP but valuable for orgs. |
| Offline support / PWA | Read on subway | MEDIUM | Service worker, cache briefings. Nice-to-have. |
| Voice briefing | Listen while getting ready | HIGH | TTS integration. Audio format different from text. |

### Anti-Features (Commonly Requested, Often Problematic)

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| Real-time updates | "I want live news!" | Defeats purpose of briefing (curated moment). Infinite scroll hell. Notification spam. | Scheduled refresh intervals (6hr, 12hr). Clear "Last updated" timestamp. |
| Social features | "Share my briefing!" | Privacy nightmare. What's in briefing is personal. Spam vector. | Share individual items, not whole briefing. Export to PDF/email. |
| Infinite customization | "Let me tweak everything!" | Paradox of choice. Maintenance burden. 80% never customize. | Curated presets with simple toggles. "News-focused" vs "Productivity-focused" templates. |
| In-app commenting | "Let me annotate items" | Scope creep. Use note-taking apps. Sync complexity. | Export to Notion/Obsidian. External integrations. |
| Push notifications | "Alert me!" | Defeats batching purpose. Becomes noise. Engagement manipulation. | Email digest. Opt-in for critical alerts only (not per-item). |
| Gamification | "Streaks! Points!" | Toxic engagement pattern. Briefing should reduce anxiety, not create it. | Simple "days using" counter. No pressure. |
| Multi-device sync (complex) | "Same state everywhere" | Overkill for stateless briefings. What's "state"? Read/unread? | Briefings are ephemeral. New one daily. Don't sync, regenerate. |
| Ads in briefing | "Monetization!" | Ruins trust. User came to REDUCE noise. | Premium tier, self-hosted, or sponsor model (transparent). |

## Feature Dependencies

```
Google OAuth → User identity → Personalized sources
User identity → Briefing generation → Read/unread tracking
Scheduled generation → Briefing storage → History/versioning
Source integration → n8n workflow → Failure handling
Mobile UI → Responsive templates → Touch-friendly controls
AI summarization → API costs → Usage limits/quotas
```

## MVP Definition

### Launch With (v1 - Bootstrap)
- [x] Google OAuth login (planned)
- [x] Mock briefing generation (planned)
- [x] Status polling UI (planned)
- [ ] **Basic source selection** — Let user pick 2-3 pre-configured sources (news feed, weather, placeholder for "work"). Don't build n8n UI yet, hardcode initial workflows.
- [ ] **Section-based layout** — News section, weather section, etc. Clear visual hierarchy.
- [ ] **Read/unread state** — Simple flag: "Mark as read" button or auto-mark on view.
- [ ] **Mobile-responsive** — DaisyUI default should handle this, verify on phone.
- [ ] **Graceful degradation** — If one source fails, show others + error message.
- [ ] **Logout** — Don't trap users.

### Add After Validation (v1.x)
- [ ] **Email delivery** — Once dashboard works, add email option (trigger: 10+ active users)
- [ ] **Briefing history** — Archive last 30 days (trigger: users ask "where's yesterday?")
- [ ] **Custom source wizard** — Guided n8n workflow builder (trigger: users want GitHub/Slack/custom)
- [ ] **AI quality improvements** — Better prompts, summarization tuning (trigger: feedback on summaries)
- [ ] **Multiple briefing times** — Morning + evening (trigger: user requests)
- [ ] **Export to PDF** — Generate downloadable briefing (trigger: users screenshot it)

### Future Consideration (v2+)
- [ ] **Cross-source synthesis** — AI detects themes across sources (why defer: complex, MVP validates if summaries matter)
- [ ] **Contextual follow-up** — Chat with briefing items (why defer: major scope, LLM costs)
- [ ] **Team dashboards** — Shared organizational briefings (why defer: multi-tenant complexity)
- [ ] **Voice briefing** — TTS audio version (why defer: different UX paradigm)
- [ ] **Smart scheduling** — ML-based timing (why defer: need usage data first)
- [ ] **PWA/offline** — Progressive web app (why defer: web-first is fine for MVP)

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| Google OAuth | HIGH | LOW (exists) | P0 (done) |
| Briefing generation | HIGH | MEDIUM (exists) | P0 (done) |
| Status polling UI | HIGH | LOW (exists) | P0 (done) |
| Source selection | HIGH | MEDIUM | P1 (MVP) |
| Section layout | HIGH | LOW | P1 (MVP) |
| Read/unread | MEDIUM | LOW | P1 (MVP) |
| Mobile responsive | HIGH | LOW | P1 (MVP) |
| Error handling | HIGH | MEDIUM | P1 (MVP) |
| Email delivery | MEDIUM | MEDIUM | P2 (post-launch) |
| Briefing history | MEDIUM | MEDIUM | P2 (post-launch) |
| Custom sources | HIGH | HIGH | P2 (post-launch) |
| AI improvements | MEDIUM | MEDIUM | P2 (iterative) |
| Cross-source synthesis | MEDIUM | HIGH | P3 (future) |
| Follow-up chat | MEDIUM | HIGH | P3 (future) |
| Team dashboards | LOW | HIGH | P3 (future) |
| Voice briefing | LOW | HIGH | P3 (future) |

## Competitor Feature Analysis

| Feature | Apple News+ / Google News | Artifact (RIP) | Feedbin / RSS readers | Our Approach |
|---------|---------------------------|----------------|----------------------|--------------|
| Personalization | Algorithmic, opaque | ML-based, adaptive | Manual source selection | **n8n workflows** — transparent, user-controlled |
| AI Summaries | Limited, article-level | Strong, cross-article | None (raw feeds) | **Full briefing summarization** via LLM |
| Custom sources | Curated publishers only | Web + social | Any RSS/JSON | **Any API via n8n** — most flexible |
| Delivery | App-only | App-only | Email/web/app | **Web-first, email later** |
| Multi-source | Siloed by publisher | Unified feed | Reader manages | **Unified briefing** — one page |
| Scheduling | Push-based | Algorithmic | User-controlled | **User-scheduled** — cron-based |
| Cost | $9.99/mo subscription | Free (ad-supported) | $5/mo or free | **Free for personal, premium for advanced** |
| Technical flexibility | None | None | Import/export OPML | **Full n8n access** — ultimate flexibility |

## Domain Patterns Observed

### Successful Patterns
- **Batching over real-time** — Daily/scheduled beats constant stream
- **Curation over firehose** — 10 quality items > 100 mediocre
- **Summaries over links** — Users want digest, not homework
- **Scheduled delivery** — Morning routine integration (7am-9am peak)
- **Escape hatch** — Always provide link to full source
- **Failure transparency** — "Weather unavailable" better than silent omission
- **Speed** — Sub-2-second load for cached briefings

### Failed Patterns
- **Over-personalization** — Filter bubbles, echo chambers
- **Social engagement** — Likes/shares distract from information consumption
- **Attention maximization** — Infinite scroll, autoplay, notifications
- **Complex configuration** — 20+ settings = abandoned
- **Platform lock-in** — Users distrust apps that trap data

## Bootstrap Phase Feature Scope

For the clickable demo with mock data, include:

1. **Login flow** — Google OAuth (real)
2. **Generate button** — Trigger briefing (mock workflow)
3. **Status polling** — "Generating..." → "Ready" (HTMX)
4. **Mock briefing display** — 3 sections:
   - **News** (3-4 summarized articles)
   - **Weather** (current + forecast)
   - **Placeholder** ("Your work updates will appear here")
5. **Visual hierarchy** — Clear sections, scannable
6. **Mobile layout** — Verify on phone
7. **Logout** — Back to login

**Explicitly skip for Bootstrap:**
- Real n8n workflows (use mock data)
- Source selection UI (hardcoded)
- Briefing persistence (show only latest)
- History/archive
- Email delivery
- Read/unread tracking (not needed for demo)

## Sources

**Note:** Unable to access WebSearch for 2026 verification. Findings based on:
- Training data through January 2025 covering:
  - News aggregator apps (Apple News, Google News, Flipboard)
  - AI summary tools (Artifact, ChatGPT summaries, Perplexity)
  - RSS readers (Feedbin, Feedly, Inoreader)
  - Dashboard tools (Notion, Obsidian dashboards)
  - Morning briefing services (theSkimm, Morning Brew)

**Confidence level: MEDIUM**
- High confidence on table stakes (stable domain patterns)
- Medium confidence on differentiators (AI landscape evolving)
- Low confidence on 2026-specific trends (unable to verify current state)

**Recommendation:** Validate anti-features list with 2-3 potential users before finalizing. The n8n workflow differentiator is unique but unproven — consider whether exposing full n8n complexity helps or hurts UX.

---
*Feature research for: Daily Briefing / Personal Dashboard / AI Summary Apps*
*Researched: 2026-02-10*
*Confidence: MEDIUM (training data, WebSearch unavailable)*
