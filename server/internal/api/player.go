package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/music-agent/music-agent/internal/tme"
)

type PlayerHandler struct {
	client *tme.Client
	creds  *tme.CredentialStore
}

func NewPlayerHandler(client *tme.Client, creds *tme.CredentialStore) *PlayerHandler {
	return &PlayerHandler{client: client, creds: creds}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func (h *PlayerHandler) HandleGetPlayURL(w http.ResponseWriter, r *http.Request) {
	songID := strings.TrimPrefix(r.URL.Path, "/api/player/url/")
	if songID == "" {
		writeJSON(w, 400, map[string]any{"error": "song_id required"})
		return
	}

	if h.creds.IsLoggedIn() {
		mid, mk := h.creds.Get()
		h.client.SetCredential(mid, mk)
	}

	url, err := h.client.GetSongURL(r.Context(), songID)
	if err != nil {
		writeJSON(w, 500, map[string]any{"error": err.Error()})
		return
	}

	source := "cdn"
	if url.URL == "" {
		source = "unavailable"
	}

	writeJSON(w, 200, map[string]any{
		"song_id":            url.SongID,
		"url":                url.URL,
		"expires_in_seconds": url.ExpiresInSeconds,
		"source":             source,
	})
}

func (h *PlayerHandler) HandleGetLyrics(w http.ResponseWriter, r *http.Request) {
	songID := strings.TrimPrefix(r.URL.Path, "/api/player/lyrics/")
	if songID == "" {
		writeJSON(w, 400, map[string]any{"error": "song_id required"})
		return
	}

	lyrics, err := h.client.GetLyrics(r.Context(), songID)
	if err != nil {
		writeJSON(w, 500, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, 200, lyrics)
}
