# Codebase Structure

**Analysis Date:** 2026-02-10

## Directory Layout

```
first-sip/
├── cmd/                        # Executable entry points
│   └── server/
│       └── main.go            # HTTP server bootstrap
├── internal/                   # Private application code
│   └── health/
│       ├── handler.go         # Health check HTTP handler
│       └── handler_test.go    # Handler unit tests
├── pkg/                        # Public/shared packages (reserved for future)
│   └── .gitkeep
├── charts/                     # Helm chart for Kubernetes deployment
│   └── first-sip/
│       ├── Chart.yaml         # Helm chart metadata
│       ├── values.yaml        # Default values
│       └── templates/         # Kubernetes resource templates
├── infra/                      # Infrastructure and deployment configs
│   ├── app/                   # Environment-specific Helm values
│   │   ├── values-dev.yaml    # Development overrides
│   │   └── values-prd.yaml    # Production overrides
│   └── argocd/                # GitOps deployment manifests
│       ├── applications/      # ArgoCD Application resources
│       └── repo-secret.yaml   # Git repository credentials
├── Dockerfile                  # Multi-stage container build
├── Makefile                    # Development tasks
├── go.mod                      # Go module definition
├── README.md                   # Project documentation
└── LICENSE                     # License file
```

## Directory Purposes

**`cmd/` - Executable Entry Points:**
- Purpose: Contains main packages that compile to binary executables
- Contains: Server initialization code
- Key files: `cmd/server/main.go` - Only entry point; starts HTTP server on :8080

**`internal/` - Private Application Code:**
- Purpose: Packages not intended for external use; enforced by Go's internal package mechanism
- Contains: Domain logic, handlers, services
- Key files: `internal/health/handler.go` - HTTP handler for `/health` endpoint

**`pkg/` - Public/Shared Packages:**
- Purpose: Reusable code that may be imported by external packages
- Contains: Currently empty (reserved for utilities, domain models, helpers)
- Key files: None yet; create subdirectories for future shared code

**`charts/` - Kubernetes Deployment:**
- Purpose: Helm chart for packaging application for Kubernetes
- Contains: Chart metadata, default values, resource templates
- Key files:
  - `Chart.yaml`: Chart version and metadata
  - `values.yaml`: Default Helm values (replicas, image, probes, etc.)
  - `templates/`: Kubernetes manifests (Deployment, Service, etc.)

**`infra/` - Infrastructure Configuration:**
- Purpose: Environment-specific deployment and GitOps configuration
- Contains: Environment overrides, ArgoCD manifests
- Subdirectories:
  - `app/`: Helm value overrides per environment
  - `argocd/`: ArgoCD Application resources for GitOps-driven deployments

**`infra/app/` - Environment-Specific Values:**
- `values-dev.yaml`: Dev overrides (Always image pull, ingress disabled)
- `values-prd.yaml`: Production overrides (IfNotPresent image pull, ingress disabled)

**`infra/argocd/applications/` - GitOps Application Definitions:**
- `firstsip-dev.yaml`: Dev environment ArgoCD Application (tracks pre-release versions)
- `firstsip-prd.yaml`: Production ArgoCD Application (pinned to specific release)

## Key File Locations

**Entry Points:**
- `cmd/server/main.go`: Application startup point; creates mux, registers handlers, starts server

**Configuration:**
- `go.mod`: Module name, Go version, dependencies
- `Dockerfile`: Multi-stage build (builder stage uses golang:1.23-alpine, final uses alpine:3.19)
- `Makefile`: Build, test, lint, run, docker-build, clean targets

**Core Logic:**
- `internal/health/handler.go`: `/health` endpoint implementation

**Testing:**
- `internal/health/handler_test.go`: Unit test using httptest.NewRecorder()

**Deployment:**
- `charts/first-sip/Chart.yaml`: Helm chart definition (v0.0.0)
- `charts/first-sip/values.yaml`: Default Helm values (port 80 service, liveness/readiness probes, 1 replica)
- `infra/argocd/applications/firstsip-dev.yaml`: Dev deployment (targets latest pre-release)
- `infra/argocd/applications/firstsip-prd.yaml`: Production deployment (pinned version, auto-updates from releases)

## Naming Conventions

**Files:**
- Lowercase with underscores for multi-word files: `handler_test.go`, `values-dev.yaml`
- Package files match package name: `handler.go` in `health/` package
- Test files follow `_test.go` suffix convention: `handler_test.go`

**Directories:**
- Lowercase, single word preferred: `cmd`, `internal`, `pkg`, `charts`, `infra`
- Multi-word: hyphenated in some cases: `first-sip` (package name), environment-specific: `values-dev.yaml`

**Go Identifiers:**
- Function names: PascalCase (exported): `Handler`
- Variable names: camelCase (unexported in functions)
- Package names: lowercase: `health`, `main`

**Kubernetes/Helm:**
- Resource names: lowercase with hyphens: `first-sip`, `first-sip-dev`, `first-sip-prd`
- Template file names: lowercase matching resource type: `deployment.yaml`, `service.yaml`

## Where to Add New Code

**New HTTP Endpoint:**
1. Create new handler file in `internal/[domain]/handler.go`
2. Implement `http.HandlerFunc` signature function
3. Register in `cmd/server/main.go` mux: `mux.HandleFunc("METHOD /path", handler.Handler)`
4. Create test in `internal/[domain]/handler_test.go` using `httptest` package

**New Shared Utility/Package:**
1. Create new directory in `pkg/[functionality]/`
2. Add `.go` files in that directory
3. Initialize public functions (PascalCase) for export
4. Import in `internal/` packages as needed: `"github.com/jimdaga/first-sip/pkg/[functionality]"`

**New Kubernetes Configuration:**
1. For new Helm template: Add `.yaml` file in `charts/first-sip/templates/`
2. For environment-specific overrides: Update `infra/app/values-[env].yaml`
3. Reference in ArgoCD Application under `helm.valueFiles`

**New Service/Domain:**
1. Create `internal/[service]/` directory
2. Add handler file and test file
3. Can add `internal/[service]/models.go` for domain types
4. Import and register in `cmd/server/main.go`

## Special Directories

**`.planning/` - Planning and Documentation:**
- Purpose: GSD analysis documents (ARCHITECTURE.md, STRUCTURE.md, etc.)
- Generated: Yes, by GSD tooling
- Committed: Yes, versioned with codebase

**`charts/first-sip/templates/` - Kubernetes Resource Templates:**
- Purpose: Helm template files that generate Kubernetes manifests
- Files generated: Deployment, Service, ServiceAccount, Ingress, HPA, tests
- Variables sourced from: `values.yaml` and environment-specific overrides

**`.git/` - Version Control:**
- Purpose: Git repository metadata and history
- Generated: Yes, by git
- Committed: Yes (automatically)

---

*Structure analysis: 2026-02-10*
