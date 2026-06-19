package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateChatHandler_WithConversationID(t *testing.T) {
	h := newTestHandler()
	router := NewRouter(h)

	body := `{"message":"test","conversationId":"conv-123"}`
	req := httptest.NewRequest("POST", "/api/chat", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+generateTestToken(h.jwtSecret))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "conversationId") {
		t.Error("missing conversationId in response:", rec.Body.String())
	}
}

func TestChatEventsHandler_MissingRunID(t *testing.T) {
	h := newTestHandler()
	router := NewRouter(h)

	token := generateTestToken(h.jwtSecret)
	req := httptest.NewRequest("GET", "/api/chat//events?token="+token, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestChatEventsHandler_NoAgentWithRunID(t *testing.T) {
	h := newTestHandler()
	router := NewRouter(h)

	token := generateTestToken(h.jwtSecret)
	h.StoreRunMessage("noagent", "test")

	req := httptest.NewRequest("GET", "/api/chat/noagent/events?token="+token, nil)
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		router.ServeHTTP(rec, req)
		close(done)
	}()
	<-done

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if rec.Body.Len() == 0 {
		t.Error("expected non-empty SSE body")
	}
}
