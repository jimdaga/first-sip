# First Sip — Daily Briefing

## What This Is

A web application that generates personalized daily briefings by pulling from multiple sources (news, work context, custom data) through n8n workflows. Users log in with Google, trigger a briefing, and see an AI-assembled summary. Built with Go, Templ, and HTMX for a server-rendered experience with dynamic UI updates. Designed for personal use first, with multi-user growth in mind.

## Core Value

A user can click "Generate" and receive a multi-source daily briefing without leaving the app — the background processing, source aggregation, and status tracking all happen seamlessly.

## Requirements

### Validated

- ✓ Go 1.23 project structure with `cmd/` and `internal/` layout — existing
- ✓ Multi-stage Docker build (golang:1.23-alpine builder, alpine:3.19 runtime) — existing
- ✓ Kubernetes deployment via Helm charts and ArgoCD — existing
- ✓ Health check endpoint at `GET /health` — existing
- ✓ CI pipeline with golangci-lint — existing
- ✓ Makefile-based build automation — existing
- ✓ Dev/prod environment separation via Helm values — existing

### Active

- [ ] Replace net/http with Gin router and Templ templates
- [ ] HTMX-driven dashboard with DaisyUI styling
- [ ] Google OAuth login via Goth
- [ ] User model with encrypted OAuth token fields (GORM)
- [ ] Briefing model with status tracking (Pending/Completed/Failed)
- [ ] Postgres database with GORM migrations
- [ ] Redis + Asynq background job infrastructure
- [ ] `briefing:generate` Asynq task with n8n webhook stub (mock data)
- [ ] HTMX polling for briefing status updates
- [ ] End-to-end clickable demo: login → generate → see briefing

### Out of Scope

- Real n8n workflow implementation — Go app comes first, n8n built separately later
- iOS mobile app — long-term goal, not this milestone
- Multi-tenancy / team features — design for it, don't build it yet
- Real-time SSE/WebSocket updates — HTMX polling sufficient for v1
- Email notifications — not needed for personal use phase
- Admin panel — single-user context doesn't require it

## Context

**Existing codebase:** Minimal Go service with health endpoint, CI/CD pipeline (GitHub Actions), Helm chart, and ArgoCD deployment. No external dependencies — uses only Go standard library. The internals will be replaced with the new stack while preserving CI/CD and deployment infrastructure.

**Architecture direction:** Structured Go layout — `/cmd` (entry points), `/internal/handlers` (HTTP/Gin), `/internal/models` (GORM structs), `/internal/services` (business logic/n8n calls), `/internal/templates` (Templ files), `/internal/worker` (Asynq background processing).

**n8n integration pattern:** Go app enqueues Asynq tasks → worker calls n8n webhook with `X-N8N-SECRET` header → n8n processes and returns briefing content. For bootstrap, worker returns mock data to prove the full flow.

**Frontend approach:** Server-rendered Templ components + HTMX for dynamic behavior. Templ gives type-safe, component-style HTML (like React but Go). HTMX handles triggering generation (`hx-post`), polling for status (`hx-get` with `hx-trigger="every 5s"`), and swapping content. DaisyUI provides component styling on top of Tailwind.

**Security considerations:** OAuth tokens encrypted via GORM hooks (`BeforeSave`). n8n webhook calls include `X-N8N-SECRET` header. Design with multi-user in mind even for v1.

## Constraints

- **Tech stack**: Go 1.23+ / Gin / Templ / HTMX / GORM / Asynq / Goth / Tailwind+DaisyUI — all specified, no substitutions
- **Infrastructure**: Must work with existing Kubernetes + Helm + ArgoCD deployment pipeline
- **Database**: Postgres (production), SQLite acceptable for local dev
- **Bootstrap scope**: Clickable demo — login, generate, see mock briefing. Not production-ready.
- **Existing code**: Keep CI/CD and Helm charts, replace Go internals with new architecture

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Templ over html/template | Type-safe, component-style HTML; better DX for complex UIs | — Pending |
| HTMX polling over SSE | Simpler implementation; SSE can be added later if needed | — Pending |
| Asynq over goroutines | Persistent job queue with Redis; survives restarts; ready for scale | — Pending |
| GORM hooks for encryption | Transparent encryption at model layer; tokens never stored in plaintext | — Pending |
| Replace internals, keep infra | Existing CI/CD and Helm are working; only Go code needs restructuring | — Pending |
| Design for multi-user | OAuth and user model from day one; avoid painful refactor later | — Pending |

---
*Last updated: 2026-02-10 after initialization*
