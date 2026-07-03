package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/music-agent/music-agent/internal/auth"
)

func wechatConfig() (string, string) {
	return "", ""
}

func (h *Handler) wechatQRLoginHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	appID, _ := wechatConfig()
	if appID == "" {
		http.Error(w, `{"error":"WeChat login not configured"}`, http.StatusServiceUnavailable)
		return
	}
	redirectURI := "http://" + r.Host + "/api/auth/callback/wechat"
	authURL := fmt.Sprintf(
		"https://open.weixin.qq.com/connect/qrconnect?appid=%s&redirect_uri=%s&response_type=code&scope=snsapi_login&state=STATE#wechat_redirect",
		appID, url.QueryEscape(redirectURI),
	)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"qrcode_url": authURL, "type": "wechat_oauth_url"})
	_ = ctx
}

func (h *Handler) wechatCallbackHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	appID, appSecret := wechatConfig()
	if appID == "" || appSecret == "" {
		http.Error(w, `{"error":"WeChat login not configured"}`, http.StatusServiceUnavailable)
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, `{"error":"no code"}`, http.StatusBadRequest)
		return
	}
	tokenURL := fmt.Sprintf(
		"https://api.weixin.qq.com/sns/oauth2/access_token?appid=%s&secret=%s&code=%s&grant_type=authorization_code",
		appID, appSecret, code,
	)
	resp, err := http.Get(tokenURL)
	if err != nil {
		http.Error(w, `{"error":"token exchange failed"}`, http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		OpenID       string `json:"openid"`
		UnionID      string `json:"unionid"`
		RefreshToken string `json:"refresh_token"`
		ErrCode      int    `json:"errcode"`
	}
	json.NewDecoder(resp.Body).Decode(&tokenResp)
	if tokenResp.ErrCode != 0 || tokenResp.OpenID == "" {
		http.Error(w, `{"error":"token exchange returned error"}`, http.StatusBadGateway)
		return
	}
	user, err := auth.FindOrCreateByProvider(ctx, h.db, "wechat", tokenResp.OpenID, "微信用户", "")
	if err != nil {
		http.Error(w, `{"error":"user creation failed"}`, http.StatusInternalServerError)
		return
	}
	jwt, err := auth.GenerateToken(user.UserID, "wechat", h.jwtSecret)
	if err != nil {
		http.Error(w, `{"error":"jwt generation failed"}`, http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/chat?token="+jwt, http.StatusFound)
}
