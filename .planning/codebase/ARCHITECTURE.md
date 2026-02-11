# Architecture

**Analysis Date:** 2026-02-10

## Pattern Overview

**Overall:** Layered HTTP service with module-based organization

**Key Characteristics:**
- Standard Go project layout (cmd/, internal/, pkg/)
- HTTP handler-based request routing
- Minimal external dependencies
- Cloud-native ready (containerized, Kubernetes deployable)
- GitOps-driven deployment via ArgoCD

## Layers

**Command Layer (Entry Point):**
- Purpose: Application bootstrapping and HTTP server initialization
- Location: `cmd/server/main.go`
- Contains: Server setup, route registration
- Depends on: `internal/health` (handler package)
- Used by: Docker container entrypoint

**Handler Layer (Business Logic):**
- Purpose: HTTP request processing and response generation
- Location: `internal/health/handler.go`
- Contains: HTTP handler functions
- Depends on: Standard library (net/http)
- Used by: `cmd/server` (mux registration)

**Package Layer (Shared Code):**
- Purpose: Reusable functionality to be exposed across modules
- Location: `pkg/` (currently empty, reserved for future shared code)
- Contains: Utilities, helpers, domain models
- Used by: Any internal or cmd modules

## Data Flow

**HTTP Request Handling:**

1. Client sends GET request to `/health`
2. `cmd/server/main.go` receives request via registered route handler
3. Request routed to `internal/health/handler.go` Handler function
4. Handler sets response headers (Content-Type: application/json)
5. Handler writes JSON response body: `{"status":"ok"}`
6. HTTP 200 OK response returned to client

**Error Handling:**

- Server startup errors logged via `log.Fatal()`
- Handler errors propagated via HTTP status codes
- No custom error handling middleware; handlers control response status directly

## Key Abstractions

**HTTP Handler:**
- Purpose: Process HTTP requests and generate responses
- Examples: `internal/health/handler.go`
- Pattern: Standard `http.HandlerFunc` signature - `func(w http.ResponseWriter, r *http.Request)`

**Service Bootstrap:**
- Purpose: Initialize application dependencies and start server
- Examples: `cmd/server/main.go`
- Pattern: Simple initialization in main() with explicit error handling

## Entry Points

**HTTP Server:**
- Location: `cmd/server/main.go`
- Triggers: Application startup via binary execution or container run
- Responsibilities:
  - Create HTTP multiplexer (mux)
  - Register request handlers
  - Bind to port 8080
  - Listen for incoming connections

**Health Check Endpoint:**
- Location: `internal/health/handler.go`
- Route: `GET /health`
- Responsibilities:
  - Return JSON status response
  - Signal application liveness/readiness to orchestrators (Kubernetes)

## Error Handling

**Strategy:** Direct error propagation with logging

**Patterns:**
- Server initialization errors: Logged via `log.Fatal()` which terminates process
- HTTP handlers: Control response status codes directly (no error wrapping)
- No middleware for centralized error handling
- No custom error types or structured error handling

## Cross-Cutting Concerns

**Logging:** Standard library `log` package for simple console output

**Validation:** None implemented; health endpoint has no input validation requirements

**Authentication:** None implemented; health endpoint is public

**Health Probes:** Kubernetes liveness and readiness probes both target `/health` endpoint with configurable delays:
- Liveness: Initial delay 5s, period 10s
- Readiness: Initial delay 3s, period 5s

---

*Architecture analysis: 2026-02-10*
