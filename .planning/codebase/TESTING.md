# Testing Patterns

**Analysis Date:** 2026-02-10

## Test Framework

**Runner:**
- `go test` (built-in Go testing framework)
- Version: Go 1.23
- Config: None (uses convention over configuration)

**Assertion Library:**
- Standard Go `*testing.T` with manual assertions
- No external assertion library (std library only)

**Run Commands:**
```bash
make test                          # Run all tests with race detection and coverage
go test -v -race -coverprofile=coverage.out ./...  # Full command with coverage
go test ./...                      # Basic test run
go test -race ./...                # With race condition detection
go test -v ./...                   # Verbose output
```

**Coverage:**
- Coverage report generated to `coverage.out`
- View coverage: `go tool cover -html=coverage.out`
- Coverage enabled in all test runs (via Makefile and CI/CD)

## Test File Organization

**Location:**
- Co-located with source code in same package
- Pattern: `handler.go` paired with `handler_test.go`
- Both in `internal/health/` directory

**Naming:**
- Files: `*_test.go` suffix
- Functions: `Test[FunctionName]` (e.g., `TestHandler`)
- Follows Go standard testing convention

**Structure:**
```
internal/
└── health/
    ├── handler.go
    └── handler_test.go
```

## Test Structure

**Suite Organization:**
```go
func TestHandler(t *testing.T) {
	// Setup
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	// Execute
	Handler(w, req)

	// Assert
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}
```

**Patterns:**
- Setup: Create test doubles (request/recorder) at top
- Execute: Call the function under test
- Assert: Verify results with `t.Errorf()` for failures
- One test function per exported function

**Assertion Style:**
- Manual assertions using `if` + `t.Errorf()`
- Error messages include expected vs actual values
- Format: `t.Errorf("expected [expected], got [actual", value)`

## Mocking

**Framework:**
- `net/http/httptest` for HTTP mocking (standard library)
- No external mocking library

**Patterns:**
```go
// Request mocking
req := httptest.NewRequest(http.MethodGet, "/health", nil)

// Response mocking
w := httptest.NewRecorder()

// Execute handler with mocked request/response
Handler(w, req)

// Inspect recorded response
resp := w.Result()
```

**What to Mock:**
- HTTP requests: Use `httptest.NewRequest()`
- HTTP responses: Use `httptest.NewRecorder()`
- Only mock external dependencies (HTTP, database, file system)

**What NOT to Mock:**
- Internal package functions (test as-is)
- Standard library types (use real instances)
- Business logic (test actual behavior)

## Fixtures and Factories

**Test Data:**
- Currently inline in test functions
- Example: Hard-coded JSON response `{"status":"ok"}`
- No shared fixtures or factories yet

**Location:**
- Co-located in `*_test.go` files
- Would create `testdata/` directory for large fixtures if needed

**Pattern:**
```go
// Inline test data
expected := `{"status":"ok"}`
```

## Coverage

**Requirements:**
- Coverage enabled: `coverprofile=coverage.out` in test runs
- No specific coverage target enforced
- Report generated but not checked for minimum threshold

**View Coverage:**
```bash
go tool cover -html=coverage.out  # Opens HTML report
go tool cover -func=coverage.out  # Function-level coverage
```

## Test Types

**Unit Tests:**
- Scope: Single function testing
- Approach: Direct function calls with test doubles
- Example: `TestHandler` tests HTTP response generation
- Execution: `go test ./internal/health/...`

**Integration Tests:**
- Currently: None (service is minimal)
- Would test: Multiple packages working together
- Pattern: Would use real HTTP server with actual handlers

**E2E Tests:**
- Framework: Not used
- Could add: Docker container testing or API contract tests
- Current: Manual testing via `curl localhost:8080/health`

## Race Detection

**Usage:**
- Enabled by default in all test runs
- Flag: `-race` in Makefile
- Purpose: Detects concurrent access to shared memory

**Pattern:**
```bash
go test -race -v ./...
```

## Common Patterns

**Async Testing:**
- Not currently used (no goroutines in code)
- Would use channels and goroutines with proper cleanup

**Error Testing:**
```go
// Check specific error conditions
if resp.StatusCode != http.StatusOK {
	t.Errorf("expected status 200, got %d", resp.StatusCode)
}
```

**Multi-assertion Testing:**
```go
func TestHandler(t *testing.T) {
	// Setup and execute...

	// Multiple assertions in sequence
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: expected 200, got %d", resp.StatusCode)
	}

	if ct != "application/json" {
		t.Errorf("content-type: expected application/json, got %s", ct)
	}

	if body != expected {
		t.Errorf("body: expected %s, got %s", expected, body)
	}
}
```

## CI/CD Integration

**Test Execution:**
- Triggered on: Every release (publish-artifacts.yaml) and main branch push (publish-artifacts-latest.yaml)
- Command: `go test -v -race -coverprofile=coverage.out ./...`
- Must pass before Docker/Helm publishing

**Test Requirements:**
- All tests must pass
- Race detector must not report issues
- Coverage report generated (not enforced)

---

*Testing analysis: 2026-02-10*
