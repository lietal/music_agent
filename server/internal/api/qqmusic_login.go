package api

import (
	"net/http"
	"strings"

	"github.com/music-agent/music-agent/internal/tme"
)

type LoginHandler struct {
	client *tme.Client
	creds  *tme.CredentialStore
}

func NewLoginHandler(client *tme.Client, creds *tme.CredentialStore) *LoginHandler {
	return &LoginHandler{client: client, creds: creds}
}

func (h *LoginHandler) HandleGetQRCode(w http.ResponseWriter, r *http.Request) {
	qr, err := h.client.GetLoginQRCode(r.Context())
	if err != nil {
		writeJSON(w, 500, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]any{
		"qrcode_url": qr.QrcodeDataURL,
		"key":        qr.Key,
	})
}

func (h *LoginHandler) HandleCheckQRStatus(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.URL.Path, "/api/qqmusic/login/status/")
	if key == "" {
		writeJSON(w, 400, map[string]any{"error": "key required"})
		return
	}

	status, err := h.client.CheckQRCodeStatus(r.Context(), key)
	if err != nil {
		writeJSON(w, 500, map[string]any{"error": err.Error()})
		return
	}

	if status.Status == "confirmed" {
		h.creds.Set(status.MusicID, status.MusicKey)
	}

	writeJSON(w, 200, map[string]any{
		"status":    status.Status,
		"user_name": status.UserName,
	})
}

func (h *LoginHandler) HandleGetStatus(w http.ResponseWriter, r *http.Request) {
	if h.creds.IsLoggedIn() {
		_, _ = h.creds.Get()
		writeJSON(w, 200, map[string]any{
			"logged_in": true,
		})
		return
	}
	writeJSON(w, 200, map[string]any{
		"logged_in": false,
	})
}
