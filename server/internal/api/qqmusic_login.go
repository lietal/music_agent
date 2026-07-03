package api

import (
	"log/slog"
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
	logger    *slog.Logger
}

func NewLoginHandler(client *tme.Client, creds *tme.CredentialStore, jwtSecret []byte, db *pgxpool.Pool) *LoginHandler {
	return &LoginHandler{client: client, creds: creds, jwtSecret: jwtSecret, db: db, logger: slog.Default()}
}

func (h *LoginHandler) SetLogger(logger *slog.Logger) {
	h.logger = logger
}

func (h *LoginHandler) HandleGetQRCode(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	qr, err := h.client.GetLoginQRCode(ctx)
	if err != nil {
		h.logger.ErrorContext(ctx, "get QR code failed", "error", err)
		writeJSON(w, 500, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]any{
		"qrcode_url": qr.QrcodeDataURL,
		"key":        qr.Key,
	})
}

func (h *LoginHandler) HandleCheckQRStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	key := strings.TrimPrefix(r.URL.Path, "/api/qqmusic/login/status/")
	if key == "" {
		writeJSON(w, 400, map[string]any{"error": "key required"})
		return
	}

	status, err := h.client.CheckQRCodeStatus(ctx, key)
	if err != nil {
		h.logger.ErrorContext(ctx, "check QR status failed", "error", err)
		writeJSON(w, 500, map[string]any{"error": err.Error()})
		return
	}

	resp := map[string]any{
		"status":    status.Status,
		"user_name": status.UserName,
	}

	if status.Status == "confirmed" {
		h.creds.Set(status.MusicID, status.MusicKey)
		h.logger.InfoContext(ctx, "QR login confirmed", "musicid", status.MusicID, "openid", status.OpenID)

		if status.OpenID != "" && h.db != nil {
			user, err := auth.FindOrCreateByProvider(ctx, h.db,
				"qqmusic", status.OpenID, status.UserName, status.AvatarURL)
			if err != nil {
				h.logger.WarnContext(ctx, "failed to create user for QR login", "error", err)
			} else {
				token, err := auth.GenerateToken(user.UserID, "qqmusic", h.jwtSecret)
				if err != nil {
					h.logger.ErrorContext(ctx, "failed to generate token", "error", err)
				} else {
					resp["token"] = token
					resp["user"] = user
				}
			}
		}
	}

	writeJSON(w, 200, resp)
}

func (h *LoginHandler) HandleGetStatus(w http.ResponseWriter, r *http.Request) {
	_ = r.Context()
	if h.creds.IsLoggedIn() {
		mid, mk := h.creds.Get()
		_ = mid
		_ = mk
		writeJSON(w, 200, map[string]any{"logged_in": true})
		return
	}
	writeJSON(w, 200, map[string]any{"logged_in": false})
}
