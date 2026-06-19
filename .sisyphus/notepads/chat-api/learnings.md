# Chat API Implementation - Learnings

## Architecture Decisions
- Handler struct holds `*event.Bus`, `jwtSecret []byte`, and `auth.OAuthProvider` for dependency injection
- EventBus (`internal/event/bus.go`) was already implemented with single-subscriber-per-runID model
- JWT middleware from `internal/auth/middleware.go` supports both `Authorization: Bearer` header and `?token=` query param - reused directly via `JWTAuthMiddleware` wrapper
- Chi router groups protect routes that need JWT auth

## Key Flow (Event Loss Prevention)
1. POST /api/chat → validate JWT → create runId (UUID) → return {run_id} immediately
2. GET /api/chat/{runId}/events → validate JWT via ?token= → set SSE headers → subscribe to EventBus → THEN start agent goroutine
3. Agent publishes events to EventBus → SSE handler reads from subscription channel → writes SSE data
4. On done/error event, SSE handler returns and cleanup runs (cancel context, unsubscribe)

## Config Changes
- Added `JWTSecret` field to `AuthConfig` in `internal/config/config.go`
- Default: `"dev-secret-change-in-production"` (via SetDefault and env fallback)
- Env binding: `JWT_SECRET`
- Validation requires jwt_secret to be present

## Files Created
- internal/api/handler.go
- internal/api/middleware.go
- internal/api/health.go
- internal/api/auth.go
- internal/api/conversation.go
- internal/api/chat.go
- internal/api/router.go
- internal/api/chat_test.go

## Files Modified
- internal/config/config.go (JWTSecret)
- internal/config/config_test.go (TestValidateAllPresent fix)
- cmd/server/main.go (wired up router)
