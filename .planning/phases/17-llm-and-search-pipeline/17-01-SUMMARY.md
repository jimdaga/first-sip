---
phase: 17-llm-and-search-pipeline
plan: 01
subsystem: worker
tags: [asynq, redis-streams, api-keys, llm, crewai, litellm]

# Dependency graph
requires:
  - phase: 16-api-key-management
    provides: apikeys.GetKeysForUser, models.UserAPIKey with AfterFind decrypt hook, models.User LLMPreferredProvider/LLMPreferredModel fields
provides:
  - handleExecutePlugin injects _llm_api_key, _llm_model, _tavily_api_key into Redis Streams payload Settings map
  - findAPIKey helper for matching keys by type and provider
  - Per-plugin _llm_model override respected (user-level default only fills when absent)
affects: [17-02-llm-and-search-pipeline, crewai-sidecar]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Graceful degradation on key fetch failure: log warn and continue without credentials (sidecar surfaces missing key errors)"
    - "Credential injection into map[string]interface{} settings before Redis Streams publish — no new struct fields needed"
    - "AfterFind hook on UserAPIKey transparently decrypts EncryptedValue before it reaches worker"

key-files:
  created: []
  modified:
    - internal/worker/worker.go

key-decisions:
  - "Graceful degrade on fetch failure — user fetch or key fetch errors log Warn and continue; sidecar will fail with a meaningful error if keys are missing"
  - "Per-plugin _llm_model override check: only set user-level default when map key is absent OR is empty string"
  - "findAPIKey iterates by index (range i) to return pointer to slice element safely"
  - "API key values never appear in any slog/logger call — CRITICAL security requirement enforced"

patterns-established:
  - "Key injection pattern: fetch user → fetch keys → findAPIKey by type+provider → inject into settings map"
  - "Settings map nil guard: initialize with make() before any injection to handle nil payloads"

requirements-completed:
  - LLM-01
  - LLM-03

# Metrics
duration: 1min
completed: 2026-03-02
---

# Phase 17 Plan 01: LLM and Search Pipeline — Key Injection Summary

**Worker-side credential injection: handleExecutePlugin now fetches decrypted LLM and Tavily API keys and injects _llm_api_key, _llm_model, and _tavily_api_key into the Redis Streams payload before publishing to the CrewAI sidecar.**

## Performance

- **Duration:** 1 min
- **Started:** 2026-03-02T21:07:51Z
- **Completed:** 2026-03-02T21:09:23Z
- **Tasks:** 1
- **Files modified:** 1

## Accomplishments
- Added `findAPIKey` helper that matches `models.UserAPIKey` by `KeyType` and `Provider` fields
- Modified `handleExecutePlugin` to fetch user record and API keys before publishing to Redis Stream
- Injects `_llm_api_key` (decrypted LLM key), `_llm_model` (LiteLLM provider/model string), and `_tavily_api_key` (decrypted Tavily key) into `payload.Settings`
- Respects per-plugin `_llm_model` override — only sets user-level default when absent or empty
- Credential fetch failures are non-fatal (warn log, continue); credentials never appear in any log output

## Task Commits

Each task was committed atomically:

1. **Task 1: Add API key injection to handleExecutePlugin** - `4b8061d` (feat)

**Plan metadata:** (docs commit below)

## Files Created/Modified
- `internal/worker/worker.go` - Added findAPIKey helper, apikeys import, and key injection block in handleExecutePlugin

## Decisions Made
- Graceful degradation chosen over hard failure on key fetch: the CrewAI sidecar will produce a clear error if credentials are missing at execution time, making failures visible without preventing the task from dispatching
- Per-plugin `_llm_model` check uses `!ok || existing == ""` to handle both absent key and explicitly empty string cases
- `findAPIKey` uses index-range iteration (`for i := range keys`) to return a pointer to the slice element rather than a copy

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Worker now injects all credentials the CrewAI sidecar needs at dispatch time
- Plan 17-02 can proceed: sidecar will receive `_llm_api_key`, `_llm_model`, and `_tavily_api_key` in the Settings map
- Blockers noted in STATE.md remain: timeout tuning for 2-5 minute CrewAI runs, key expiry surfacing in phase 18

## Self-Check: PASSED

- internal/worker/worker.go: FOUND
- 17-01-SUMMARY.md: FOUND
- Commit 4b8061d: FOUND
- _llm_api_key injection: FOUND (1 occurrence)
- _tavily_api_key injection: FOUND (1 occurrence)
- _llm_model injection: FOUND (3 occurrences — map write, override check, assignment)

---
*Phase: 17-llm-and-search-pipeline*
*Completed: 2026-03-02*
