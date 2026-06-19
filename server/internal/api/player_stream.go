package api

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/music-agent/music-agent/internal/tme"
)

type StreamHandler struct {
	client *tme.Client
}

func NewStreamHandler(client *tme.Client) *StreamHandler {
	return &StreamHandler{client: client}
}

func (h *StreamHandler) HandleStream(w http.ResponseWriter, r *http.Request) {
	songID := strings.TrimPrefix(r.URL.Path, "/api/player/stream/")
	if songID == "" {
		writeJSON(w, 400, map[string]any{"error": "song_id required"})
		return
	}

	url, err := h.client.GetSongURL(r.Context(), songID)
	if err != nil {
		writeJSON(w, 500, map[string]any{"error": err.Error()})
		return
	}

	if url.URL == "" {
		writeJSON(w, 404, map[string]any{"error": "song url not found"})
		return
	}

	cdnReq, err := http.NewRequestWithContext(r.Context(), http.MethodGet, url.URL, nil)
	if err != nil {
		writeJSON(w, 500, map[string]any{"error": fmt.Sprintf("create cdn request: %v", err)})
		return
	}

	if rangeHdr := r.Header.Get("Range"); rangeHdr != "" {
		cdnReq.Header.Set("Range", rangeHdr)
	}

	cdnResp, err := http.DefaultClient.Do(cdnReq)
	if err != nil {
		writeJSON(w, 502, map[string]any{"error": fmt.Sprintf("cdn request failed: %v", err)})
		return
	}
	defer cdnResp.Body.Close()

	if cdnResp.StatusCode >= 400 {
		writeJSON(w, 502, map[string]any{"error": fmt.Sprintf("cdn returned status %d", cdnResp.StatusCode)})
		return
	}

	if contentType := cdnResp.Header.Get("Content-Type"); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	if contentRange := cdnResp.Header.Get("Content-Range"); contentRange != "" {
		w.Header().Set("Content-Range", contentRange)
	}
	if contentLength := cdnResp.Header.Get("Content-Length"); contentLength != "" {
		w.Header().Set("Content-Length", contentLength)
	}

	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.WriteHeader(cdnResp.StatusCode)

	io.Copy(w, cdnResp.Body)
}
