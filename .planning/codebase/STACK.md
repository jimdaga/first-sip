# Technology Stack

**Analysis Date:** 2026-02-10

## Languages

**Primary:**
- Go 1.23 - Full application and service implementation

## Runtime

**Environment:**
- Go 1.23 standard runtime
- Alpine Linux 3.19 for container runtime

**Package Manager:**
- Go modules (go.mod)
- Lockfile: Not present (go.sum not committed)

## Frameworks

**Core:**
- Go standard library `net/http` - HTTP server and routing

**Build/Dev:**
- Make - Build automation (`Makefile` at root)
- Docker - Container building (multi-stage Dockerfile with golang:1.23-alpine builder)
- golangci-lint - Linting (`golangci-lint run ./...`)

## Key Dependencies

**Testing:**
- Go standard library `testing` package - Unit testing
- Go standard library `net/http/httptest` - HTTP test utilities

**No external third-party dependencies detected** - Application uses only Go standard library

## Configuration

**Environment:**
- Kubernetes Helm charts for configuration management
- Environment variables passed via `app.env` in Helm values
- Deployment configuration: `charts/first-sip/values.yaml`
- Override values: `infra/app/values-prd.yaml`, `infra/app/values-dev.yaml`

**Build:**
- Multi-stage Docker build: `Dockerfile` at root
- Linting configuration: `.golangci.yml` with enabled linters: errcheck, govet, staticcheck, unused, gosimple, ineffassign, typecheck
- Timeout: 5m for linting

## Platform Requirements

**Development:**
- Go 1.23
- Make
- Docker
- golangci-lint

**Production:**
- Kubernetes cluster (1.23+)
- Docker registry access (jimdaga Docker Hub account)
- ArgoCD for deployment automation
- Helm 3.x

## Ports & Networking

**Application:**
- HTTP port: 8080 (container)
- Service port: 80 (Kubernetes ClusterIP)
- Health endpoint: `GET /health` (JSON response: `{"status":"ok"}`)

---

*Stack analysis: 2026-02-10*
