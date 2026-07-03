package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/music-agent/music-agent/internal/auth"
)

var convs = map[string]map[string]any{}

func (h *Handler) createConversationHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := auth.UserFromContext(ctx)
	if user == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	id := uuid.New().String()
	if h.db != nil {
		_, err := h.db.Exec(ctx,
			`INSERT INTO conversations (id, user_id, title, status, created_at) VALUES ($1, $2, '', 'active', $3)`,
			id, user.UserID, time.Now())
		if err != nil {
			http.Error(w, `{"error":"failed to create conversation"}`, http.StatusInternalServerError)
			return
		}
	} else {
		convs[id] = map[string]any{"id": id, "title": "", "status": "active", "userID": user.UserID}
	}

	conv := map[string]any{"id": id, "title": "", "status": "active"}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(conv)
}

func (h *Handler) listConversationsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := auth.UserFromContext(ctx)
	if user == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var result []map[string]any
	if h.db != nil {
		rows, err := h.db.Query(ctx,
			`SELECT id, title, created_at, updated_at FROM conversations WHERE user_id=$1 ORDER BY updated_at DESC`,
			user.UserID)
		if err != nil {
			http.Error(w, `{"error":"failed to list conversations"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		for rows.Next() {
			var id1, title string
			var createdAt, updatedAt time.Time
			if err := rows.Scan(&id1, &title, &createdAt, &updatedAt); err != nil {
				continue
			}
			result = append(result, map[string]any{
				"id": id1, "title": title,
				"createdAt": createdAt.Format(time.RFC3339),
				"updatedAt": updatedAt.Format(time.RFC3339),
			})
		}
	} else {
		for _, c := range convs {
			result = append(result, c)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *Handler) getConversationHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	if h.db != nil {
		var title string
		err := h.db.QueryRow(ctx, `SELECT title FROM conversations WHERE id=$1`, id).Scan(&title)
		if err != nil {
			http.Error(w, `{"error":"conversation not found"}`, http.StatusNotFound)
			return
		}
		rows, err := h.db.Query(ctx,
			`SELECT id, role, content, metadata, created_at FROM messages WHERE conversation_id=$1 ORDER BY created_at`, id)
		var msgs []map[string]any
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var mid, role, content, meta string
				var ca time.Time
				if rows.Scan(&mid, &role, &content, &meta, &ca) == nil {
					m := map[string]any{
						"id": mid, "role": role, "content": content, "timestamp": ca.UnixMilli(),
					}
					if meta != "" && meta != "{}" {
						var parsed any
						if json.Unmarshal([]byte(meta), &parsed) == nil {
							if obj, ok := parsed.(map[string]any); ok {
								if songs, ok := obj["songs"]; ok {
									m["songs"] = normalizeSongs(songs)
								}
							}
						}
					}
					msgs = append(msgs, m)
				}
			}
		}
		if msgs == nil {
			msgs = []map[string]any{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"id": id, "title": title, "messages": msgs})
		return
	}
	conv, ok := convs[id]
	if !ok {
		http.Error(w, `{"error":"conversation not found"}`, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(conv)
}

func strval(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	default:
		return fmt.Sprintf("%v", val)
	}
}

func normalizeSongs(songs any) []map[string]any {
	arr, ok := songs.([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(arr))
	for _, s := range arr {
		raw, ok := s.(map[string]any)
		if !ok {
			continue
		}
		artist := strval(raw["artist"])
		if artist == "" {
			if artists, ok := raw["artists"].([]any); ok && len(artists) > 0 {
				artist = fmt.Sprintf("%v", artists[0])
			}
		}
		coverUrl := strval(raw["coverUrl"])
		if coverUrl == "" {
			coverUrl = strval(raw["artwork_url"])
		}
		if coverUrl == "" {
			coverUrl = strval(raw["cover_url"])
		}
		duration := 0
		if d, ok := raw["durationSeconds"].(float64); ok {
			duration = int(d)
		} else if d, ok := raw["duration_seconds"].(float64); ok {
			duration = int(d)
		}
		out = append(out, map[string]any{
			"id":              strval(raw["id"]),
			"title":           strval(raw["title"]),
			"artist":          artist,
			"album":           strval(raw["album"]),
			"coverUrl":        coverUrl,
			"durationSeconds": duration,
		})
	}
	return out
}
