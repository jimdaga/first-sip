---
phase: 09-crewai-sidecar-integration
plan: 01
subsystem: streams
tags: [redis-streams, crewai, async-messaging, producer-consumer]
dependency_graph:
  requires: [internal/plugins, internal/config]
  provides: [streams-publisher, streams-consumer, result-handler]
  affects: [cmd/server, worker-lifecycle]
tech_stack:
  added: [redis-streams, go-redis/v9]
  patterns: [consumer-groups, xreadgroup-xack, background-goroutine, graceful-shutdown]
key_files:
  created:
    - internal/streams/types.go
    - internal/streams/producer.go
    - internal/streams/consumer.go
    - internal/streams/handler.go
  modified:
    - cmd/server/main.go
    - go.mod
decisions:
  - Stream names follow colon pattern (plugin:requests, plugin:results)
  - Consumer groups named by role (go-workers, crewai-workers)
  - Schema version v1 embedded in stream messages for forward compatibility
  - Non-fatal error handling for result consumer (logs warning, continues)
  - StartResultConsumer follows same (stop func(), err error) pattern as worker lifecycle
  - Result handler uses PluginRunID field (not GORM ID) for external tracking
  - Failed handler calls leave message in PEL for retry (no ACK)
metrics:
  duration_minutes: 3
  tasks_completed: 3
  files_created: 4
  files_modified: 2
  completed_date: 2026-02-14
---

# Phase 09 Plan 01: Redis Streams Infrastructure Summary

**One-liner:** Go-side Redis Streams publisher/consumer for CrewAI plugin execution requests and results with background result consumer in worker lifecycle.

## What Was Built

Created the `internal/streams` package providing bidirectional Redis Streams communication for plugin execution:

**Producer side (Go → CrewAI):**
- `Publisher` wrapping `go-redis/v9` XAdd for publishing to `plugin:requests` stream
- `PluginRequest` message type with plugin_run_id, plugin_name, user_id, settings
- Stream max length of 10,000 messages with approximate trimming

**Consumer side (CrewAI → Go):**
- `ResultConsumer` wrapping `go-redis/v9` XReadGroup for consuming from `plugin:results` stream
- Consumer group pattern with group name `go-workers` (Python side uses `crewai-workers`)
- XReadGroup + XACK pattern for reliable message processing
- Failed handler calls leave messages in PEL for retry (not ACKed)

**Result handler:**
- `HandlePluginResult` closure that updates `PluginRun` database records based on stream results
- Maps `completed` status → PluginRunStatusCompleted + output JSONB
- Maps `failed` status → PluginRunStatusFailed + error_message text
- Uses `PluginRunID` field (UUID string) for external tracking, not GORM ID

**Lifecycle integration:**
- `StartResultConsumer(redisURL, db)` returns `(stop func(), err error)` matching worker pattern
- Started in both worker mode and embedded dev mode in `cmd/server/main.go`
- Graceful shutdown via context cancellation and consumer close
- Non-fatal error handling (logs warning, doesn't crash app if streams consumer fails)

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Redis Streams types and publisher | cddc469 | types.go, producer.go, go.mod |
| 2 | Redis Streams consumer and result handler | c27f8f1 | consumer.go, handler.go |
| 3 | Wire ResultConsumer into application lifecycle | 1357a19 | cmd/server/main.go |

## Deviations from Plan

None - plan executed exactly as written.

## Testing & Verification

**Compilation:**
- `go build ./internal/streams/...` ✓
- `go build ./cmd/server/...` ✓
- `go vet ./internal/streams/... ./cmd/server/...` ✓

**Verification checks:**
- go-redis/v9 promoted to direct dependency in go.mod ✓
- types.go exports PluginRequest, PluginResult, stream/group constants ✓
- producer.go exports Publisher, NewPublisher, PublishPluginRequest ✓
- consumer.go exports ResultConsumer, NewResultConsumer, ConsumeResults, StartResultConsumer ✓
- handler.go exports HandlePluginResult and imports internal/plugins ✓
- cmd/server/main.go calls StartResultConsumer in both worker and embedded modes (2 calls) ✓
- Graceful shutdown calls stopResultConsumer() in embedded mode ✓

## Technical Notes

**Stream architecture:**
- Two streams: `plugin:requests` (Go → Python) and `plugin:results` (Python → Go)
- Each stream has its own consumer group for horizontal scaling
- Messages include schema_version field for protocol evolution
- XReadGroup blocking reads (5s timeout) for efficiency
- Context-aware loops for graceful shutdown

**Message processing:**
- Consumer reads in batches (count=10) for throughput
- Each message unmarshaled to PluginResult struct
- Handler success → XACK (remove from stream)
- Handler failure → no XACK (message stays in PEL for retry)
- Invalid payloads logged but skipped (no retry)

**Error handling philosophy:**
- Result consumer failures are non-fatal (v1.0 briefings still work)
- Logs warnings instead of crashing app
- Graceful degradation if Redis Streams unavailable
- Follows same pattern as plugin discovery (non-blocking failures)

## Integration Points

**Upstream dependencies:**
- `internal/plugins` for PluginRun model and status constants
- `internal/config` for RedisURL configuration
- `gorm.io/gorm` for database operations
- `gorm.io/datatypes` for JSONB field updates

**Downstream usage (future plans):**
- Plan 09-02 will create Python CrewAI sidecar that consumes plugin:requests
- Plan 09-03 will integrate Publisher into worker tasks (replace n8n webhook)
- Phase 10 will use this for scheduled plugin execution

**Lifecycle integration:**
- Worker mode: starts scheduler + result consumer, then blocks on worker.Run()
- Embedded dev mode: starts worker + scheduler + result consumer in goroutines
- Shutdown sequence: HTTP → scheduler → result consumer → worker (proper ordering)

## Self-Check: PASSED

**Created files verified:**
- ✓ internal/streams/types.go exists
- ✓ internal/streams/producer.go exists
- ✓ internal/streams/consumer.go exists
- ✓ internal/streams/handler.go exists

**Modified files verified:**
- ✓ cmd/server/main.go imports streams package
- ✓ go.mod has redis/go-redis/v9 in direct requires

**Commits verified:**
- ✓ cddc469 (Task 1: types + publisher)
- ✓ c27f8f1 (Task 2: consumer + handler)
- ✓ 1357a19 (Task 3: lifecycle wiring)

All files exist, all commits present in git log, all verification steps passed.
