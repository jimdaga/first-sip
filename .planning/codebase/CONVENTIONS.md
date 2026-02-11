# Coding Conventions

**Analysis Date:** 2026-02-10

## Naming Patterns

**Files:**
- Command binaries: `main.go` in package directories (e.g., `cmd/server/main.go`)
- Package handlers/logic: Named by function (e.g., `handler.go` for HTTP handlers)
- Tests: `*_test.go` suffix in same package (e.g., `handler_test.go`)

**Functions:**
- HTTP handlers use PascalCase: `Handler` - exported, standard http.Handler signature
- Package-private would use camelCase
- All public functions exported with capital letter (Go convention)
- Simple descriptive names: `Handler` rather than `HealthHandler` (package name provides context)

**Packages:**
- Lowercase, single word: `health`, `server`
- Organized by responsibility not type
- Placed in `cmd/` for executables, `internal/` for private packages, `pkg/` for public packages

**Variables:**
- Lowercase camelCase for local variables: `w`, `r`, `req`, `resp`, `ct`, `body`
- Short names acceptable when scope is limited (function-level)
- Receiver names: `w` for http.ResponseWriter, `r` for *http.Request (standard library convention)

**Types:**
- Would use PascalCase (e.g., `type ServerConfig struct`)
- Not yet in use in current codebase

## Code Style

**Formatting:**
- Go standard: `gofmt` (automatic formatting)
- Enforced via linting, no explicit configuration needed
- Line length: Standard Go defaults (80-120 character suggestion)
- Indentation: Tabs (Go standard)

**Linting:**
- Tool: `golangci-lint`
- Config: `.golangci.yml`
- Enabled linters:
  - `errcheck`: Ensures error values are checked
  - `govet`: Catches vet issues
  - `staticcheck`: Static analysis
  - `unused`: Detects unused code
  - `gosimple`: Simplification suggestions
  - `ineffassign`: Inefficient assignments
  - `typecheck`: Type errors
- Timeout: 5 minutes
- Exclude-use-default: false (uses all default exclusions)
- Run with: `make lint`

## Import Organization

**Order:**
1. Standard library imports first (grouped in parentheses when multiple)
2. External/third-party packages (none currently)
3. Local package imports (github.com/jimdaga/first-sip/...)

**Pattern:**
```go
import (
	"log"
	"net/http"

	"github.com/jimdaga/first-sip/internal/health"
)
```

**Path Aliases:**
- Not used in current codebase (would use full module path only)

## Error Handling

**Patterns:**
- Explicit error checking: `if err := operation(); err != nil`
- Panic for unrecoverable errors: `log.Fatal(err)` in main entry point
- No silent error dropping - all errors must be handled
- Error propagation via return values (Go idiomatic)

**Examples:**
- `cmd/server/main.go`: Uses `log.Fatal()` for server startup failures
- No custom error types currently used

## Logging

**Framework:** `log` (Go standard library)

**Patterns:**
- Used only for important server lifecycle events
- `log.Println()` for informational messages (server starting)
- `log.Fatal()` for fatal errors (causes immediate exit)
- No structured logging, simple line output

**Example:**
```go
log.Println("starting server on :8080")
if err := http.ListenAndServe(":8080", mux); err != nil {
	log.Fatal(err)
}
```

## Comments

**When to Comment:**
- Currently no comments in code (code is self-explanatory)
- Comments would explain "why", not "what" (Go convention)
- Not currently used; codebase is simple enough to be self-documenting

**Documentation:**
- README.md provides high-level documentation
- HTTP handlers documented via README (GET /health endpoint)

## Function Design

**Size:**
- Keep functions small and focused
- `Handler` function: 5 lines of logic
- `main()` function: 6 lines (setup + start)

**Parameters:**
- Functions accept only required parameters
- Handlers use standard http.Handler signature: `(w http.ResponseWriter, r *http.Request)`
- No variadic or optional parameters currently used

**Return Values:**
- Functions returning errors use standard `error` type
- HTTP handlers follow std library: no explicit return, write to ResponseWriter

**Example:**
```go
func Handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}
```

## Module Design

**Exports:**
- Functions exported (capitalized) when needed by other packages
- `Handler` exported from `health` package for use in `cmd/server`
- Keep surface area minimal

**Package Organization:**
- `cmd/server/`: Entry point
- `internal/health/`: Health check functionality
- `pkg/`: Empty (for future public packages)

**Barrel Files:**
- Not used (not needed for small codebase)
- Would be `package.go` file exporting selected types if needed

## Standards

**Go Version:**
- Target: Go 1.23
- Modern Go idioms apply: error wrapping, standard library http multiplexer

**Standard Library Usage:**
- Prefer std library over external packages (exemplified by using `net/http` directly)
- No external dependencies currently (go.mod has no requires)

---

*Convention analysis: 2026-02-10*
