package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/music-agent/music-agent/internal/auth"
)

type playlistItem struct {
	SongID   string `json:"songId"`
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	CoverURL string `json:"coverUrl,omitempty"`
}

func (h *Handler) createPlaylistHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := auth.UserFromContext(ctx)
	if user == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	var req struct {
		Name  string         `json:"name"`
		Songs []playlistItem `json:"songs,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		http.Error(w, `{"error":"name required"}`, http.StatusBadRequest)
		return
	}
	id := uuid.New().String()
	_, err := h.db.Exec(ctx,
		`INSERT INTO playlists (id, user_id, name) VALUES ($1,$2,$3)`, id, user.UserID, req.Name)
	if err != nil {
		http.Error(w, `{"error":"create failed"}`, http.StatusInternalServerError)
		return
	}
	for _, s := range req.Songs {
		_, err := h.db.Exec(ctx,
			`INSERT INTO playlist_songs (playlist_id, song_id, title, artist, cover_url) VALUES ($1,$2,$3,$4,$5)`,
			id, s.SongID, s.Title, s.Artist, s.CoverURL)
		if err != nil {
			h.logger.Warn("failed to add song to playlist", "error", err)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"id": id, "name": req.Name})
}

func (h *Handler) listPlaylistsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := auth.UserFromContext(ctx)
	if user == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	rows, err := h.db.Query(ctx,
		`SELECT id, name, created_at FROM playlists WHERE user_id=$1 ORDER BY updated_at DESC`, user.UserID)
	if err != nil {
		http.Error(w, `{"error":"list failed"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var result []map[string]any
	for rows.Next() {
		var id, name string
		var ca time.Time
		if rows.Scan(&id, &name, &ca) == nil {
			result = append(result, map[string]any{"id": id, "name": name, "createdAt": ca.Format(time.RFC3339)})
		}
	}
	if result == nil {
		result = []map[string]any{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *Handler) getPlaylistHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	if h.db == nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	var name string
	err := h.db.QueryRow(ctx, `SELECT name FROM playlists WHERE id=$1`, id).Scan(&name)
	if err != nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	rows, _ := h.db.Query(ctx,
		`SELECT song_id, title, artist, cover_url FROM playlist_songs WHERE playlist_id=$1 ORDER BY added_at`, id)
	defer rows.Close()
	var songs []map[string]any
	for rows != nil && rows.Next() {
		var sid, t, a, c string
		if rows.Scan(&sid, &t, &a, &c) == nil {
			songs = append(songs, map[string]any{"songId": sid, "title": t, "artist": a, "coverUrl": c})
		}
	}
	if songs == nil {
		songs = []map[string]any{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"id": id, "name": name, "songs": songs})
}

func (h *Handler) addToPlaylistHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := auth.UserFromContext(ctx)
	if user == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	var song playlistItem
	if json.NewDecoder(r.Body).Decode(&song) != nil || song.SongID == "" {
		http.Error(w, `{"error":"song required"}`, http.StatusBadRequest)
		return
	}
	_, err := h.db.Exec(ctx,
		`INSERT INTO playlist_songs (playlist_id, song_id, title, artist, cover_url) VALUES ($1,$2,$3,$4,$5)`,
		id, song.SongID, song.Title, song.Artist, song.CoverURL)
	if err != nil {
		http.Error(w, `{"error":"add failed"}`, http.StatusInternalServerError)
		return
	}
	_, err = h.db.Exec(ctx, `UPDATE playlists SET updated_at=now() WHERE id=$1`, id)
	if err != nil {
		h.logger.Warn("failed to update playlist timestamp", "error", err)
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) deletePlaylistHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	if h.db != nil {
		_, err := h.db.Exec(ctx, `DELETE FROM playlists WHERE id=$1`, id)
		if err != nil {
			h.logger.Error("failed to delete playlist", "error", err)
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) renamePlaylistHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	var req struct {
		Name string `json:"name"`
	}
	if json.NewDecoder(r.Body).Decode(&req) != nil || req.Name == "" {
		http.Error(w, `{"error":"name required"}`, http.StatusBadRequest)
		return
	}
	if h.db != nil {
		_, err := h.db.Exec(ctx,
			`UPDATE playlists SET name=$1, updated_at=now() WHERE id=$2`, req.Name, id)
		if err != nil {
			http.Error(w, `{"error":"rename failed"}`, http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
}
