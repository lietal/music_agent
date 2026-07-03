package api

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/music-agent/music-agent/internal/tme"
)

type StreamHandler struct {
	client *tme.Client
	logger *slog.Logger
}

func NewStreamHandler(client *tme.Client) *StreamHandler {
	return &StreamHandler{client: client, logger: slog.Default()}
}

func (h *StreamHandler) SetLogger(logger *slog.Logger) {
	h.logger = logger
}

func (h *StreamHandler) HandleStream(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	songID := strings.TrimPrefix(r.URL.Path, "/api/player/stream/")
	h.logger.DebugContext(ctx, "stream request", "song_id", songID)
	if songID == "" {
		writeJSON(w, 400, map[string]any{"error": "song_id required"})
		return
	}

	url, err := h.client.GetSongURL(ctx, songID)
	if err != nil {
		h.logger.ErrorContext(ctx, "stream get URL failed", "error", err, "song_id", songID)
		writeJSON(w, 500, map[string]any{"error": err.Error()})
		return
	}

	if url.URL == "" {
		h.logger.WarnContext(ctx, "stream URL empty", "song_id", songID)
		writeJSON(w, 404, map[string]any{"error": "song url not found"})
		return
	}

	h.logger.DebugContext(ctx, "stream proxying", "song_id", songID, "url_len", len(url.URL))

	cdnReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url.URL, nil)
	if err != nil {
		h.logger.ErrorContext(ctx, "create cdn request failed", "error", err)
		writeJSON(w, 500, map[string]any{"error": fmt.Sprintf("create cdn request: %v", err)})
		return
	}

	if rangeHdr := r.Header.Get("Range"); rangeHdr != "" {
		cdnReq.Header.Set("Range", rangeHdr)
	}

	cdnResp, err := http.DefaultClient.Do(cdnReq)
	if err != nil {
		h.logger.ErrorContext(ctx, "cdn request failed", "error", err)
		writeJSON(w, 502, map[string]any{"error": fmt.Sprintf("cdn request failed: %v", err)})
		return
	}
	defer cdnResp.Body.Close()

	if cdnResp.StatusCode >= 400 {
		h.logger.ErrorContext(ctx, "cdn returned error", "status", cdnResp.StatusCode)
		writeJSON(w, 502, map[string]any{"error": fmt.Sprintf("cdn returned status %d", cdnResp.StatusCode)})
		return
	}

	for k, v := range cdnResp.Header {
		if k == "Content-Type" || k == "Content-Range" || k == "Content-Length" {
			w.Header().Set(k, v[0])
		}
	}

	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.WriteHeader(cdnResp.StatusCode)

	io.Copy(w, cdnResp.Body)
}
