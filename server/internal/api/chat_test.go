package api

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/music-agent/music-agent/internal/auth"
	"github.com/music-agent/music-agent/internal/event"
)

func newTestHandler() *Handler {
	bus := event.NewBus()
	jwtSecret := []byte("test-secret")
	return NewHandler(bus, jwtSecret, nil)
}

func generateTestToken(secret []byte) string {
	claims := jwt.MapClaims{
		"user_id":  "test-user",
		"provider": "wechat",
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := tok.SignedString(secret)
	return tokenString
}

func TestCreateChat_ValidJWT(t *testing.T) {
	h := newTestHandler()
	router := NewRouter(h)

	req := httptest.NewRequest("POST", "/api/chat", strings.NewReader(`{"message":"test"}`))
	req.Header.Set("Authorization", "Bearer "+generateTestToken(h.jwtSecret))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp["runId"] == "" {
		t.Error("expected runId in response")
	}
}

func TestCreateChat_UnauthorizedNoToken(t *testing.T) {
	h := newTestHandler()
	router := NewRouter(h)

	req := httptest.NewRequest("POST", "/api/chat", strings.NewReader(`{"message":"test"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestCreateChat_InvalidToken(t *testing.T) {
	h := newTestHandler()
	router := NewRouter(h)

	req := httptest.NewRequest("POST", "/api/chat", strings.NewReader(`{"message":"test"}`))
	req.Header.Set("Authorization", "Bearer invalid-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestCreateChat_PreferredAuthHeaderOverQuery(t *testing.T) {
	h := newTestHandler()
	router := NewRouter(h)

	tok := generateTestToken(h.jwtSecret)

	req := httptest.NewRequest("POST", "/api/chat?token=bad-token", strings.NewReader(`{"message":"test"}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201 with valid header token, got %d", rec.Code)
	}
}

func TestChatEvents_SSEStream(t *testing.T) {
	h := newTestHandler()
	router := NewRouter(h)

	runID := "test-run-id"

	tok := generateTestToken(h.jwtSecret)
	req := httptest.NewRequest("GET", "/api/chat/"+runID+"/events?token="+tok, nil)
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		router.ServeHTTP(rec, req)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("SSE handler timed out")
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	if !strings.Contains(body, "event: ") {
		t.Error("expected SSE event: prefix in response")
	}
	if !strings.Contains(body, "data:") {
		t.Error("expected SSE data: prefix in response")
	}

	scanner := bufio.NewScanner(strings.NewReader(body))
	var currentType string
	var types []string
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "event: ") {
			currentType = strings.TrimPrefix(line, "event: ")
		}
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data != "" {
				var raw json.RawMessage
				if err := json.Unmarshal([]byte(data), &raw); err != nil {
					t.Logf("failed to parse event data: %s", data)
				}
			}
			types = append(types, currentType)
		}
	}

	if len(types) == 0 {
		t.Fatal("expected at least one SSE event")
	}

	hasDelta := false
	hasDone := false
	for _, typ := range types {
		if typ == event.TypeDelta {
			hasDelta = true
		}
		if typ == event.TypeDone {
			hasDone = true
		}
	}

	if !hasDelta {
		t.Error("expected delta event in SSE stream")
	}
	if !hasDone {
		t.Error("expected done event in SSE stream")
	}
}

func TestChatEvents_UnauthorizedNoToken(t *testing.T) {
	h := newTestHandler()
	router := NewRouter(h)

	req := httptest.NewRequest("GET", "/api/chat/test-run/events", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestChatEvents_UnauthorizedInvalidToken(t *testing.T) {
	h := newTestHandler()
	router := NewRouter(h)

	req := httptest.NewRequest("GET", "/api/chat/test-run/events?token=bad-token", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestHealthEndpoint(t *testing.T) {
	h := newTestHandler()
	router := NewRouter(h)

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp["status"] != "ok" {
		t.Errorf("expected status ok, got %q", resp["status"])
	}
}

func TestAuthMe_Unauthorized(t *testing.T) {
	h := newTestHandler()
	router := NewRouter(h)

	req := httptest.NewRequest("GET", "/api/auth/me", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestAuthMe_ValidToken(t *testing.T) {
	h := newTestHandler()
	router := NewRouter(h)
	tok := generateTestToken(h.jwtSecret)

	req := httptest.NewRequest("GET", "/api/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var user auth.UserInfo
	if err := json.Unmarshal(rec.Body.Bytes(), &user); err != nil {
		t.Fatalf("failed to parse user: %v", err)
	}

	if user.UserID != "test-user" {
		t.Errorf("expected user_id test-user, got %q", user.UserID)
	}
}

func TestConversations_Unauthorized(t *testing.T) {
	h := newTestHandler()
	router := NewRouter(h)

	req := httptest.NewRequest("POST", "/api/conversations", strings.NewReader(`{"user_id":"u1"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestConversations_CreateAndList(t *testing.T) {
	h := newTestHandler()
	router := NewRouter(h)
	tok := generateTestToken(h.jwtSecret)

	createReq := httptest.NewRequest("POST", "/api/conversations", strings.NewReader(`{"user_id":"u1"}`))
	createReq.Header.Set("Authorization", "Bearer "+tok)
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", createRec.Code, createRec.Body.String())
	}

	var conv Conversation
	json.Unmarshal(createRec.Body.Bytes(), &conv)

	listReq := httptest.NewRequest("GET", "/api/conversations", nil)
	listReq.Header.Set("Authorization", "Bearer "+tok)
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", listRec.Code)
	}

	getReq := httptest.NewRequest("GET", "/api/conversations/"+conv.ID, nil)
	getReq.Header.Set("Authorization", "Bearer "+tok)
	getRec := httptest.NewRecorder()
	router.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", getRec.Code)
	}
}

func TestChatEvents_ClientDisconnect(t *testing.T) {
	h := newTestHandler()
	router := NewRouter(h)

	tok := generateTestToken(h.jwtSecret)
	req := httptest.NewRequest("GET", "/api/chat/disconnect-run/events?token="+tok, nil)

	ctx, cancel := timeoutContext(50 * time.Millisecond)
	defer cancel()
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func timeoutContext(d time.Duration) (ctx *timeoutCtx, cancel func()) {
	ctx = &timeoutCtx{done: make(chan struct{})}
	cancel = func() {
		select {
		case <-ctx.done:
		default:
			close(ctx.done)
		}
	}
	return ctx, cancel
}

type timeoutCtx struct {
	done chan struct{}
}

func (c *timeoutCtx) Deadline() (time.Time, bool) { return time.Time{}, false }
func (c *timeoutCtx) Done() <-chan struct{}        { return c.done }
func (c *timeoutCtx) Err() error {
	select {
	case <-c.done:
		return contextCanceled{}
	default:
		return nil
	}
}
func (c *timeoutCtx) Value(key interface{}) interface{} { return nil }

type contextCanceled struct{}

func (contextCanceled) Error() string { return "context canceled" }
