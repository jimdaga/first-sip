# first-sip

A Go service foundation with CI/CD, Helm packaging, and ArgoCD deployment.

## Prerequisites

- Go 1.23+
- Docker
- Helm 3
- golangci-lint

## Getting Started

```bash
# Run the server locally
make run

# Run tests
make test

# Run linter
make lint
```

## Make Targets

| Target         | Description                        |
| -------------- | ---------------------------------- |
| `build`        | Build the Go binary                |
| `test`         | Run tests with race detection      |
| `lint`         | Run golangci-lint                  |
| `run`          | Run the server locally             |
| `docker-build` | Build the Docker image             |
| `clean`        | Remove build artifacts             |

## Docker

```bash
make docker-build
docker run -p 8080:8080 jimdaga/first-sip:local
curl localhost:8080/health
```

## Health Check

```
GET /health
```

Returns:
```json
{"status":"ok"}
```
