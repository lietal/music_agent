package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/music-agent/music-agent/internal/event"
)

func TestNewHandler(t *testing.T) {
	bus := event.NewBus()
	h := NewHandler(bus, []byte("secret"), nil)
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
	if h.bus != bus {
		t.Error("bus not set")
	}
	if string(h.jwtSecret) != "secret" {
		t.Error("jwt secret not set")
	}
}

func TestHandler_SetAgent(t *testing.T) {
	h := NewHandler(event.NewBus(), []byte("secret"), nil)
	if h.agent != nil {
		t.Error("expected nil agent initially")
	}
}

func TestHandler_StoreAndPopRunMessage(t *testing.T) {
	h := NewHandler(event.NewBus(), []byte("secret"), nil)
	h.StoreRunMessage("run1", "hello world")
	h.StoreRunMessage("run2", "another message")

	msg, ok := h.PopRunMessage("run1")
	if !ok {
		t.Fatal("expected run1 to exist")
	}
	if msg != "hello world" {
		t.Errorf("got %q", msg)
	}

	_, ok = h.PopRunMessage("run1")
	if ok {
		t.Error("expected run1 to be deleted after pop")
	}

	msg, ok = h.PopRunMessage("run2")
	if !ok {
		t.Error("expected run2 to exist")
	}
	if msg != "another message" {
		t.Errorf("got %q", msg)
	}

	_, ok = h.PopRunMessage("nonexistent")
	if ok {
		t.Error("expected missing key")
	}
}

func TestCreateChatHandler_ValidBody(t *testing.T) {
	h := newTestHandler()
	router := NewRouter(h)

	req := httptest.NewRequest("POST", "/api/chat", strings.NewReader(`{"message":"测试消息"}`))
	req.Header.Set("Authorization", "Bearer "+generateTestToken(h.jwtSecret))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	msg, ok := h.PopRunMessage("")
	if ok {
		t.Log("found message:", msg)
	}
}

func TestCreateChatHandler_EmptyMessage(t *testing.T) {
	h := newTestHandler()
	router := NewRouter(h)

	req := httptest.NewRequest("POST", "/api/chat", strings.NewReader(`{"message":""}`))
	req.Header.Set("Authorization", "Bearer "+generateTestToken(h.jwtSecret))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestCreateChatHandler_InvalidJSON(t *testing.T) {
	h := newTestHandler()
	router := NewRouter(h)

	req := httptest.NewRequest("POST", "/api/chat", strings.NewReader(`not json`))
	req.Header.Set("Authorization", "Bearer "+generateTestToken(h.jwtSecret))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestJWTAuthMiddleware(t *testing.T) {
	mw := JWTAuthMiddleware([]byte("test-secret"))
	if mw == nil {
		t.Fatal("expected non-nil middleware")
	}
}
