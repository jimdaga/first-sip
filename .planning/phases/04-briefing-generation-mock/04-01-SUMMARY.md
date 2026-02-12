---
phase: 04-briefing-generation-mock
plan: 01
subsystem: webhook-client
tags: [n8n-integration, webhook, mock-data, briefing-types]
dependency_graph:
  requires: [Phase 03 worker infrastructure]
  provides: [webhook.Client, BriefingContent types, n8n config fields]
  affects: [briefing generation handlers, worker task processing]
tech_stack:
  added: [internal/webhook package]
  patterns: [stub mode pattern, custom http.Client timeout]
key_files:
  created:
    - internal/webhook/types.go
    - internal/webhook/client.go
  modified:
    - internal/models/briefing.go
    - internal/config/config.go
decisions:
  - Use custom http.Client with 30s timeout (not http.DefaultClient) to prevent infinite hangs
  - Default N8N_STUB_MODE to true for safe development experience
  - Add 2s delay in stub mode to make polling UI visible during demos
  - Keep datatypes.JSON for Content field (defer typed wrapper to Phase 5 if needed)
metrics:
  duration: 1 min
  tasks_completed: 2
  files_created: 2
  files_modified: 2
  commits: 2
  completed_at: 2026-02-12
---

# Phase 04 Plan 01: Webhook Client Foundation Summary

**One-liner:** Created n8n webhook client with typed BriefingContent (News/Weather/Work sections), stub mode with mock data, and X-N8N-SECRET authentication for production mode.

## Overview

Established the webhook integration layer that connects to n8n for briefing generation. Created strongly-typed domain models for briefing content (news items, weather info, work summary) and implemented a client that supports both stub mode (for development) and production mode (with authenticated HTTP requests).

This foundation enables Phase 04 Plan 02 to build handlers and worker logic without needing actual n8n infrastructure.

## Tasks Completed

### Task 1: Create webhook package with types and stub-mode client
**Files:** `internal/webhook/types.go`, `internal/webhook/client.go`
**Commit:** `7287d18`

Created `internal/webhook/types.go` with four exported types:
- `BriefingContent` - top-level structure with News, Weather, Work sections
- `NewsItem` - Title, Summary, URL (for news articles)
- `WeatherInfo` - Location, Temperature, Condition
- `WorkSummary` - TodayEvents, TomorrowTasks (string slices)

Created `internal/webhook/client.go` with:
- `Client` struct with baseURL, secret, httpClient, stubMode fields
- `NewClient()` constructor that creates custom `http.Client{Timeout: 30 * time.Second}` (critical: avoids infinite hangs from http.DefaultClient per research)
- `GenerateBriefing()` method with dual behavior:
  - **Stub mode:** Returns hardcoded mock data (2 news items, SF weather 65F, 3 today events, 3 tomorrow tasks) with 2s sleep to simulate processing and make polling UI visible
  - **Production mode:** POSTs to `{baseURL}/generate` with `X-N8N-SECRET` header, decodes JSON response into BriefingContent

**Verification:** `go build ./internal/webhook/...` and `go vet ./internal/webhook/...` passed

### Task 2: Extend Briefing model and config for n8n integration
**Files:** `internal/models/briefing.go`, `internal/config/config.go`
**Commit:** `0b88ff2`

Updated `internal/models/briefing.go`:
- Added `GeneratedAt *time.Time` field to track when briefing content was generated (between ErrorMessage and ReadAt)
- Kept existing `Content datatypes.JSON` field for storing marshaled BriefingContent

Updated `internal/config/config.go`:
- Added three new Config fields: `N8NWebhookURL`, `N8NWebhookSecret`, `N8NStubMode`
- Load from env: `N8N_WEBHOOK_URL`, `N8N_WEBHOOK_SECRET`, `N8N_STUB_MODE` (defaults to "true")
- Added `parseStubMode()` helper that returns true for "true" or "1"
- No production warnings for missing n8n config (stub mode is safe default)

**Verification:** `go build ./internal/models/... ./internal/config/...` and `go vet` passed

## Deviations from Plan

None - plan executed exactly as written.

## Key Decisions

1. **Custom http.Client timeout:** Used `&http.Client{Timeout: 30 * time.Second}` instead of `http.DefaultClient` to prevent infinite hangs (research pitfall 4 from 04-RESEARCH.md)

2. **Stub mode default:** Defaulted `N8N_STUB_MODE` to "true" for safe development experience - developers can run the app immediately without n8n infrastructure

3. **Demo-friendly delay:** Added `time.Sleep(2 * time.Second)` in stub mode so polling UI is visible during demos (without it, mock response would be instant and UI states would flash)

4. **Preserve datatypes.JSON:** Kept existing `Content datatypes.JSON` field instead of upgrading to `datatypes.JSONType[BriefingContent]` - the typed wrapper adds complexity and the raw JSON field works fine for Phase 4 scope

## Verification Results

All plan verification criteria passed:

- [x] `go build ./...` compiles entire project without errors
- [x] `go vet ./...` passes
- [x] `internal/webhook/types.go` defines BriefingContent, NewsItem, WeatherInfo, WorkSummary
- [x] `internal/webhook/client.go` has NewClient constructor and GenerateBriefing with stub mode branch
- [x] `internal/models/briefing.go` has GeneratedAt field
- [x] `internal/config/config.go` has N8NWebhookURL, N8NWebhookSecret, N8NStubMode fields

Must-haves verification:

- [x] Webhook client returns mock BriefingContent in stub mode without HTTP calls
- [x] Webhook client sends X-N8N-SECRET header when not in stub mode
- [x] BriefingContent struct has typed News, Weather, Work sections
- [x] Config reads N8N_WEBHOOK_URL, N8N_WEBHOOK_SECRET, N8N_STUB_MODE from environment

## Impact on Roadmap

**Unblocks:**
- Phase 04 Plan 02: Can now implement POST /briefings endpoint and worker task handler using webhook.Client
- Future briefing handlers can import and use typed BriefingContent without needing JSON parsing

**Dependencies satisfied:**
- Phase 03 worker infrastructure (complete)

**Next steps:**
- Plan 02 will create POST /briefings endpoint, implement worker task handler that calls webhook.Client, add UI "Generate" button, and create polling mechanism for status updates

## Testing Notes

No automated tests added in this plan (foundation layer). Plan 02 will add integration tests when handlers and worker logic are connected.

Manual verification available: Can instantiate `webhook.NewClient("", "", true)` and call `GenerateBriefing(ctx, userID)` to get mock data.

## Files Changed

**Created:**
- `internal/webhook/types.go` - 4 exported types with JSON tags
- `internal/webhook/client.go` - Client struct with stub/prod mode logic

**Modified:**
- `internal/models/briefing.go` - Added GeneratedAt field
- `internal/config/config.go` - Added 3 n8n config fields + parseStubMode helper

## Self-Check: PASSED

**Created files verification:**
```
FOUND: internal/webhook/types.go
FOUND: internal/webhook/client.go
```

**Modified files verification:**
```
FOUND: internal/models/briefing.go (GeneratedAt on line 26)
FOUND: internal/config/config.go (N8NWebhookURL on line 17)
```

**Commits verification:**
```
FOUND: 7287d18 (feat(04-01): create webhook package with types and stub-mode client)
FOUND: 0b88ff2 (feat(04-01): extend Briefing model and config for n8n integration)
```

All artifacts created as specified. Ready for Plan 02.
