## Database Layer Implementation

### Key Decisions
- Migrations live in `internal/db/migrations/` (not project root) because `go:embed` does not support parent directory paths (`..`). The `//go:embed migrations/*.sql` directive in `migrate.go` works relative to the package directory.
- Used `golang-migrate/migrate/v4` with `iofs.New` for embedded migration files — no external migration tool needed at runtime.
- Pool config: MaxConns=20, MinConns=5, MaxConnIdleTime=10m, HealthCheckPeriod=30s via `pgxpool.ParseConfig` + struct config.

### TDD Notes
- Test uses `t.Skipf()` when PostgreSQL is unavailable (no Docker daemon), making it safe for CI environments without PG.
- Test covers full up/down cycle + idempotent re-run + user_id column existence on dimension tables + schema_migrations table verification.

## EventBus + SSE Implementation

### Key Decisions
- EventBus uses `sync.RWMutex` — RLock for Publish (read-heavy path), full Lock for Subscribe/Unsubscribe (mutating path). Bounded channel capacity of 64 prevents unbounded memory growth.
- Publish uses `select { case ch <- evt: default: }` for non-blocking send — silently drops if subscriber channel is full, matching the "at-least-once-but-not-guaranteed" SSE semantics.
- Resubscribe (same runID) closes old channel before creating new one, preventing stale subscribers from leaking goroutines.
- SSE handler calls `http.NewResponseController(w).Flush()` immediately after setting headers (no error check — Go 1.25 removes the second return value). This signals HTTP 200 to the client before any events arrive.
- Heartbeat runs as a per-connection goroutine using `time.NewTicker(30s)`, stopped via `defer heartbeat.Stop()`. The `r.Context().Done()` case handles client disconnect.
- SSE format: `event: {type}\ndata: {json}\n\n` where data is `json.RawMessage` marshaled via `json.Marshal` (handles nil → "null" correctly).

### Go 1.25 Notes
- `http.NewResponseController(w)` returns only `*http.ResponseController` (no error) in Go 1.25. This differs from Go 1.20-1.24 which returned `(*http.ResponseController, error)`.
- The project uses `go 1.25.0` — make sure to check Go version when referencing stdlib APIs that changed across versions.

### Test Notes
- SSE handler tests use `httptest.NewServer` (real TCP server) because `httptest.NewRecorder` doesn't support streaming responses.
- Heartbeat test takes ~30s (one tick cycle). Run with `-timeout 120s` to avoid test suite timeout.
- Client disconnect test: close the response body (simulates TCP close), then cancel the request context, then verify the handler exited by checking re-subscription behavior.

## LoginPage + useAuth Hook

### Key Decisions
- `useAuth` wraps existing `api/client.ts` token utilities (`getToken`, `setToken`, `clearToken`) — single source of truth for localStorage key `'jwt'`.
- JWT expiry check uses standard `atob` + base64url decode (handles `-` → `+`, `_` → `/` substitution). On expired token, clears it from localStorage before returning `false`.
- Auto-redirect guard uses `useEffect` with `location.pathname !== '/login'` check to prevent redirect loops on the login page itself.
- LoginPage OAuth callback: `useSearchParams` to read `?token=`, then `setToken()` + `navigate('/chat', { replace: true })` — replace prevents back-navigation to the callback URL.
- `login()` uses `window.location.href` (not `navigate()`) because WeChat OAuth requires a full page redirect to the external auth endpoint.

## React Skeleton (Task 10)

### Key Decisions
- Used `npx create-vite@latest web --template react-ts` for scaffolding (Vite 8, React 19, TypeScript 6).
- Tailwind CSS v4 uses `@import "tailwindcss"` in index.css (no `tailwind.config.js` needed). The `@tailwindcss/vite` plugin handles CSS processing.
- Dev proxy configured as string `'/api': 'http://localhost:8080'` (not object with `target`/`changeOrigin`) — the simple form works for standard proxying.
- Layout uses `NavLink` with `isActive` callback for active state styling (declarative, no useState needed).
- API client stores JWT in localStorage under key `'jwt'` — same key used across `api/`, `hooks/`, and `components/`.

### Issues Encountered
- `create-vite` scaffold was non-interactive when directory already existed — had to `rm -rf web/` first.
- `npm install` timed out due to network; re-ran with longer timeout.
- `picomatch` missing error after install: `rm -rf node_modules package-lock.json && npm install` fixed it.
- Some source files (LoginPage, HistoryPage, SettingsPage, client.ts, useAuth.ts) were already tracked in git from prior work — left them unchanged.
- ChatPage.tsx was overwritten with full implementation (136 lines importing non-existent components) by unknown process — reverted to 8-line placeholder.

## Docker Compose & Makefile Finalization

### Docker Compose
- Multi-stage Dockerfile already existed (builder + alpine:3.20 runtime).
- `docker compose config` fails when `env_file` references a missing `.env` — validate with a temporary `.env` copy.
- Added `restart: unless-stopped` to both services for production resilience.
- Added backend healthcheck via `wget -qO- http://localhost:8080/health`.
- Pass `JWT_SECRET`, `LLM_BASE_URL`, `LLM_MODEL` via environment for full Docker-based configuration.

### Makefile
- `dev` target now starts docker compose THEN runs `npm run dev` for frontend — full-stack in one command.
- Split `test`/`build`/`lint` into `*-backend` and `*-frontend` subtargets for granular control.
- `build-frontend` runs `npm ci` before `npm run build` for clean installs in CI.
- `clean` target removes both `bin/` and `web/dist/`.
- `lint-backend` requires `golangci-lint` binary — target exists but may fail if not installed.

### Config
- Added `[auth]` section with `jwt_secret` to `config.example.toml` — required by config validation.
