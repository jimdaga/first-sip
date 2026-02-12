# Phase 3: Background Job Infrastructure - Context

**Gathered:** 2026-02-12
**Status:** Ready for planning

<domain>
## Phase Boundary

Asynchronous task processing using Asynq with Redis. The worker runs in the same binary as the web server (flag-based mode switching), processing tasks enqueued from HTTP handlers. This phase establishes the job infrastructure that Phases 4-7 build on for briefing generation, scheduling, and status tracking. Does NOT implement any specific task types — that's Phase 4.

</domain>

<decisions>
## Implementation Decisions

### Worker process model
- Same binary, mode flag (e.g., `--worker`) — NOT a separate binary
- Code structured cleanly in `internal/worker/` so it's easy to spin out to a separate binary later
- Local dev: `make dev` starts both web server and worker together in one process
- K8s: Separate Deployment for the worker (same container image, different entrypoint/flag)
- Asynq concurrency: 5 concurrent task processors (low, appropriate for personal use)

### Retry & failure policy
- Max 3 retries before dead letter queue
- Dead letter handling: log the failure and update briefing status to 'failed' with error message — no external notifications
- Single default queue — no priority queues needed for current task types

### Claude's Discretion — task timeout
- Claude picks an appropriate task timeout based on mock vs real webhook patterns

### Redis strategy
- Add Redis to existing docker-compose.yml alongside Postgres — one `docker compose up -d` starts everything
- Redis data persists via named volume (queued tasks survive container restart)
- Keep cookie-based sessions — do NOT move sessions to Redis (cookie sessions work fine)
- Production: in-cluster Redis pod (not external managed service), app reads REDIS_URL env var
- REDIS_URL env var pattern, same as DATABASE_URL — add to env.local

### Monitoring & visibility
- Asynqmon web dashboard in Docker Compose — accessible at localhost:8081 for dev
- Verbose logging in local dev (every task start, completion, retry, failure)
- Configurable log level via environment variable (verbose for dev, quieter for prod)
- /health endpoint stays web-server-only — worker health checked separately via K8s probe
- JSON structured logging in production, plain text in dev — configurable via environment

</decisions>

<specifics>
## Specific Ideas

- "Same binary as long as the code is structured in a way it's easy to spin out later" — clean separation in internal/worker/ is key
- Asynqmon for visual task monitoring during development
- Log level and format both configurable per environment

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 03-background-job-infrastructure*
*Context gathered: 2026-02-12*
