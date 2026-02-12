# First Sip

A daily briefing app that aggregates news, weather, and work updates into a single morning summary. Built with Go, HTMX, and background job processing.

## Prerequisites

- Go 1.24+
- Docker & Docker Compose
- [templ](https://templ.guide/) — `go install github.com/a-h/templ/cmd/templ@latest`
- [golangci-lint](https://golangci-lint.run/)

## Quick Start

```bash
# 1. Start infrastructure (Postgres, Redis, Asynqmon)
make db-up

# 2. Set up environment
cp env.local env.local   # already exists with defaults
source env.local

# 3. Configure Google OAuth (see below)

# 4. Run the app
make dev
```

Open http://localhost:8080 in your browser.

## Google OAuth Setup

The app uses Google OAuth for authentication. You need to create credentials in the Google Cloud Console.

### 1. Create a Google Cloud Project

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project (or select an existing one)

### 2. Configure the OAuth Consent Screen

1. Navigate to **APIs & Services > OAuth consent screen**
2. Select **External** user type (or Internal if using Google Workspace)
3. Fill in the required fields:
   - **App name**: First Sip (or whatever you like)
   - **User support email**: your email
   - **Developer contact**: your email
4. Under **Scopes**, add: `email`, `profile`, `openid`
5. Under **Test users**, add your Google email address
6. Save

### 3. Create OAuth Credentials

1. Navigate to **APIs & Services > Credentials**
2. Click **Create Credentials > OAuth client ID**
3. Select **Web application**
4. Set the **Authorized redirect URI**:
   ```
   http://localhost:8080/auth/google/callback
   ```
5. Click **Create** and copy the **Client ID** and **Client Secret**

### 4. Configure Environment

Edit `env.local` and fill in your credentials:

```bash
export GOOGLE_CLIENT_ID="your-client-id-here.apps.googleusercontent.com"
export GOOGLE_CLIENT_SECRET="your-client-secret-here"
export GOOGLE_CALLBACK_URL="http://localhost:8080/auth/google/callback"
export SESSION_SECRET="$(openssl rand -hex 32)"
```

Then reload: `source env.local`

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `GOOGLE_CLIENT_ID` | Yes | — | Google OAuth client ID |
| `GOOGLE_CLIENT_SECRET` | Yes | — | Google OAuth client secret |
| `GOOGLE_CALLBACK_URL` | Yes | — | OAuth redirect URI |
| `SESSION_SECRET` | Yes | — | Cookie encryption key (generate with `openssl rand -hex 32`) |
| `DATABASE_URL` | Yes | — | Postgres connection string |
| `ENCRYPTION_KEY` | Yes | — | AES-256-GCM key for token encryption (`openssl rand -base64 32`) |
| `REDIS_URL` | Yes | — | Redis connection URL |
| `N8N_STUB_MODE` | No | `true` | Use mock briefing data instead of calling n8n |
| `N8N_WEBHOOK_URL` | No | — | n8n webhook endpoint (only when stub mode is off) |
| `N8N_WEBHOOK_SECRET` | No | — | n8n webhook auth secret (only when stub mode is off) |
| `ENV` | No | `development` | Environment (`development` or `production`) |
| `PORT` | No | `8080` | HTTP server port |
| `LOG_LEVEL` | No | `debug` | Log level (`debug`, `info`, `warn`, `error`) |
| `LOG_FORMAT` | No | `text` | Log format (`text` or `json`, forced `json` in production) |

## Infrastructure

Docker Compose provides Postgres, Redis, and Asynqmon:

```bash
make db-up      # Start services
make db-down    # Stop services
make db-reset   # Wipe data and restart
```

| Service | Port | URL |
|---------|------|-----|
| App | 8080 | http://localhost:8080 |
| Postgres | 5432 | `postgres://first_sip:local_dev_password@localhost:5432/first_sip` |
| Redis | 6379 | `redis://localhost:6379` |
| Asynqmon | 8081 | http://localhost:8081 |

## Make Targets

| Target | Description |
|--------|-------------|
| `dev` | Run the server with embedded worker (development) |
| `worker` | Run standalone worker process |
| `build` | Build the Go binary |
| `test` | Run tests with race detection |
| `lint` | Run golangci-lint |
| `templ-generate` | Regenerate Go code from .templ files |
| `docker-build` | Build the Docker image |
| `db-up` | Start Docker Compose services |
| `db-down` | Stop Docker Compose services |
| `db-reset` | Wipe volumes and restart services |
| `clean` | Remove build artifacts |

## Health Check

```
GET /health → {"status":"ok"}
```
