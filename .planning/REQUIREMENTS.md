# Requirements: First Sip — Daily Briefing

**Defined:** 2026-02-10
**Core Value:** A user can click "Generate" and receive a multi-source daily briefing without leaving the app — the background processing, source aggregation, and status tracking all happen seamlessly.

## v1 Requirements

Requirements for initial release. Each maps to roadmap phases.

### Authentication

- [ ] **AUTH-01**: User can log in with Google OAuth and be redirected to dashboard
- [ ] **AUTH-02**: User session persists across browser refresh
- [ ] **AUTH-03**: User can log out from any page

### Briefing Generation

- [ ] **BGEN-01**: User can click "Generate Daily Summary" to trigger briefing creation
- [ ] **BGEN-02**: Briefing tracks status through Pending → Completed → Failed states
- [ ] **BGEN-03**: Asynq worker generates briefing using mock n8n data (stub)
- [ ] **BGEN-04**: Briefings auto-generate daily on a configurable schedule via Asynq cron

### Briefing Display

- [ ] **BDISP-01**: Dashboard shows briefing organized in distinct sections (News/Weather/Work)
- [ ] **BDISP-02**: UI is mobile-responsive using DaisyUI components
- [ ] **BDISP-03**: User can see read/unread state on briefings
- [ ] **BDISP-04**: User can browse past briefings (last 30 days)

### Infrastructure

- [ ] **INFR-01**: Asynq worker process runs with Redis for background job processing
- [ ] **INFR-02**: User and Briefing models stored in Postgres via GORM with migrations
- [ ] **INFR-03**: n8n webhook HTTP client sends requests with X-N8N-SECRET header (stub mode)
- [ ] **INFR-04**: Failed tasks retry with configurable policy and dead letter queue

## v2 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Authentication Enhancements

- **AUTH-04**: OAuth tokens encrypted at rest via GORM hooks (BeforeSave/AfterFind)
- **AUTH-05**: Auth middleware protects all dashboard routes, redirects unauthenticated users

### Briefing Enhancements

- **BGEN-05**: Real n8n webhook integration (replace mock stub with live workflows)
- **BGEN-06**: Graceful source failures — partial briefing if one n8n source fails

### Source Management

- **SRC-01**: User can select which sources to include in their briefing
- **SRC-02**: User can preview/test a source workflow before enabling

### Delivery

- **DLVR-01**: User can receive daily briefing via email
- **DLVR-02**: User can configure preference for email vs dashboard delivery

## Out of Scope

| Feature | Reason |
|---------|--------|
| Real-time push updates | Defeats batching purpose; HTMX polling sufficient for status |
| Social sharing | Privacy concern, spam vector, not aligned with personal tool |
| Cross-source synthesis ("3 sources mention X") | High complexity, validate basic summaries first |
| Interactive chat/follow-up on briefing items | Major scope expansion, high LLM costs |
| Voice/TTS briefing | Different UX paradigm, defer to v2+ |
| Team/org dashboards | Multi-tenant complexity, start single-user |
| Push notifications | Defeats batching, becomes noise |
| Gamification | Toxic engagement patterns, briefing should reduce anxiety |
| Full n8n UI exposure | Paradox of choice; start with preset sources |
| iOS mobile app | Long-term goal, web-first |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| AUTH-01 | Phase 1 | Pending |
| AUTH-02 | Phase 1 | Pending |
| AUTH-03 | Phase 1 | Pending |
| BGEN-01 | Phase 4 | Pending |
| BGEN-02 | Phase 4 | Pending |
| BGEN-03 | Phase 4 | Pending |
| BGEN-04 | Phase 6 | Pending |
| BDISP-01 | Phase 5 | Pending |
| BDISP-02 | Phase 5 | Pending |
| BDISP-03 | Phase 5 | Pending |
| BDISP-04 | Phase 7 | Pending |
| INFR-01 | Phase 3 | Pending |
| INFR-02 | Phase 2 | Pending |
| INFR-03 | Phase 4 | Pending |
| INFR-04 | Phase 3 | Pending |

**Coverage:**
- v1 requirements: 15 total
- Mapped to phases: 15
- Unmapped: 0 ✓

---
*Requirements defined: 2026-02-10*
*Last updated: 2026-02-10 after roadmap creation*
