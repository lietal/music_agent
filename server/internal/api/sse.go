package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/music-agent/music-agent/internal/event"
)

func SSEHandler(bus *event.Bus) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		runID := r.URL.Query().Get("run_id")
		if runID == "" {
			http.Error(w, "missing run_id", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("X-Accel-Buffering", "no")
		w.Header().Set("Connection", "keep-alive")

		rc := http.NewResponseController(w)

		rc.Flush()

		ch := bus.Subscribe(runID)
		defer bus.Unsubscribe(runID)

		heartbeat := time.NewTicker(30 * time.Second)
		defer heartbeat.Stop()

		for {
			select {
			case evt, ok := <-ch:
				if !ok {
					return
				}
				data, _ := json.Marshal(evt.Data)
				fmt.Fprintf(w, "event: %s\ndata: %s\n\n", evt.Type, data)
				rc.Flush()
			case <-heartbeat.C:
				fmt.Fprintf(w, ": heartbeat\n\n")
				rc.Flush()
			case <-r.Context().Done():
				return
			}
		}
	}
}
