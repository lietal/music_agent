# Music Agent

AI-powered music search and recommendation agent with a Go backend, React frontend, and PostgreSQL.

## Architecture

```
┌──────────────────────────────────────────────────┐
│                   Client (Browser)               │
│                React + TypeScript                │
│               Vite dev server :5173              │
└──────────────────┬───────────────────────────────┘
                   │ HTTP / SSE
┌──────────────────▼───────────────────────────────┐
│                 Go Backend :8080                 │
│  ┌──────────┬──────────┬──────────┬───────────┐ │
│  │  Auth    │  Chat    │  Conv    │  Health   │ │
│  │ (OAuth)  │ (SSE)    │ (CRUD)   │           │ │
│  └──────────┴──────────┴──────────┴───────────┘ │
│           chi router + JWT middleware            │
└──────────────────┬───────────────────────────────┘
                   │ SQL
┌──────────────────▼───────────────────────────────┐
│              PostgreSQL :5432                    │
│       Conversations, user data, analytics        │
└──────────────────────────────────────────────────┘
```

## Quick Start

```bash
# 1. Copy and edit the configuration
cp config.example.toml config.toml
# Edit config.toml with your LLM API key, WeChat credentials, etc.

# 2. Create .env for Docker secrets (optional)
cp .env.example .env

# 3. Start the full stack
make dev
```

This starts PostgreSQL, the Go backend on `http://localhost:8080`, and the React frontend on `http://localhost:5173`.

## API Endpoints

| Method | Path                          | Auth | Description                  |
|--------|-------------------------------|------|------------------------------|
| GET    | `/health`                     | No   | Health check                 |
| GET    | `/api/auth/{provider}`        | No   | OAuth provider redirect      |
| GET    | `/api/auth/callback/{provider}`| No  | OAuth callback, returns JWT  |
| GET    | `/api/auth/me`                | JWT  | Current authenticated user   |
| POST   | `/api/conversations`          | JWT  | Create conversation          |
| GET    | `/api/conversations`          | JWT  | List conversations           |
| GET    | `/api/conversations/{id}`     | JWT  | Get conversation by ID       |
| POST   | `/api/chat`                   | JWT  | Start a chat run             |
| GET    | `/api/chat/{runId}/events`    | JWT  | SSE stream for chat events   |

### Example: Start a chat session

```bash
# Create a chat run
curl -X POST http://localhost:8080/api/chat \
  -H "Authorization: Bearer <jwt-token>" \
  -H "Content-Type: application/json"

# Stream events via SSE
curl -N http://localhost:8080/api/chat/<runId>/events \
  -H "Authorization: Bearer <jwt-token>"
```

## Configuration

Copy `config.example.toml` to `config.toml` and adjust values:

| Section          | Key           | Description                       |
|------------------|---------------|-----------------------------------|
| `[server]`       | `host`        | Listen address (default: 0.0.0.0) |
| `[server]`       | `port`        | Listen port (default: 8080)       |
| `[database]`     | `url`         | PostgreSQL connection string      |
| `[auth.wechat]`  | `app_id`      | WeChat Mini Program App ID        |
| `[auth.wechat]`  | `app_secret`  | WeChat Mini Program App Secret    |
| `[llm]`          | `provider`    | LLM provider (openai, etc.)       |
| `[llm]`          | `api_key`     | LLM API key                       |
| `[llm]`          | `model`       | Model name (e.g. gpt-4o-mini)     |

Environment variables (used by Docker Compose, `.env` file):

| Variable          | Description                     |
|-------------------|---------------------------------|
| `DB_PASSWORD`     | PostgreSQL password             |
| `LLM_API_KEY`     | LLM provider API key            |
| `LLM_BASE_URL`    | LLM API base URL                |
| `LLM_MODEL`       | LLM model name                  |
| `WECHAT_APP_ID`   | WeChat OAuth App ID             |
| `WECHAT_APP_SECRET`| WeChat OAuth App Secret        |
| `JWT_SECRET`      | JWT signing secret              |

## Makefile Targets

| Target           | Description                                    |
|------------------|------------------------------------------------|
| `make dev`       | Start PostgreSQL + backend + frontend dev      |
| `make test`      | Run all tests                                  |
| `make build`     | Build Go binary + React production bundle      |
| `make lint`      | Run golangci-lint                              |
| `make clean`     | Tear down Docker services, remove build output |

## Tech Stack

- **Backend**: Go 1.25, chi router, JWT auth, SSE streaming
- **Frontend**: React 19, TypeScript, Vite, Tailwind CSS v4
- **Database**: PostgreSQL 16
- **Infrastructure**: Docker Compose, multi-stage Docker builds
