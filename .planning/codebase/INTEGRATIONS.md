# External Integrations

**Analysis Date:** 2026-02-10

## APIs & External Services

**Not detected** - Application does not integrate with external APIs or services at this time

## Data Storage

**Databases:**
- Not detected - No persistent data storage configured

**File Storage:**
- Not detected - Application does not use file storage

**Caching:**
- Not detected - No caching layers configured

## Authentication & Identity

**Auth Provider:**
- Not applicable - Application does not implement authentication

**Security:**
- No auth mechanism currently implemented
- Health endpoint is unauthenticated public endpoint

## Monitoring & Observability

**Error Tracking:**
- Not detected - No error tracking service integrated

**Logs:**
- Standard output via `log` package - Logs to stdout for container consumption
- Kubernetes collects container logs for persistence

## CI/CD & Deployment

**Hosting:**
- Kubernetes cluster (configured via ArgoCD)
- Docker Hub registry: `jimdaga/first-sip` image repository

**CI Pipeline:**
- GitHub Actions workflows: `.github/workflows/publish-artifacts.yaml`, `.github/workflows/publish-artifacts-latest.yaml`
- Triggers: Release published events
- Process:
  1. Checkout code (v4)
  2. Setup Go 1.23 (actions/setup-go@v5)
  3. Run tests: `go test -v -race -coverprofile=coverage.out ./...`
  4. Docker login (via secrets.DOCKER_USERNAME, secrets.DOCKER_PASSWORD)
  5. Build and push Docker image with semver tagging
  6. Release Helm chart via chart-releaser-action@v1.5.0
  7. Update ArgoCD production application targetRevision

**ArgoCD Integration:**
- Repository: https://jimdaga.github.io/first-sip (Helm chart)
- Git source: git@github.com:jimdaga/first-sip.git
- Applications:
  - `infra/argocd/applications/firstsip-dev.yaml` - Development environment
  - `infra/argocd/applications/firstsip-prd.yaml` - Production environment (auto-updated)
- Sync policy: Automated with auto-prune and self-heal enabled
- Target namespaces: `firstsip-dev`, `firstsip-prd`

## Environment Configuration

**Required env vars:**
- Not currently implemented - Application has no environment-specific configuration requirements

**Deployment configuration location:**
- Helm values overrides: `infra/app/values-prd.yaml`, `infra/app/values-dev.yaml`
- Container environment variables: Configured via `app.env` in Helm values (currently empty)

**Secrets location:**
- GitHub Actions secrets for Docker Hub: `DOCKER_USERNAME`, `DOCKER_PASSWORD`
- Git SSH key for ArgoCD: Used via `git@github.com` clone URLs

## Health Checks

**Kubernetes Probes:**
- Liveness probe: `GET /health`, initial delay 5s, period 10s
- Readiness probe: `GET /health`, initial delay 3s, period 5s
- Both probes expect HTTP 200 status code

## Webhooks & Callbacks

**Incoming:**
- Not detected - Application does not expose webhook endpoints

**Outgoing:**
- Not detected - Application does not trigger webhooks

---

*Integration audit: 2026-02-10*
