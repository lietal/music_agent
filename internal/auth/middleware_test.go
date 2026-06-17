package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
)

func TestJWTAuth_ValidToken(t *testing.T) {
	secret := []byte("test-secret")
	claims := jwt.MapClaims{
		"user_id":  "user123",
		"provider": "wechat",
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := tok.SignedString(secret)
	if err != nil {
		t.Fatal(err)
	}

	r := chi.NewRouter()
	r.Use(JWTAuth(secret))
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		user := UserFromContext(r.Context())
		if user == nil || user.UserID != "user123" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestJWTAuth_InvalidToken(t *testing.T) {
	secret := []byte("test-secret")

	r := chi.NewRouter()
	r.Use(JWTAuth(secret))
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestJWTAuth_ExpiredToken(t *testing.T) {
	secret := []byte("test-secret")
	claims := jwt.MapClaims{
		"user_id":  "user123",
		"provider": "wechat",
		"exp":      time.Now().Add(-1 * time.Hour).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := tok.SignedString(secret)

	r := chi.NewRouter()
	r.Use(JWTAuth(secret))
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestJWTAuth_QueryParamToken(t *testing.T) {
	secret := []byte("test-secret")
	claims := jwt.MapClaims{
		"user_id":  "user123",
		"provider": "wechat",
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := tok.SignedString(secret)

	r := chi.NewRouter()
	r.Use(JWTAuth(secret))
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		user := UserFromContext(r.Context())
		if user == nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/?token="+tokenString, nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestJWTAuth_NoToken(t *testing.T) {
	secret := []byte("test-secret")

	r := chi.NewRouter()
	r.Use(JWTAuth(secret))
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestJWTAuth_PreferAuthorizationHeader(t *testing.T) {
	secret := []byte("test-secret")

	headerClaims := jwt.MapClaims{
		"user_id":  "header-user",
		"provider": "wechat",
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
	}
	headerTok := jwt.NewWithClaims(jwt.SigningMethodHS256, headerClaims)
	headerToken, _ := headerTok.SignedString(secret)

	queryClaims := jwt.MapClaims{
		"user_id":  "query-user",
		"provider": "wechat",
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
	}
	queryTok := jwt.NewWithClaims(jwt.SigningMethodHS256, queryClaims)
	queryToken, _ := queryTok.SignedString(secret)

	r := chi.NewRouter()
	r.Use(JWTAuth(secret))
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		user := UserFromContext(r.Context())
		if user == nil || user.UserID != "header-user" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/?token="+queryToken, nil)
	req.Header.Set("Authorization", "Bearer "+headerToken)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}
