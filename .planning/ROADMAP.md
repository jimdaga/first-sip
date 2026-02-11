# Roadmap: First Sip — Daily Briefing

## Overview

This roadmap transforms the minimal Go service into a production-ready daily briefing application through seven phases. Starting with Google OAuth authentication, we establish database models and background job infrastructure, then deliver core briefing generation with mock data to prove the flow works. After validating the UI with real briefings, we add scheduled automation and history browsing to complete the v1 experience.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [ ] **Phase 1: Authentication** - Google OAuth login with session persistence
- [ ] **Phase 2: Database & Models** - Postgres setup with GORM migrations
- [ ] **Phase 3: Background Job Infrastructure** - Asynq worker with Redis queue
- [ ] **Phase 4: Briefing Generation (Mock)** - Core generation flow with stub data
- [ ] **Phase 5: Briefing Display** - Dashboard UI with section layout
- [ ] **Phase 6: Scheduled Generation** - Daily auto-generation via cron
- [ ] **Phase 7: Briefing History** - Browse past briefings with read tracking

## Phase Details

### Phase 1: Authentication
**Goal**: Users can securely access their accounts
**Depends on**: Nothing (first phase)
**Requirements**: AUTH-01, AUTH-02, AUTH-03
**Success Criteria** (what must be TRUE):
  1. User can click "Login with Google" and complete OAuth flow
  2. User session persists across browser refresh and restarts
  3. User can click "Logout" from dashboard and be returned to login page
  4. Protected routes redirect unauthenticated users to login
**Plans**: TBD

Plans:
- TBD (created during /gsd:plan-phase 1)

### Phase 2: Database & Models
**Goal**: Application has persistent storage for users and briefings
**Depends on**: Phase 1
**Requirements**: INFR-02
**Success Criteria** (what must be TRUE):
  1. User records persist in Postgres with encrypted OAuth tokens
  2. Briefing records can be created, read, and updated via GORM
  3. Database migrations run automatically on application start
  4. Local development works with SQLite, production uses Postgres
**Plans**: TBD

Plans:
- TBD (created during /gsd:plan-phase 2)

### Phase 3: Background Job Infrastructure
**Goal**: Application can process long-running tasks asynchronously
**Depends on**: Phase 2
**Requirements**: INFR-01, INFR-04
**Success Criteria** (what must be TRUE):
  1. Asynq worker process starts and connects to Redis
  2. Tasks enqueued from HTTP handlers execute in worker process
  3. Failed tasks retry automatically with exponential backoff
  4. Tasks that exceed retry limit move to dead letter queue
  5. Worker can be monitored via Asynqmon UI (optional)
**Plans**: TBD

Plans:
- TBD (created during /gsd:plan-phase 3)

### Phase 4: Briefing Generation (Mock)
**Goal**: Users can trigger briefing generation and see results
**Depends on**: Phase 3
**Requirements**: BGEN-01, BGEN-02, BGEN-03, INFR-03
**Success Criteria** (what must be TRUE):
  1. User clicks "Generate Daily Summary" button on dashboard
  2. Briefing status shows "Pending" immediately after click
  3. Status updates to "Completed" automatically after worker finishes
  4. Generated briefing displays mock content (news, weather, work sections)
  5. n8n webhook client sends requests with X-N8N-SECRET header (stub mode)
  6. Failed generation shows "Failed" status with error message
**Plans**: TBD

Plans:
- TBD (created during /gsd:plan-phase 4)

### Phase 5: Briefing Display
**Goal**: Dashboard presents briefings in mobile-friendly, organized layout
**Depends on**: Phase 4
**Requirements**: BDISP-01, BDISP-02, BDISP-03
**Success Criteria** (what must be TRUE):
  1. Briefing sections (News/Weather/Work) display in distinct visual cards
  2. Dashboard layout is responsive and usable on mobile screens
  3. DaisyUI components style all UI elements consistently
  4. Read/unread indicator shows on each briefing
  5. User can mark briefing as read by clicking it
**Plans**: TBD

Plans:
- TBD (created during /gsd:plan-phase 5)

### Phase 6: Scheduled Generation
**Goal**: Briefings generate automatically on daily schedule
**Depends on**: Phase 5
**Requirements**: BGEN-04
**Success Criteria** (what must be TRUE):
  1. Asynq cron task triggers daily at configured time (default 6 AM)
  2. User wakes up to new briefing without manual generation
  3. Schedule is configurable via environment variable
  4. Scheduled generation follows same flow as manual generation
**Plans**: TBD

Plans:
- TBD (created during /gsd:plan-phase 6)

### Phase 7: Briefing History
**Goal**: Users can browse and review past briefings
**Depends on**: Phase 6
**Requirements**: BDISP-04
**Success Criteria** (what must be TRUE):
  1. Dashboard shows "History" navigation link
  2. History page displays last 30 days of briefings
  3. User can click date to view past briefing
  4. Read/unread state persists in history view
  5. Briefings older than 30 days are archived (not deleted)
**Plans**: TBD

Plans:
- TBD (created during /gsd:plan-phase 7)

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3 → 4 → 5 → 6 → 7

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Authentication | 0/? | Not started | - |
| 2. Database & Models | 0/? | Not started | - |
| 3. Background Job Infrastructure | 0/? | Not started | - |
| 4. Briefing Generation (Mock) | 0/? | Not started | - |
| 5. Briefing Display | 0/? | Not started | - |
| 6. Scheduled Generation | 0/? | Not started | - |
| 7. Briefing History | 0/? | Not started | - |

---
*Created: 2026-02-10*
*Last updated: 2026-02-10*
