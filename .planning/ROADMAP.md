# Roadmap: First Sip — Daily Briefing

## Milestones

- ✅ **v1.0 MVP** — Phases 1-7 (shipped 2026-02-13)
- ✅ **v1.1 Plugin Architecture** — Phases 8-15 (shipped 2026-02-27)
- 🚧 **v1.2 Live AI Generation** — Phases 16-19 (in progress)

## Phases

<details>
<summary>✅ v1.0 MVP (Phases 1-7) — SHIPPED 2026-02-13</summary>

- [x] Phase 1: Authentication (2/2 plans) — completed 2026-02-10
- [x] Phase 2: Database & Models (2/2 plans) — completed 2026-02-11
- [x] Phase 3: Background Job Infrastructure (2/2 plans) — completed 2026-02-12
- [x] Phase 4: Briefing Generation / Mock (2/2 plans) — completed 2026-02-12
- [x] Phase 5: Briefing Display (1/1 plan) — completed 2026-02-12
- [x] Phase 6: Scheduled Generation (1/1 plan) — completed 2026-02-13
- [x] Phase 7: Briefing History (1/1 plan) — completed 2026-02-13

</details>

<details>
<summary>✅ v1.1 Plugin Architecture (Phases 8-15) — SHIPPED 2026-02-27</summary>

- [x] Phase 8: Plugin Framework Foundation (3/3 plans) — completed 2026-02-14
- [x] Phase 9: CrewAI Sidecar Integration (4/4 plans) — completed 2026-02-18
- [x] Phase 10: Per-User Scheduling (2/2 plans) — completed 2026-02-22
- [x] Phase 11: Tile-Based Dashboard (3/3 plans) — completed 2026-02-22
- [x] Phase 12: Dynamic Settings UI (2/2 plans) — completed 2026-02-23
- [x] Phase 13: Account Tier Scaffolding (2/2 plans) — completed 2026-02-25
- [x] Phase 14: Integration Pipeline Fix (2/2 plans) — completed 2026-02-26
- [x] Phase 15: Verification & Documentation Closure (2/2 plans) — completed 2026-02-26

</details>

### 🚧 v1.2 Live AI Generation (In Progress)

**Milestone Goal:** Make the daily news digest produce real AI-generated content by connecting per-user API keys to the CrewAI sidecar with live web search.

#### Phase 16: API Key Management — COMPLETE 2026-03-02
**Goal**: Users can securely store and manage their LLM and search API keys
**Depends on**: Phase 15 (v1.1 complete)
**Requirements**: KEYS-01, KEYS-02, KEYS-03, KEYS-04, KEYS-05
**Success Criteria** (what must be TRUE):
  1. User can add an LLM provider API key (OpenAI, Anthropic, Groq, etc.) and it is stored encrypted
  2. User can add a Tavily search API key and it is stored encrypted
  3. User can view stored keys with values masked (e.g., sk-...xxxx)
  4. User can update or delete any stored key
  5. User can select their preferred LLM provider and model
**Plans**: 2/2 complete

Plans:
- [x] 16-01-PLAN.md — Data layer: UserAPIKey model, SQL migrations, encryption hooks, service CRUD, provider definitions
- [x] 16-02-PLAN.md — UI layer: API Keys settings page, handlers, sidebar/hub integration, HTMX interactions

#### Phase 17: LLM and Search Pipeline — COMPLETE 2026-03-03
**Goal**: API keys flow through Redis Streams to the CrewAI sidecar, enabling live LLM calls and web search
**Depends on**: Phase 16
**Requirements**: LLM-01, LLM-02, LLM-03, SRCH-01, SRCH-02, SRCH-03
**Success Criteria** (what must be TRUE):
  1. Redis Streams job payload carries the user's LLM API key and provider/model string
  2. CrewAI sidecar accepts LLM config and initializes the crew with the specified provider/model via LiteLLM
  3. User can override the LLM model per plugin via plugin settings
  4. When user has a Tavily key, the researcher agent searches via Tavily
  5. When user has no Tavily key, the researcher agent falls back to DuckDuckGo
  6. Search queries use the user's topic preferences from plugin settings
**Plans**: 3/3 complete

Plans:
- [x] 17-01-PLAN.md — Go-side key injection: fetch user API keys and LLM preferences, inject into Redis Streams payload settings
- [x] 17-02-PLAN.md — Sidecar LLM/search integration: executor credential extraction, LiteLLM provider/model config, Tavily/DuckDuckGo search tool selection, imperative crew.py rewrite
- [x] 17-03-PLAN.md — Gap closure: add _llm_model field to settings schema for per-plugin LLM model override (LLM-03)

#### Phase 18: Live Generation and Content Rendering
**Goal**: Daily news digest generates real AI content and briefing tiles display it as formatted Markdown
**Depends on**: Phase 17
**Requirements**: GEN-01, GEN-02, GEN-03, GEN-04
**Success Criteria** (what must be TRUE):
  1. Triggering a daily news digest generation calls the live LLM and produces real content
  2. Briefing tile displays generated content rendered as formatted Markdown (headings, links, lists)
  3. When generation fails, the briefing tile shows a clear error message rather than empty content
  4. The full path works end-to-end: scheduled trigger fires, CrewAI runs, content appears in dashboard tile
**Plans**: TBD

Plans:
- [ ] 18-01: End-to-end generation flow with real CrewAI execution and timeout handling
- [ ] 18-02: Markdown content rendering in briefing tiles and error state display

#### Phase 19: Legacy Cleanup
**Goal**: N8N webhook generation path is removed and the codebase has no dead webhook code
**Depends on**: Phase 18
**Requirements**: CLN-01, CLN-02, CLN-03
**Success Criteria** (what must be TRUE):
  1. The N8N webhook client and briefing:generate Asynq task are deleted from the codebase
  2. Webhook-related environment variables and configuration keys are removed
  3. Existing briefing history records remain readable and display correctly after the cleanup
**Plans**: TBD

Plans:
- [ ] 19-01: Remove N8N webhook client, briefing:generate task, and all webhook config references

## Progress

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. Authentication | v1.0 | 2/2 | Complete | 2026-02-10 |
| 2. Database & Models | v1.0 | 2/2 | Complete | 2026-02-11 |
| 3. Background Job Infrastructure | v1.0 | 2/2 | Complete | 2026-02-12 |
| 4. Briefing Generation (Mock) | v1.0 | 2/2 | Complete | 2026-02-12 |
| 5. Briefing Display | v1.0 | 1/1 | Complete | 2026-02-12 |
| 6. Scheduled Generation | v1.0 | 1/1 | Complete | 2026-02-13 |
| 7. Briefing History | v1.0 | 1/1 | Complete | 2026-02-13 |
| 8. Plugin Framework Foundation | v1.1 | 3/3 | Complete | 2026-02-14 |
| 9. CrewAI Sidecar Integration | v1.1 | 4/4 | Complete | 2026-02-18 |
| 10. Per-User Scheduling | v1.1 | 2/2 | Complete | 2026-02-22 |
| 11. Tile-Based Dashboard | v1.1 | 3/3 | Complete | 2026-02-22 |
| 12. Dynamic Settings UI | v1.1 | 2/2 | Complete | 2026-02-23 |
| 13. Account Tier Scaffolding | v1.1 | 2/2 | Complete | 2026-02-25 |
| 14. Integration Pipeline Fix | v1.1 | 2/2 | Complete | 2026-02-26 |
| 15. Verification & Documentation Closure | v1.1 | 2/2 | Complete | 2026-02-26 |
| 16. API Key Management | v1.2 | Complete    | 2026-03-02 | - |
| 17. LLM and Search Pipeline | v1.2 | Complete    | 2026-03-03 | 2026-03-03 |
| 18. Live Generation and Content Rendering | v1.2 | 0/2 | Not started | - |
| 19. Legacy Cleanup | v1.2 | 0/1 | Not started | - |

---
*Created: 2026-02-10*
*Last updated: 2026-03-03 — Phase 17 fully complete (Plan 03 gap closure: _llm_model field added to settings schema for per-plugin LLM model override; all KEYS-01 through SRCH-03 requirements Done)*
