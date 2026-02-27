# First Sip — Daily Briefing

## What This Is

A web application that generates personalized daily briefings through a plugin-based architecture. Each briefing type is an independent plugin with its own CrewAI workflow, per-user schedule, and configuration. Users log in with Google OAuth, configure which plugins they want, set their timezone and schedule preferences, and see their latest briefings on a tile-based dashboard — all through a liquid glass UI built with Go, Templ, and HTMX.

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
- ✓ HTMX polling for real-time briefing status updates — v1.0
- ✓ End-to-end flow: login → generate → see briefing — v1.0
- ✓ Automated daily briefing via Asynq cron scheduler — v1.0
- ✓ Briefing history with date grouping and HTMX pagination — v1.0
- ✓ Read/unread tracking with click-to-mark-read — v1.0
- ✓ Plugin metadata in YAML with directory discovery and registry — v1.1
- ✓ Plugin, UserPluginConfig, PluginRun database models with migrations — v1.1
- ✓ JSON Schema settings definition per plugin with validation — v1.1
- ✓ CrewAI sidecar with Redis Streams communication pipeline — v1.1
- ✓ Multi-agent workflow execution (researcher → writer → reviewer) — v1.1
- ✓ Per-user per-plugin scheduling with timezone-aware cron evaluation — v1.1
- ✓ Global cron scheduler replaced by database-backed per-user schedules — v1.1
- ✓ CSS Grid tile-based dashboard with expand-in-place content — v1.1
- ✓ Dynamic settings UI with JSON Schema-driven forms and type coercion — v1.1
- ✓ Account tier scaffolding with free/pro tiers and constraint enforcement — v1.1
- ✓ Daily news digest example plugin end-to-end — v1.1

### Active

(None — ready for next milestone requirements)

### Out of Scope

- Real-time SSE/WebSocket updates — HTMX polling works well for briefing status
- iOS mobile app — long-term goal, web-first approach
- Multi-tenancy / team features — designed for it, defer building
- Email/push notifications — not needed for personal use phase
- Admin panel — single-user context doesn't require it
- Cross-source synthesis ("3 sources mention X") — validate basic summaries first
- Interactive chat/follow-up on briefing items — major scope expansion
- Voice/TTS briefing — different UX paradigm
- Payment / Stripe integration — tier scaffolding only, billing deferred
- Plugin marketplace / third-party plugins — internal plugins only, marketplace needs security review
- Plugin detail pages / drill-down views — tiles with summary only for now
- CrewAI memory backend (context across briefings) — high complexity, defer
- Multiple plugins beyond daily news digest — prove architecture first, add more in v1.2+

## Context

**Shipped v1.1** with ~13,800 LOC across Go (9,428), Templ (1,538), CSS (2,254), and Python (568).

**Tech stack:** Go 1.24, Gin, Templ, HTMX 2.0, GORM (PostgreSQL), Asynq (Redis), Goth (Google OAuth), CrewAI (Python/FastAPI sidecar), custom liquid glass CSS design system.

**Architecture:** `/cmd/server/main.go` (entry point with embedded worker), `/internal/auth` (OAuth), `/internal/config`, `/internal/database` (GORM + migrations 000001-000009), `/internal/models` (User, AuthIdentity, Plugin, UserPluginConfig, PluginRun, AccountTier), `/internal/plugins` (metadata, discovery, registry), `/internal/worker` (Asynq tasks + Redis Streams publisher/consumer), `/internal/dashboard` (tile view models + handlers), `/internal/settings` (JSON Schema forms + handlers), `/internal/tiers` (TierService), `/internal/templates` (Templ layouts + components), `/sidecar` (FastAPI + CrewAI executor), `/plugins/daily-news-digest` (example plugin with CrewAI crew).

**Current state:** v1.1 shipped with full plugin architecture. Monolithic briefing generation replaced by per-plugin CrewAI workflows. Global cron replaced by per-user per-plugin scheduling. Dashboard is tile-based with expand-in-place content. Settings page generates forms dynamically from JSON Schema. Account tiers scaffolded (free/pro limits enforced, no payment).

**Known issues / tech debt:**
- Session store is cookie-based (Redis-backed sessions deferred)
- `pluginRegistry` variable in main.go assigned but never consumed by handlers (dead allocation)
- `computeNextRun` display uses plain cron parse vs scheduler's `CRON_TZ` prefix (potential DST edge case for display only)
- SeedDevData dev user has NULL AccountTierID (relies on TierService NULL fallback)
- Human verification needed: Docker sidecar E2E tests require running containers

## Constraints

- **Tech stack**: Go 1.24 / Gin / Templ / HTMX / GORM / Asynq / Goth / CrewAI / Tailwind+custom CSS
- **Infrastructure**: Must work with existing Kubernetes + Helm + ArgoCD deployment pipeline
- **Database**: Postgres everywhere (production and local dev via Docker Compose)
- **AI Sidecar**: Python/FastAPI with CrewAI, communicates via Redis Streams

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
| CrewAI over n8n | Code-first AI workflows (Python), large community, easily bundled per plugin | ✓ Good — clean separation, scales independently, YAML-configured crews |
| Redis Streams for Go ↔ CrewAI | No new infra, async decoupling, independent scaling | ✓ Good — reliable pub/sub with consumer groups and XACK |
| Database-backed scheduling | Per-minute scheduler queries DB for due pairs (NOT per-user Asynq cron) | ✓ Good — avoids O(users × plugins) Redis entries, timezone-aware |
| kaptinlin/jsonschema for validation | Google's official JSON Schema for Go, SetPreserveExtra for x-extensions | ✓ Good — robust validation with inline error display |
| JSONB for flexible storage | Capabilities, configs, settings, run output all JSONB | ✓ Good — query flexibility without schema migrations per plugin |
| Non-fatal plugin initialization | App serves v1.0 features if plugins/sidecar fail | ✓ Good — graceful degradation, never blocks core functionality |
| AccountTierID as nullable pointer | NULL means pre-migration user, falls back to free tier | ✓ Good — backwards compatible, no data migration needed |

---
*Last updated: 2026-02-27 after v1.1 milestone*
