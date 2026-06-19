# Music Agent — Project Guidelines

## ⛔ Coverage Requirement (HARD GATE)

**No code change is accepted unless both backend and frontend unit test coverage exceed 80%.**

This is non-negotiable. Any PR, commit, or refactor that drops coverage below 80% MUST be accompanied by additional tests.

### Backend (Go)

```bash
cd server && go test ./internal/... -coverprofile=coverage.out -covermode=atomic -count=1 && go tool cover -func=coverage.out | grep total:
```

Target: `total: (statements) ≥ 80.0%`

### Frontend (TypeScript/React)

```bash
cd web && npx vitest run --coverage
```

Target: `All files ≥ 80% Stmts`

### Pre-Commit Checklist

- [ ] `make test-backend` produces `total: ≥ 80.0%`
- [ ] `cd web && npx vitest run --coverage` produces `All files ≥ 80%`
- [ ] No skipped tests without documented reason
- [ ] No `.only` calls in test files

### MSW (Mock Service Worker)

Frontend tests use MSW to intercept HTTP requests at the network level. Handlers are defined in `web/src/test-handlers.ts`. The MSW server is set up in `web/src/test-setup.ts` with `beforeAll/afterEach/afterAll`.

Page/component tests that make API calls do NOT need `vi.mock` or `vi.stubGlobal` — MSW handles all `fetch` requests automatically.

### Per-Package Benchmarks (current)

| Package | Coverage | Key Constraint |
|---------|----------|----------------|
| `event` | 100% | Pure pub/sub, trivially testable |
| `uuid` | 100% | Wrapper, trivially testable |
| `tool` | 89% | Mock tools + TME search wrapper |
| `api` | 85% | SSE handlers refactored with `writeSSEEvents()` + `AgentRunner` interface |
| `config` | 82% | Viper TOML loader |
| `auth` | 81% | JWT + bcrypt, mem fallback tested independently |
| `llm` | 81% | OpenAI-compatible HTTP client |
| `tme` | 80% | TME/QQ Music API client, mock servers for all endpoints |
| `agent` | 74% | AgentLoop + LLM Planner + Executor |
| `db` | 63% | PostgreSQL pool (needs real DB) |

| Frontend Module | Coverage |
|-----------------|----------|
| `hooks/useSSE.ts` | 97% |
| `hooks/useAuth.ts` | 97% |
| `api/client.ts` | 95% |
| `pages/LoginPage.tsx` | 91% |
| `pages/SettingsPage.tsx` | 75% |
| `components/AgentMessageList.tsx` | 71% |
| `components/Layout.tsx` | 67% |
| `pages/HistoryPage.tsx` | 63% |
| `pages/ChatPage.tsx` | 33% |
| `components/TracePanel.tsx` | 57% |
| `components/SongCards.tsx` | 100% |
| `App.tsx` | 100% |
| `auth` | 81% | JWT + bcrypt, mem fallback tested independently |
| `llm` | 81% | OpenAI-compatible HTTP client |
| `tme` | 80% | TME/QQ Music API client, mock servers for all endpoints |
| `agent` | 74% | AgentLoop + LLM Planner + Executor |
| `db` | 63% | PostgreSQL pool (needs real DB) |

### Test Patterns

- **`tme` package**: Every API method tested with `httptest.NewServer` mocking `musicu.fcg`
- **`api` package**: `AgentRunner` interface enables mock agent injection; `writeSSEEvents()` is a pure function tested independently
- **`agent` package**: `mockPlanner` + `mockLLM` for testing AgentLoop and Planner
- **`auth` package**: `memRegister`/`memLogin` in-memory fallback tests run without DB; full DB tests require running Postgres

## Project Structure

```
music_searchrecom_agent/
├── server/                     # Go backend
│   ├── cmd/server/main.go
│   ├── internal/
│   │   ├── agent/              # AgentLoop + Planner + Executor
│   │   ├── api/                # HTTP handlers + SSE + router
│   │   ├── auth/               # JWT + bcrypt
│   │   ├── config/             # viper TOML config
│   │   ├── db/                 # PostgreSQL pool + migrations
│   │   ├── event/              # Event bus
│   │   ├── llm/                # OpenAI-compatible client
│   │   ├── tme/                # QQ Music native API client
│   │   ├── tool/               # Tool interface + mocks + TME search
│   │   └── uuid/
│   └── tests/
├── web/                        # React frontend
├── config.toml
├── docker-compose.yml
└── Makefile
```

## Key Design Decisions

1. **AgentRunner interface** (`api/handler.go`): `Handler.agent` is `AgentRunner` interface, not concrete `*AgentLoop`. Enables mock injection in tests.

2. **SSE writer extraction** (`api/chat.go`): `writeSSEEvents(w, rc, ctx, ch)` is a pure function — takes a channel of events and writes SSE format. Both `chatEventsHandler` and `fallbackMockSSE` use it. Testable independently.

3. **TME native Go client** (`tme/`): Direct HTTP POST to `u.y.qq.com/cgi-bin/musicu.fcg`. No Python sidecar. No signing. Minimal `comm` params (`ct:11`).

4. **LLM Planner** (`agent/planner_llm.go`): LLM decides tool calls. `AnswerGenerator` interface generates streaming final responses. AgentLoop type-asserts planner for this optional capability.

## Running Tests

```bash
# All backend tests
make test-backend

# Coverage report
cd server && go test ./internal/... -coverprofile=coverage.out -covermode=atomic -count=1 && go tool cover -func=coverage.out

# Single package
cd server && go test ./internal/tme/ -v -count=1
```

Note: `TestRunMigrations` is destructive (runs down-migration). Run it first, then run other DB-dependent tests: `cd server && go test ./internal/db/ -run TestRunMigrations && go test ./internal/auth/ -v`
