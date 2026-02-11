# Codebase Concerns

**Analysis Date:** 2026-02-10

## Tech Debt

**Hardcoded server port:**
- Issue: The HTTP server is hardcoded to listen on `:8080` without configuration flexibility
- Files: `cmd/server/main.go` (line 14-15)
- Impact: Cannot run multiple instances on different ports; requires code changes to use different ports in development or testing
- Fix approach: Extract port to environment variable (e.g., `PORT` or `HTTP_PORT`) with a default fallback. Update main.go to read from environment and pass to ListenAndServe.

**Missing graceful shutdown:**
- Issue: The server has no graceful shutdown mechanism for stopping in-flight requests
- Files: `cmd/server/main.go`
- Impact: Kubernetes pod termination (SIGTERM) will forcefully kill connections; deployments cannot safely roll updates
- Fix approach: Add signal handler (syscall.SIGTERM, syscall.SIGINT) that calls server.Shutdown() with context timeout. Use http.Server with http.ListenAndServe wrapper or directly instantiate *http.Server with proper shutdown logic.

**Unused directory structure:**
- Issue: `pkg/` directory exists but is empty (only contains `.gitkeep`); increases confusion about package organization
- Files: `pkg/.gitkeep`
- Impact: Unclear where shared libraries should go; developers may create packages at wrong level
- Fix approach: Either populate `pkg/` with actual shared code or document in README that only `internal/` is used for packages. Remove `.gitkeep` if no shared packages are planned.

**Missing configuration management:**
- Issue: No centralized config system; settings are hardcoded or missing entirely
- Files: `cmd/server/main.go`, `charts/first-sip/values.yaml`
- Impact: Application cannot be configured for different environments without code changes; no log level, timeout, or feature flags
- Fix approach: Implement config package that reads environment variables and config files. Use a library like viper or implement custom struct unmarshaling.

## Known Bugs

**Empty error handling:**
- Issue: Server startup errors go directly to log.Fatal without context about what failed
- Files: `cmd/server/main.go` (line 16)
- Impact: When ListenAndServe fails, error message is minimal; difficult to diagnose port conflicts or permission issues
- Workaround: Check logs carefully; use `sudo lsof -i :8080` to debug port conflicts
- Fix approach: Wrap error with context: `if err != nil { log.Fatalf("failed to start server: %v", err) }`

## Security Considerations

**No request validation:**
- Risk: Health endpoint accepts any HTTP method and request body without validation
- Files: `internal/health/handler.go`
- Current mitigation: Only `/health` endpoint exists; limited attack surface
- Recommendations: Validate request method (GET only), reject requests with body content, set strict Content-Type validation. Use middleware to enforce these patterns as the API grows.

**Missing HTTP security headers:**
- Risk: No X-Frame-Options, X-Content-Type-Options, or other security headers set
- Files: `internal/health/handler.go` (line 6)
- Current mitigation: Health endpoint is low-risk; main application may not exist yet
- Recommendations: Add middleware to inject security headers (X-Frame-Options: DENY, X-Content-Type-Options: nosniff, Strict-Transport-Security, Content-Security-Policy). Use package like github.com/urfave/negroni or similar.

**Unauthenticated health endpoint:**
- Risk: `/health` endpoint is publicly accessible without auth; allows information leakage about system state
- Files: `cmd/server/main.go`, `internal/health/handler.go`
- Current mitigation: Health endpoint only returns static status; no sensitive data exposed
- Recommendations: For production, consider IP allowlisting health checks or requiring basic auth. Document this as a known open endpoint.

**Container runs as non-root but with permissive security context:**
- Risk: Kubernetes deployment has empty `securityContext: {}` allowing unnecessary capabilities
- Files: `charts/first-sip/templates/deployment.yaml` (line 22-23, 37-39)
- Current mitigation: Dockerfile adds non-root user (appuser); container still allows capabilities escalation
- Recommendations: Set explicit securityContext in values.yaml:
  ```yaml
  securityContext:
    allowPrivilegeEscalation: false
    readOnlyRootFilesystem: true
    runAsNonRoot: true
    capabilities:
      drop:
        - ALL
  ```

## Performance Bottlenecks

**No request timeout configuration:**
- Problem: Server has unlimited read/write timeouts; slow clients can exhaust server resources
- Files: `cmd/server/main.go` (line 15)
- Cause: Using basic http.ListenAndServe without Server configuration
- Improvement path: Create http.Server struct with ReadTimeout, WriteTimeout, IdleTimeout set (e.g., 15s, 10s, 60s). Use server.ListenAndServe() instead of http.ListenAndServe().

**Single replica default:**
- Problem: Helm default is `replicaCount: 1`; single container can't handle traffic spikes
- Files: `charts/first-sip/values.yaml` (line 1)
- Cause: Conservative defaults; autoscaling disabled
- Improvement path: Set sensible defaults (minReplicas: 2-3) or enable autoscaling in production values with CPU target (80%). Document recommendation in README.

**No resource limits defined:**
- Problem: Kubernetes deployment has empty `resources: {}`; pods can consume unlimited CPU/memory
- Files: `charts/first-sip/values.yaml` (line 42), `charts/first-sip/templates/deployment.yaml` (line 55-58)
- Cause: Defaults not set; requires manual configuration per environment
- Improvement path: Set reasonable defaults in values.yaml:
  ```yaml
  resources:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 500m
      memory: 512Mi
  ```

## Fragile Areas

**Health endpoint implementation:**
- Files: `internal/health/handler.go`
- Why fragile: Direct `w.Write()` call could panic if write fails; no error handling. String concatenation for JSON response is fragile.
- Safe modification: Use `json.NewEncoder(w).Encode()` for proper JSON encoding with error handling. Add unit tests for malformed responses.
- Test coverage: Only basic test present; missing edge cases (write failures, nil writer)

**Server startup sequence:**
- Files: `cmd/server/main.go`
- Why fragile: No dependency initialization; as code grows, order matters. log.Fatal() kills process ungracefully.
- Safe modification: Introduce proper initialization chain: config → logging → handlers → server. Use context for lifecycle management.
- Test coverage: No integration tests for server startup

**Helm chart values:**
- Files: `charts/first-sip/values.yaml`
- Why fragile: Many defaults are empty or disabled (ingress, resources, security context, autoscaling). Unclear which are required vs optional.
- Safe modification: Document each value with comments. Provide example values-production.yaml. Add Helm lint validation.
- Test coverage: No helm template tests; chart-releaser only validates syntax

## Scaling Limits

**Single health endpoint:**
- Current capacity: Can handle hundreds of requests/sec on a single container (Go is fast)
- Limit: No caching layer; every health check hits application logic
- Scaling path: Add in-memory caching with TTL for health status. Use middleware to cache responses for 1-5s.

**No logging infrastructure:**
- Current capacity: Logs go to stdout only; single container
- Limit: Can't aggregate logs across multiple replicas; hard to debug production issues
- Scaling path: Integrate with observability stack (ELK, Datadog, etc). Implement structured logging (json format). Add contextual request IDs.

**Health probes may overload service:**
- Current capacity: Liveness every 10s, readiness every 5s per pod
- Limit: With many replicas (>50), health checks become significant traffic
- Scaling path: Consider health check endpoint on separate port/path to not compete with business logic. Cache health responses.

## Dependencies at Risk

**No go.sum file in repository:**
- Risk: No lock file means builds could be non-reproducible if go.mod transitive dependencies change
- Files: `go.mod` exists but `go.sum` is missing (likely in .gitignore)
- Impact: `go mod download` commands may fetch different versions on different machines/times
- Migration plan: Commit go.sum to repository. Update .gitignore to track `go.sum`.

**Go 1.23 with minimal stdlib:**
- Risk: Early point release; may have unfixed bugs
- Impact: Stability concerns for production
- Migration plan: Consider Go 1.24+ once available for stability. Set minimum Go version in go.mod constraints.

**Alpine 3.19 base image:**
- Risk: No vulnerability scanning; may contain CVEs
- Impact: Container images deployed to production without security audit
- Migration plan: Use container scanning in CI (e.g., Trivy). Update base images in dependency scanning workflow.

## Missing Critical Features

**No structured logging:**
- Problem: Only `log.Println()` and `log.Fatal()` used; no context, levels, or timestamps
- Blocks: Can't troubleshoot production issues; can't filter by severity or component
- Fix approach: Add logging middleware. Use structured logger (logrus, zap, slog). Implement log level configuration.

**No metrics/observability:**
- Problem: Application has no instrumentation; can't measure requests, latency, or errors
- Blocks: No visibility into application health; can't debug performance issues
- Fix approach: Add Prometheus metrics middleware. Export request count, duration, status codes. Integrate with monitoring.

**No API documentation:**
- Problem: Only single health endpoint documented in README
- Blocks: Future endpoints will lack specification; API contract unclear
- Fix approach: Add OpenAPI/Swagger spec. Use go-swagger or similar for Go. Document in README and generate HTML docs.

**No configuration for environment-specific values:**
- Problem: No way to pass environment-specific settings (log level, timeouts, feature flags)
- Blocks: Can't run different configs in dev/staging/prod without multiple code branches
- Fix approach: Implement environment config pattern. Use 12-factor app principles (env vars). Document config in README.

## Test Coverage Gaps

**Limited health handler testing:**
- What's not tested: Edge cases (request with body, HEAD/POST methods), large responses, concurrent requests, writer errors
- Files: `internal/health/handler_test.go`
- Risk: Bugs in handler could go unnoticed; response mutation could break clients
- Priority: Medium - Basic functionality tested, but edge cases missing

**No server startup tests:**
- What's not tested: Port binding failures, graceful shutdown, signal handling
- Files: `cmd/server/main.go`
- Risk: Server could fail to start in production without detection; updates could cause downtime
- Priority: High - Critical path for application availability

**No integration tests:**
- What's not tested: Full request/response cycle with real HTTP server, handler composition
- Files: None exist
- Risk: Handlers could break when integrated; middleware ordering issues
- Priority: Medium - Low complexity for now, but needed before adding more endpoints

**No benchmark tests:**
- What's not tested: Performance regressions in handler or request path
- Files: None exist
- Risk: Performance degradation unnoticed; deployments could impact production latency
- Priority: Low - Not critical for MVP, but should add before scaling

---

*Concerns audit: 2026-02-10*
