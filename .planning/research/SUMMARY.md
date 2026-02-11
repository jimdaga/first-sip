# Project Research Summary

**Project:** First Sip - Daily Briefing Application
**Domain:** Go Web Application with Server-Side Rendering + Background Jobs (Daily Briefing/Personal Dashboard)
**Researched:** 2026-02-10
**Confidence:** MEDIUM-HIGH

## Executive Summary

First Sip is a daily briefing application that aggregates personalized content (news, weather, work updates) using n8n workflows and presents them through a server-rendered web interface. Based on research across modern Go web patterns, successful briefing applications, and production pitfalls, the recommended approach is a Gin + Templ + HTMX frontend with Asynq background workers orchestrating n8n workflows, backed by PostgreSQL for persistence and Redis for session management and job queuing.

The core value proposition centers on n8n workflow customization as a differentiator—allowing power users to build custom data sources—while maintaining table stakes features like OAuth authentication, scheduled generation, and mobile-responsive UI. The architecture follows standard Go web patterns with clear handler-service-repository boundaries, Templ component composition for maintainable templates, and HTMX for interactive polling without complex JavaScript. Critical to success is establishing proper transaction boundaries early, implementing HTMX polling with backoff to avoid thundering herd problems, and never passing sensitive data through Asynq task payloads.

Key risks include GORM N+1 query explosions (mitigate with eager loading and query monitoring), OAuth token refresh race conditions (mitigate with distributed locks), and HTMX polling storms under concurrent load (mitigate with exponential backoff and Redis caching). The recommended phase structure prioritizes core authentication and briefing display first, followed by real n8n integration and source customization, with advanced features like email delivery and cross-source synthesis deferred to later phases after validating core value proposition.

## Key Findings

### Recommended Stack

The recommended stack leverages Go 1.23+ with Gin for HTTP routing, Templ for type-safe HTML templating, and HTMX for client-side interactivity. GORM provides ORM capabilities with PostgreSQL, while Asynq handles background job processing via Redis. Authentication uses Goth for multi-provider OAuth with gorilla/sessions for session management. The frontend uses Tailwind CSS + DaisyUI for styling.

**Core technologies:**
- **Gin v1.10.0+**: HTTP router and middleware — battle-tested, excellent middleware ecosystem, 15x faster routing than stdlib
- **Templ v0.2.747+**: Type-safe HTML templating — compile-time type safety, generates Go code, better DX than html/template
- **HTMX v2.0.0+**: Client-side interactivity — hypermedia-driven, minimal JS, perfect for server-rendered apps
- **GORM v1.25.11+**: ORM for Postgres — most popular Go ORM, excellent Postgres support, hooks system, migration support
- **Asynq v0.24.1+**: Background job queue — Redis-backed, built-in retries, scheduled tasks, monitoring UI
- **Goth v1.80.0+**: Multi-provider OAuth — supports 40+ providers including Google, simple API
- **PostgreSQL 16+**: Primary database — rock-solid ACID compliance, excellent GORM support, JSON types
- **Redis 7+**: Job queue and cache — required for Asynq, can also cache session data and status polling results
- **Tailwind CSS v3.4+ + DaisyUI v4.12+**: Utility-first CSS with component library — industry standard, pre-built components

**Critical gotchas:**
- Templ requires build step (`templ generate`) before `go build`
- HTMX v2 has breaking changes from v1 (hx-boost behavior, WebSocket syntax)
- GORM AutoMigrate dangerous in production—use explicit migrations
- Never use pgx driver with GORM (compatibility issues)—use lib/pq
- Asynq task payloads must be JSON-serializable

### Expected Features

Daily briefing applications have clear table stakes features that users expect, several competitive differentiators, and anti-features that appear valuable but create problems. The Bootstrap phase should focus on proving core value with mock data before investing in complex customization features.

**Must have (table stakes):**
- **Google OAuth login** — users won't create another password (already in codebase)
- **Scheduled daily generation** — set-and-forget automation is core value
- **Mobile-responsive UI** — 60%+ of users consume on mobile
- **Section-based organization** — weather/news/calendar distinct for scanning
- **Graceful source failures** — one API down should not break entire briefing
- **Read/unread tracking** — don't show stale content

**Should have (competitive advantage):**
- **n8n workflow customization** — power users build their own sources (huge differentiator if done well)
- **AI summarization quality** — better summaries = more value than competitors
- **Briefing history/versioning** — "What did I see last Tuesday?" competes with email delivery
- **Custom source integration** — "Add my GitHub notifications" via n8n

**Defer to v2+ (avoid scope creep):**
- **Cross-source synthesis** — "3 sources mention Ukraine" (complex, validate if summaries matter first)
- **Contextual follow-up chat** — interactive AI on briefing items (major scope, high LLM costs)
- **Team dashboards** — shared organizational briefings (multi-tenant complexity)
- **Voice briefing** — TTS audio version (different UX paradigm)

**Anti-features to avoid:**
- **Real-time updates** — defeats batching purpose, creates notification spam
- **Social sharing** — privacy nightmare, spam vector
- **Infinite customization** — paradox of choice, 80% never customize
- **Push notifications** — defeats batching, becomes noise
- **Gamification** — toxic engagement patterns, briefing should reduce anxiety

### Architecture Approach

The architecture separates concerns into distinct processes: Gin HTTP server (stateless, horizontally scalable) and Asynq workers (CPU-bound, vertically scalable). HTMX handles client-side polling and DOM updates without complex JavaScript. Templ components compose hierarchically (layout + page + partials) with conditional rendering for HTMX requests versus full page loads. Background jobs communicate with external n8n workflows via HTTP webhooks, aggregate responses, and store results in PostgreSQL. Redis serves dual purpose as session store and job queue.

**Major components:**
1. **Gin HTTP Server** — routes requests, middleware pipeline (logging, recovery, CORS, sessions, auth), renders Templ components, enqueues Asynq tasks
2. **Asynq Worker Process** — executes background jobs (briefing generation, email sending, cleanup), calls n8n webhooks, stores results in Postgres
3. **Templ Templates** — type-safe HTML generation with component composition, compile-time checked
4. **GORM Models** — database ORM shared by server and worker processes
5. **Goth Auth Service** — OAuth flow management with Google, session storage in Redis
6. **n8n External Service** — workflow orchestration engine, called via HTTP webhooks from workers

**Critical patterns:**
- **Handler → Service → Repository** — separate HTTP layer, business logic, data access for testability
- **Templ component composition** — nested layout(header, content, footer) with partial rendering for HTMX
- **HTMX partial rendering** — check HX-Request header, return fragments vs full pages
- **Asynq task serialization** — JSON payloads, only pass IDs (never sensitive data)
- **Transaction boundaries** — render to buffer first, commit only after successful template render

### Critical Pitfalls

Research identified 10 critical pitfalls specific to this stack combination. The top 5 that must be addressed in Bootstrap phase:

1. **GORM Hooks Causing N+1 Queries** — encryption/decryption hooks in AfterFind trigger per-record, multiplying database queries exponentially. Avoid by keeping hooks to pure computation only, never query database inside hooks, monitor query count with middleware that alerts if >5 queries per request.

2. **HTMX Polling Without Backoff Creates Thundering Herd** — `hx-trigger="every 2s"` with 100 concurrent users = 50 requests/second. Avoid by returning HX-Trigger headers to stop polling on completion, implement exponential backoff (1s, 2s, 5s, 10s), add Redis cache layer for status endpoints.

3. **Asynq Task Serialization Exposes Sensitive Data** — task payloads stored as JSON in Redis, passing OAuth tokens directly exposes them. Avoid by only passing user IDs in payloads, fetch sensitive data inside task handlers, enable Redis encryption at rest, set task retention policies.

4. **Templ Component Boundaries Cause Layout Duplication** — HTMX needs fragments but developers duplicate layout code per endpoint. Avoid by establishing layout pattern early (layout + content), detect HX-Request header to return fragments vs full pages, create RenderWithLayout helper.

5. **OAuth Token Refresh Race Conditions** — multiple tabs trigger simultaneous refresh, provider invalidates token after first request. Avoid by using Redis distributed lock for refresh operations, implement single-flight pattern with golang.org/x/sync/singleflight, add jitter to expiry checks.

## Implications for Roadmap

Based on research, suggested phase structure emphasizes validating core value proposition before investing in complex customization features. Phase ordering follows dependency chains identified in architecture research and avoids pitfalls flagged in security/performance analysis.

### Phase 1: Authentication & User Management
**Rationale:** Foundation for all personalized features. OAuth integration is table stakes and gates access to rest of application. Must establish security patterns early (session management, CSRF protection, token storage).

**Delivers:**
- Google OAuth login/logout flow
- Session management with Redis backend
- User model and database schema
- Auth middleware for protected routes

**Addresses:**
- OAuth authentication (table stakes from FEATURES.md)
- User identity for personalization

**Avoids:**
- OAuth token refresh race conditions (Critical Pitfall #5) — implement distributed locking from start
- Storing tokens insecurely (Security Mistakes) — establish encrypted token pattern

**Research flags:** Standard OAuth patterns, unlikely to need phase-specific research

### Phase 2: Core Briefing Display (Mock Data)
**Rationale:** Prove UI/UX value before investing in backend complexity. Establish Templ component patterns and HTMX interaction patterns with mock data. Validates mobile-responsive design and section organization.

**Delivers:**
- Briefing display page with section layout (news, weather, placeholder)
- Templ component library (layout, header, footer, briefing card)
- HTMX status polling mechanism
- Mock briefing generation (returns hardcoded data)

**Addresses:**
- Mobile-responsive UI (table stakes)
- Section-based organization (table stakes)
- Fast load time (table stakes)
- Status polling UI (already planned)

**Avoids:**
- Templ layout duplication (Critical Pitfall #4) — establish component composition pattern early
- HTMX polling without backoff (Critical Pitfall #2) — implement exponential backoff from start
- HTMX OOB swaps accumulation (Pitfall #8) — use outerHTML for replacements

**Research flags:** Standard patterns, unlikely to need research. Templ + HTMX integration patterns are well-documented.

### Phase 3: Background Job Infrastructure
**Rationale:** Required before real briefing generation. Establishes Asynq patterns, task serialization, error handling, and monitoring. Must get this right before connecting to external services.

**Delivers:**
- Asynq worker process setup
- Task handler registration and serialization patterns
- Retry configuration and dead letter queue
- Basic job status tracking in Postgres
- Asynqmon monitoring UI (optional)

**Addresses:**
- Scheduled daily generation (table stakes) — cron-based task enqueuing
- Graceful failure handling (table stakes) — retry policies

**Avoids:**
- Asynq sensitive data in payloads (Critical Pitfall #3) — establish ID-only payload pattern
- Worker panics losing tasks (Pitfall #7) — configure generous retry with error handler
- GORM connection leaks in workers (Performance Trap) — proper connection pooling

**Research flags:** Standard async job patterns, unlikely to need research. Asynq documentation is comprehensive.

### Phase 4: n8n Workflow Integration
**Rationale:** Connect to real data sources. This is first integration with external system, needs careful error handling. Start with simple webhook calls before complex orchestration.

**Delivers:**
- n8n HTTP client library
- Basic workflow webhook calls (news, weather)
- Response parsing and validation
- Error handling for failed webhooks
- Briefing storage in Postgres

**Addresses:**
- Real briefing generation (replaces mock data)
- Multiple data sources (table stakes)
- Graceful source failures (table stakes) — partial briefing if one source fails

**Avoids:**
- Long-running HTTP requests in handlers (Anti-Pattern #4) — all n8n calls via Asynq
- Missing transaction boundaries (Pitfall #6) — render to buffer before commit

**Research flags:** Likely needs phase-specific research on n8n webhook API patterns, authentication, and error responses. Low confidence on n8n integration patterns from initial research.

### Phase 5: Source Selection & Customization
**Rationale:** Now that infrastructure works, add user control over sources. Start simple (checkbox selection) before exposing full n8n workflow builder.

**Delivers:**
- Source selection UI (choose from pre-configured workflows)
- User preferences storage (which sources enabled)
- Workflow parameter passing (user-specific context)
- Preview/test workflow button

**Addresses:**
- Personalization/source selection (table stakes)
- Custom source integration (differentiator) — start with presets, expose n8n later

**Avoids:**
- Infinite customization anti-feature — curated presets first
- Over-personalization failure pattern — provide sane defaults

**Research flags:** Standard preferences management, unlikely to need research unless exposing n8n UI directly.

### Phase 6: History & Read Tracking
**Rationale:** Once daily generation works, users want to review past briefings and track what they've read. Relatively independent feature, can be added without changing core flow.

**Delivers:**
- Briefing archive (last 30 days)
- Read/unread state tracking
- History page with date navigation
- "Mark all as read" functionality

**Addresses:**
- Read/unread tracking (table stakes)
- Briefing history/versioning (differentiator)

**Avoids:**
- GORM N+1 queries (Critical Pitfall #1) — use Preload for relations
- GORM Preload + Select incompatibility (Pitfall #9) — include foreign keys

**Research flags:** Standard CRUD patterns, unlikely to need research.

### Phase 7: Email Delivery (Optional)
**Rationale:** Defer until validated that dashboard approach works. Email is alternative delivery mechanism, not core to initial value prop. Add only if users request it.

**Delivers:**
- Email template rendering (reuse Templ components)
- SMTP integration
- User preference for email vs dashboard
- Daily email task in Asynq scheduler

**Addresses:**
- Delivery mechanism (table stakes) — dashboard already sufficient, email optional

**Avoids:**
- Adding complexity before validation

**Research flags:** Standard email patterns with Go, unlikely to need research.

### Phase Ordering Rationale

- **Authentication first** because all other features require user identity and session management. OAuth patterns must be correct before building on top.
- **Mock data display second** to validate UI/UX and establish frontend patterns (Templ/HTMX) without backend complexity. Proves value proposition early.
- **Background jobs third** because real briefing generation requires async processing. Must establish job patterns before external integrations.
- **n8n integration fourth** after infrastructure proven stable. External service integration is highest risk, save for when foundation is solid.
- **Source customization fifth** after proving basic generation works. Customization is differentiator but not required for initial validation.
- **History/email last** as enhancements after core loop proven. Nice-to-have features that don't block basic functionality.

This ordering follows architecture dependencies (auth → display → jobs → external integrations → enhancements) while frontloading critical pitfall mitigation (OAuth security, HTMX polling, Asynq patterns). Phases are sized to deliver value incrementally while avoiding scope creep from anti-features.

### Research Flags

**Phases likely needing deeper research during planning:**
- **Phase 4 (n8n Integration):** External system with low confidence from initial research. Needs investigation of n8n webhook API patterns, authentication mechanisms, error response formats, timeout handling, and rate limiting.
- **Phase 5 (Source Customization):** Depends on whether exposing n8n UI directly or building custom wizard. If exposing n8n, needs UX research on complexity vs power user value trade-off.

**Phases with standard patterns (skip research-phase):**
- **Phase 1 (Authentication):** Well-documented OAuth patterns, Goth library examples abundant
- **Phase 2 (Display):** Standard Go web patterns, Templ/HTMX documented adequately
- **Phase 3 (Background Jobs):** Asynq documentation comprehensive, standard queue patterns
- **Phase 6 (History):** Standard CRUD with GORM, no novel patterns
- **Phase 7 (Email):** Standard Go email libraries, Templ templates reusable

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | MEDIUM | High confidence on Go stdlib, Gin, GORM, HTMX. Medium-low on Templ (fast-moving), Asynq (version staleness). Critical to verify current versions before starting. |
| Features | MEDIUM | High confidence on table stakes (stable domain patterns). Medium on differentiators (n8n customization unproven). Low confidence on 2026-specific trends (WebSearch unavailable). |
| Architecture | MEDIUM-HIGH | High confidence on Go web patterns, GORM, Gin middleware. Medium on Templ + HTMX integration (newer stack combination). Low on n8n integration (external system). |
| Pitfalls | MEDIUM-HIGH | High confidence on GORM, Asynq, OAuth patterns (well-documented). Medium on HTMX-specific behaviors. Medium-low on Templ pitfalls (newer tool, less production data). |

**Overall confidence:** MEDIUM-HIGH

Confident in core Go web patterns, GORM/Postgres, background job architectures, and OAuth security—these are mature, well-documented areas. Less confident in Templ + HTMX integration patterns (newer combination) and n8n webhook integration (external system, sparse documentation in training data). Version numbers are likely dated (training data January 2025, now February 2026)—must verify current releases before implementation.

### Gaps to Address

- **Templ version and API stability:** Fast-moving project, version 0.2.747+ may have changed significantly. Verify current version, check for breaking changes, review official examples for latest patterns. Handle during Phase 2 planning.

- **HTMX v2 adoption and ecosystem:** v2 released mid-2024 with breaking changes. Verify community adoption, check for regression issues, confirm DaisyUI compatibility. Handle during Phase 2 planning.

- **n8n webhook API patterns:** Low confidence on authentication, error handling, timeout behavior, rate limits. Needs dedicated research during Phase 4 planning—use `/gsd:research-phase` to investigate n8n webhook integration patterns, authentication options, and error response formats.

- **n8n workflow customization UX:** Exposing full n8n interface vs building guided wizard is unresolved. Validate with 2-3 potential users during Phase 5 planning to understand power user vs simplicity trade-off.

- **AI summarization model choice:** No research conducted on LLM selection, prompt engineering, context preservation, or cost optimization. Defer to implementation phase, start with simple OpenAI API calls, iterate based on quality feedback.

- **Performance numbers validation:** Database query limits, concurrent user thresholds, polling frequency recommendations are estimates. Establish performance baselines during Phase 2-3 with load testing. Adjust thresholds based on actual measurements.

- **Security compliance requirements:** No research on GDPR, data retention policies, user data export requirements. Clarify legal requirements before Phase 1, especially for OAuth token storage and briefing content retention.

## Sources

### Primary (HIGH confidence)
- Go documentation and stdlib patterns (training data through January 2025)
- Gin framework documentation v1.10.0 patterns
- GORM documentation on hooks, transactions, preloading
- HTMX documentation on polling, OOB swaps, headers
- Asynq documentation on task serialization, retries, dead queues
- OAuth 2.0 security best practices and token management patterns
- PostgreSQL documentation on connection pooling and GORM integration

### Secondary (MEDIUM confidence)
- Templ documentation and component patterns (v0.2.747+ era)
- DaisyUI + Tailwind CSS integration patterns
- Background job queue patterns in Go ecosystems
- Server-side rendering with HTMX architectural patterns
- Daily briefing application feature analysis (Apple News, Artifact, Feedbin comparisons)

### Tertiary (LOW confidence, needs validation)
- n8n webhook API patterns and integration approaches
- Templ + HTMX production deployment experiences
- Current version numbers for all dependencies (January 2025 cutoff)
- 2026-specific trends in daily briefing applications
- Cross-source AI synthesis implementation approaches

**Note:** WebSearch and WebFetch were unavailable during research. All findings based on training data with January 2025 cutoff. Strongly recommend verifying current package versions, checking for breaking changes, and consulting official documentation before implementation starts.

---
*Research completed: 2026-02-10*
*Ready for roadmap: yes*
