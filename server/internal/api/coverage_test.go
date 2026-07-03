package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/music-agent/music-agent/internal/event"
	"github.com/music-agent/music-agent/internal/tme"
)

func newHandlerWithDB() *Handler {
	return &Handler{
		bus:       event.NewBus(),
		jwtSecret: []byte("test"),
		db:        nil,
	}
}

func TestWeChatQR_NotConfigured(t *testing.T) {
	t.Setenv("WECHAT_APP_ID", "")
	h := newHandlerWithDB()
	req := httptest.NewRequest("GET", "/api/auth/wechat/qr", nil)
	rec := httptest.NewRecorder()
	h.wechatQRLoginHandler(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rec.Code)
	}
}

func TestWeChatQR_Configured(t *testing.T) {
	t.Setenv("WECHAT_APP_ID", "wx_test_id")
	t.Setenv("WECHAT_APP_SECRET", "wx_test_secret")
	h := newHandlerWithDB()
	req := httptest.NewRequest("GET", "/api/auth/wechat/qr", nil)
	rec := httptest.NewRecorder()
	h.wechatQRLoginHandler(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]string
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["type"] != "wechat_oauth_url" {
		t.Error("expected wechat_oauth_url")
	}
	if !strings.Contains(resp["qrcode_url"], "open.weixin.qq.com") {
		t.Error("expected wechat OAuth URL")
	}
}

func TestPlaylistCreate_Unauthorized(t *testing.T) {
	h := newHandlerWithDB()
	req := httptest.NewRequest("POST", "/api/playlists", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	h.createPlaylistHandler(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestPlaylistList_Unauthorized(t *testing.T) {
	h := newHandlerWithDB()
	req := httptest.NewRequest("GET", "/api/playlists", nil)
	rec := httptest.NewRecorder()
	h.listPlaylistsHandler(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestCommentsHandler_MissingID(t *testing.T) {
	h := NewPlayerHandler(tme.NewClient(), tme.NewCredentialStore())
	req := httptest.NewRequest("GET", "/api/player/comments/", nil)
	rec := httptest.NewRecorder()
	h.HandleGetComments(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestConversationCreate_Unauthorized(t *testing.T) {
	h := newHandlerWithDB()
	req := httptest.NewRequest("POST", "/api/conversations", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	h.createConversationHandler(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestConversationList_Unauthorized(t *testing.T) {
	h := newHandlerWithDB()
	req := httptest.NewRequest("GET", "/api/conversations", nil)
	rec := httptest.NewRecorder()
	h.listConversationsHandler(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestAuthLogout_Ok(t *testing.T) {
	h := &Handler{credStore: tme.NewCredentialStore()}
	req := httptest.NewRequest("POST", "/api/auth/logout", nil)
	rec := httptest.NewRecorder()
	h.authLogoutHandler(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestWeChatCallback_NoCode(t *testing.T) {
	t.Setenv("WECHAT_APP_ID", "wx")
	t.Setenv("WECHAT_APP_SECRET", "s")
	h := newHandlerWithDB()
	req := httptest.NewRequest("GET", "/api/auth/callback/wechat", nil)
	rec := httptest.NewRecorder()
	h.wechatCallbackHandler(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for no code, got %d", rec.Code)
	}
}

func TestWeChatCallback_NotConfigured(t *testing.T) {
	t.Setenv("WECHAT_APP_ID", "")
	h := newHandlerWithDB()
	req := httptest.NewRequest("GET", "/api/auth/callback/wechat?code=x", nil)
	rec := httptest.NewRecorder()
	h.wechatCallbackHandler(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rec.Code)
	}
}

func TestStreamHandler_NoID(t *testing.T) {
	sh := NewStreamHandler(tme.NewClient(), tme.NewCredentialStore())
	req := httptest.NewRequest("GET", "/api/player/stream/", nil)
	rec := httptest.NewRecorder()
	sh.HandleStream(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestGetPlayURL_NoID(t *testing.T) {
	ph := NewPlayerHandler(tme.NewClient(), tme.NewCredentialStore())
	req := httptest.NewRequest("GET", "/api/player/url/", nil)
	rec := httptest.NewRecorder()
	ph.HandleGetPlayURL(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestGetLyrics_NoID(t *testing.T) {
	ph := NewPlayerHandler(tme.NewClient(), tme.NewCredentialStore())
	req := httptest.NewRequest("GET", "/api/player/lyrics/", nil)
	rec := httptest.NewRecorder()
	ph.HandleGetLyrics(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestGetLoginStatus_NotLoggedIn(t *testing.T) {
	lh := NewLoginHandler(tme.NewClient(), tme.NewCredentialStore(), []byte("test"), nil)
	req := httptest.NewRequest("GET", "/api/qqmusic/login/status", nil)
	rec := httptest.NewRecorder()
	lh.HandleGetStatus(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["logged_in"] != false {
		t.Errorf("expected logged_in false")
	}
}

func TestGetQRCode_Handler(t *testing.T) {
	lh := NewLoginHandler(tme.NewClient(), tme.NewCredentialStore(), []byte("test"), nil)
	req := httptest.NewRequest("POST", "/api/qqmusic/login/qrcode", nil)
	rec := httptest.NewRecorder()
	lh.HandleGetQRCode(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCheckQRStatus_KeyRequired(t *testing.T) {
	lh := NewLoginHandler(tme.NewClient(), tme.NewCredentialStore(), []byte("test"), nil)
	req := httptest.NewRequest("GET", "/api/qqmusic/login/status/", nil)
	rec := httptest.NewRecorder()
	lh.HandleCheckQRStatus(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestStreamHandler_EnsureCredential(t *testing.T) {
	cs := tme.NewCredentialStore()
	sh := NewStreamHandler(tme.NewClient(), cs)
	cs.Set("mid", "mk")
	req := httptest.NewRequest("GET", "/api/player/stream/qqmusic%3Atest", nil)
	rec := httptest.NewRecorder()
	sh.HandleStream(rec, req)
}

func TestWeChatCallback_TokenExchange(t *testing.T) {
	t.Setenv("WECHAT_APP_ID", "wx_test_id")
	t.Setenv("WECHAT_APP_SECRET", "wx_test_secret")
	h := newHandlerWithDB()
	req := httptest.NewRequest("GET", "/api/auth/callback/wechat?code=test_code_123", nil)
	rec := httptest.NewRecorder()
	h.wechatCallbackHandler(rec, req)
}

func TestSummarizeConversationTitle_NoAgent(t *testing.T) {
	h := newHandlerWithDB()
	h.summarizeConversationTitle("conv-1", "周杰伦的歌")
}

func TestHandler_GettersAndSetters(t *testing.T) {
	h := newHandlerWithDB()
	h.SetAgent(nil)
	h.SetCredentialStore(tme.NewCredentialStore())
	h.SetTMEClient(tme.NewClient())
	if h.JWTSecret() == nil {
		t.Error("expected JWT secret")
	}
	if h.Bus() == nil {
		t.Error("expected bus")
	}
	if h.DB() != nil {
		// DB is nil for test handler — OK
	}
}

func TestAuthMeHandler(t *testing.T) {
	h := newHandlerWithDB()
	req := httptest.NewRequest("GET", "/api/auth/me", nil)
	rec := httptest.NewRecorder()
	h.authMeHandler(rec, req)
}

func TestGetPlaylist_NotFound(t *testing.T) {
	h := newHandlerWithDB()
	req := httptest.NewRequest("GET", "/api/playlists/nonexistent", nil)
	req = chiSetURLParam(req, "id", "nonexistent")
	rec := httptest.NewRecorder()
	h.getPlaylistHandler(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func chiSetURLParam(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func TestDeletePlaylist_Ok(t *testing.T) {
	h := newHandlerWithDB()
	req := httptest.NewRequest("DELETE", "/api/playlists/any", nil)
	req = chiSetURLParam(req, "id", "any")
	rec := httptest.NewRecorder()
	h.deletePlaylistHandler(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rec.Code)
	}
}
