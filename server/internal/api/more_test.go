package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAuthRegisterHandler_Valid(t *testing.T) {
	h := newTestHandler()
	router := NewRouter(h)

	body := `{"username":"newuser","password":"pass123","displayName":"New User"}`
	req := httptest.NewRequest("POST", "/api/auth/register", strings.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAuthRegisterHandler_InvalidBody(t *testing.T) {
	h := newTestHandler()
	router := NewRouter(h)

	req := httptest.NewRequest("POST", "/api/auth/register", strings.NewReader(`not json`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestAuthRegisterHandler_ShortPassword(t *testing.T) {
	h := newTestHandler()
	router := NewRouter(h)

	req := httptest.NewRequest("POST", "/api/auth/register", strings.NewReader(`{"username":"u","password":"ab","displayName":"Short"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	t.Log("short password response:", rec.Code, rec.Body.String())
}

func TestAuthLoginHandler_Valid(t *testing.T) {
	h := newTestHandler()
	router := NewRouter(h)

	regBody := `{"username":"loginuser","password":"pass123","displayName":"Login"}`
	regReq := httptest.NewRequest("POST", "/api/auth/register", strings.NewReader(regBody))
	regRec := httptest.NewRecorder()
	router.ServeHTTP(regRec, regReq)

	loginBody := `{"username":"loginuser","password":"pass123"}`
	loginReq := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(loginBody))
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)

	if loginRec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", loginRec.Code, loginRec.Body.String())
	}
}

func TestAuthLoginHandler_WrongPassword(t *testing.T) {
	h := newTestHandler()
	router := NewRouter(h)

	regBody := `{"username":"wpuser","password":"pass123","displayName":"WP"}`
	regReq := httptest.NewRequest("POST", "/api/auth/register", strings.NewReader(regBody))
	regRec := httptest.NewRecorder()
	router.ServeHTTP(regRec, regReq)

	loginBody := `{"username":"wpuser","password":"wrong"}`
	loginReq := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(loginBody))
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)

	if loginRec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", loginRec.Code)
	}
}

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	healthHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestCreateConversationHandler(t *testing.T) {
	h := newTestHandler()
	router := NewRouter(h)

	req := httptest.NewRequest("POST", "/api/conversations", strings.NewReader(`{"user_id":"u1"}`))
	req.Header.Set("Authorization", "Bearer "+generateTestToken(h.jwtSecret))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", rec.Code)
	}
}

func TestListConversationsHandler(t *testing.T) {
	h := newTestHandler()
	router := NewRouter(h)

	req := httptest.NewRequest("GET", "/api/conversations", nil)
	req.Header.Set("Authorization", "Bearer "+generateTestToken(h.jwtSecret))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}
