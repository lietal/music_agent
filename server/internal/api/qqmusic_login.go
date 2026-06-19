package api

import (
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/music-agent/music-agent/internal/auth"
	"github.com/music-agent/music-agent/internal/tme"
)

type LoginHandler struct {
	client    *tme.Client
	creds     *tme.CredentialStore
	jwtSecret []byte
	db        *pgxpool.Pool
}

func NewLoginHandler(client *tme.Client, creds *tme.CredentialStore, jwtSecret []byte, db *pgxpool.Pool) *LoginHandler {
	return &LoginHandler{client: client, creds: creds, jwtSecret: jwtSecret, db: db}
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

	resp := map[string]any{
		"status":    status.Status,
		"user_name": status.UserName,
	}

	if status.Status == "confirmed" {
		h.creds.Set(status.MusicID, status.MusicKey)

		if status.OpenID != "" && h.db != nil {
			user, err := auth.FindOrCreateByProvider(r.Context(), h.db,
				"qqmusic", status.OpenID, status.UserName, status.AvatarURL)
			if err == nil {
				token, err := auth.GenerateToken(user.UserID, "qqmusic", h.jwtSecret)
				if err == nil {
					resp["token"] = token
					resp["user"] = user
				}
			}
		}
	}

	writeJSON(w, 200, resp)
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
