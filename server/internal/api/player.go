package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/music-agent/music-agent/internal/tme"
)

type PlayerHandler struct {
	client *tme.Client
	creds  *tme.CredentialStore
	logger *slog.Logger
}

func NewPlayerHandler(client *tme.Client, creds *tme.CredentialStore) *PlayerHandler {
	return &PlayerHandler{client: client, creds: creds, logger: slog.Default()}
}

func (h *PlayerHandler) SetLogger(logger *slog.Logger) {
	h.logger = logger
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func (h *PlayerHandler) HandleGetPlayURL(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	songID := strings.TrimPrefix(r.URL.Path, "/api/player/url/")
	h.logger.DebugContext(ctx, "get play URL", "song_id", songID, "logged_in", h.creds.IsLoggedIn())
	if songID == "" {
		writeJSON(w, 400, map[string]any{"error": "song_id required"})
		return
	}

	if h.creds.IsLoggedIn() {
		mid, mk := h.creds.Get()
		h.client.SetCredential(mid, mk)
	}

	url, err := h.client.GetSongURL(ctx, songID)
	if err != nil {
		h.logger.ErrorContext(ctx, "get song URL failed", "error", err, "song_id", songID)
		writeJSON(w, 500, map[string]any{"error": err.Error()})
		return
	}

	source := "cdn"
	errMsg := ""
	if url.URL == "" {
		source = "unavailable"
		errMsg = "该歌曲暂无播放源"
	}
	h.logger.DebugContext(ctx, "song URL result", "song_id", songID, "source", source, "url_len", len(url.URL))

	writeJSON(w, 200, map[string]any{
		"song_id":            url.SongID,
		"url":                url.URL,
		"expires_in_seconds": url.ExpiresInSeconds,
		"source":             source,
		"error_msg":          errMsg,
	})
}

func (h *PlayerHandler) HandleGetLyrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	songID := strings.TrimPrefix(r.URL.Path, "/api/player/lyrics/")
	if songID == "" {
		writeJSON(w, 400, map[string]any{"error": "song_id required"})
		return
	}

	lyrics, err := h.client.GetLyrics(ctx, songID)
	if err != nil {
		h.logger.ErrorContext(ctx, "get lyrics failed", "error", err, "song_id", songID)
		writeJSON(w, 500, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, 200, lyrics)
}
