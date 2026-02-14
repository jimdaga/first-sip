# First Sip — Daily Briefing

## What This Is

A web application that generates personalized daily briefings through a plugin-based architecture. Each briefing type is an independent plugin with its own CrewAI workflow, schedule, and configuration. Users log in with Google OAuth, configure which plugins they want, and see their latest briefings on a tile-based dashboard — all through a liquid glass UI built with Go, Templ, and HTMX.

## Core Value

A user's configured briefing plugins run on schedule and their latest results appear automatically on a tile-based dashboard — no manual action needed to receive fresh, personalized briefings every day.

## Requirements

### Validated

- ✓ Go project structure with `cmd/` and `internal/` layout — existing
- ✓ Multi-stage Docker build — existing
- ✓ Kubernetes deployment via Helm charts and ArgoCD — existing
- ✓ Health check endpoint at `GET /health` — existing
- ✓ CI pipeline with golangci-lint — existing
- ✓ Makefile-based build automation — existing
- ✓ Dev/prod environment separation via Helm values — existing
- ✓ Gin router with Templ templates replacing net/http — v1.0
- ✓ HTMX-driven dashboard with liquid glass design system — v1.0
- ✓ Google OAuth login via Goth with session persistence — v1.0
- ✓ User model with AES-256-GCM encrypted OAuth tokens (GORM) — v1.0
- ✓ Briefing model with status tracking (Pending/Completed/Failed) — v1.0
- ✓ Postgres database with GORM auto-migrations — v1.0
- ✓ Redis + Asynq background job infrastructure with retry policy — v1.0
- ✓ `briefing:generate` Asynq task with n8n webhook stub (mock data) — v1.0
- ✓ HTMX polling for real-time briefing status updates — v1.0
- ✓ End-to-end flow: login → generate → see briefing — v1.0
- ✓ Automated daily briefing via Asynq cron scheduler — v1.0
- ✓ Briefing history with date grouping and HTMX pagination — v1.0
- ✓ Read/unread tracking with click-to-mark-read — v1.0

### Active

**Current Milestone: v1.1 Plugin Architecture**

- Plugin framework with YAML metadata and settings schema
- Centralized per-user, per-plugin scheduler replacing global cron
- Tile-based homepage with plugin tiles showing latest briefing and status
- Full settings page with plugin management, enable/disable, schedule config, manual trigger
- Daily news digest plugin with real CrewAI workflow integration
- Basic account tier scaffolding (tier field, limit checks, no enforcement yet)

### Out of Scope

- Real-time SSE/WebSocket updates — HTMX polling works well for briefing status
- iOS mobile app — long-term goal, web-first approach
- Multi-tenancy / team features — designed for it, defer building
- Email notifications — not needed for personal use phase
- Admin panel — single-user context doesn't require it
- Cross-source synthesis ("3 sources mention X") — validate basic summaries first
- Interactive chat/follow-up on briefing items — major scope expansion
- Voice/TTS briefing — different UX paradigm
- Push notifications — defeats batching purpose
- Multiple plugins beyond daily news digest — prove architecture first, add more plugins in v1.2+
- Payment / tier enforcement — scaffolding only this milestone
- Plugin marketplace / third-party plugins — internal plugins only
- Plugin detail pages / drill-down views — tiles with summary only for now

## Context

**Shipped v1.0** with ~4,000 LOC across Go (2,749), Templ (357), and CSS (912).

**Tech stack:** Go 1.24, Gin, Templ, HTMX 2.0, GORM (PostgreSQL), Asynq (Redis), Goth (Google OAuth), custom liquid glass CSS design system.

**Architecture:** `/cmd/server/main.go` (entry point with embedded worker), `/internal/auth` (OAuth), `/internal/config`, `/internal/database` (GORM + migrations), `/internal/models` (User, AuthIdentity, Briefing), `/internal/briefings` (handlers), `/internal/templates` (Templ layouts + components), `/internal/worker` (Asynq server + tasks). AI workflows powered by CrewAI (Python sidecar service).

**Current state:** v1.0 shipped with monolithic briefing generation (mock data, global cron). v1.1 redesigns around plugin-based architecture — each briefing type becomes an independent plugin with its own CrewAI workflow, schedule, and user configuration.

**Known issues / tech debt:**
- n8n webhook client to be replaced by CrewAI integration — v1.1 adds real workflow for news digest
- No per-user schedule configuration (global cron only) — v1.1 replaces with per-plugin scheduler
- Session store is cookie-based (Redis-backed sessions deferred)
- No error recovery UI for failed briefings (just shows "Failed" badge)
- Monolithic briefing model needs refactoring to support plugin-sourced briefings

## Constraints

- **Tech stack**: Go 1.24 / Gin / Templ / HTMX / GORM / Asynq / Goth / Tailwind+custom CSS
- **Infrastructure**: Must work with existing Kubernetes + Helm + ArgoCD deployment pipeline
- **Database**: Postgres everywhere (production and local dev via Docker Compose)
- **Existing code**: CI/CD and Helm charts preserved, Go internals fully replaced

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Templ over html/template | Type-safe, component-style HTML; better DX for complex UIs | ✓ Good — clean component composition, compile-time safety |
| HTMX polling over SSE | Simpler implementation; SSE can be added later if needed | ✓ Good — polling works well for briefing status, simple to implement |
| Asynq over goroutines | Persistent job queue with Redis; survives restarts; ready for scale | ✓ Good — embedded mode great for dev, standalone for production |
| GORM hooks for encryption | Transparent encryption at model layer; tokens never stored in plaintext | ✓ Good — AES-256-GCM with random nonce, idempotent encrypt/decrypt |
| Replace internals, keep infra | Existing CI/CD and Helm are working; only Go code needs restructuring | ✓ Good — preserved deployment pipeline, clean rebuild of application |
| Design for multi-user | OAuth and user model from day one; avoid painful refactor later | ✓ Good — user model + auth identity supports multiple providers |
| Liquid glass over DaisyUI | Custom design system with glass morphism aesthetic | ✓ Good — distinctive visual identity, warm coffee-inspired tones |
| Embedded worker in dev mode | Single process for HTTP + worker + scheduler | ✓ Good — eliminates tmux/multiple terminal requirement |
| Cookie sessions, defer Redis | Simplest session store for v1 personal use | ⚠️ Revisit — move to Redis sessions for multi-instance deployment |
| Global cron schedule | Single schedule for all users via env var | ⚠️ Revisit — per-user schedules needed for multi-user |
| CrewAI over n8n | Code-first AI workflows (Python), large community, easily bundled per plugin | Decision — replaces n8n webhook stub for v1.1 plugin architecture |

---
*Last updated: 2026-02-13 after v1.1 milestone start*
