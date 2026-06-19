package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/music-agent/music-agent/internal/agent"
	"github.com/music-agent/music-agent/internal/auth"
	"github.com/music-agent/music-agent/internal/event"
	"github.com/music-agent/music-agent/internal/tool"
)

type chatRequest struct {
	Message        string `json:"message"`
	ConversationID string `json:"conversationId,omitempty"`
}

func (h *Handler) createChatHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if req.Message == "" {
		http.Error(w, `{"error":"message is required"}`, http.StatusBadRequest)
		return
	}

	runID := uuid.New().String()
	h.StoreRunMessage(runID, req.Message)

	if req.ConversationID != "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"runId":          runID,
			"conversationId": req.ConversationID,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"runId": runID,
	})
}

func (h *Handler) chatEventsHandler(w http.ResponseWriter, r *http.Request) {
	runID := chi.URLParam(r, "runId")
	if runID == "" {
		http.Error(w, `{"error":"missing runId"}`, http.StatusBadRequest)
		return
	}

	if h.agent == nil {
		h.fallbackMockSSE(w, r, runID)
		return
	}

	message, ok := h.PopRunMessage(runID)
	if !ok {
		http.Error(w, `{"error":"unknown runId"}`, http.StatusBadRequest)
		return
	}

	rc, err := setupSSE(w)
	if err != nil {
		return
	}

	eventCh := h.startAgentEventCh(r.Context(), runID, message)
	writeSSEEvents(w, rc, r.Context(), eventCh)
}

func (h *Handler) startAgentEventCh(ctx context.Context, runID, message string) <-chan event.Event {
	state := agent.LoopState{
		RunID:         runID,
		UserID:        "",
		Goal:          tool.AgentGoal{Intent: message, TaskType: "chat"},
		MaxSteps:      3,
		ExecutedCalls: make(map[string]bool),
	}

	if user := auth.UserFromContext(ctx); user != nil {
		state.UserID = user.UserID
	}

	agentCtx, cancel := context.WithCancel(ctx)
	go func() {
		<-ctx.Done()
		cancel()
	}()

	return h.agent.Run(agentCtx, state)
}

func (h *Handler) fallbackMockSSE(w http.ResponseWriter, r *http.Request, runID string) {
	rc, err := setupSSE(w)
	if err != nil {
		return
	}

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	ch := h.bus.Subscribe(runID)
	defer h.bus.Unsubscribe(runID)

	go runMockAgent(ctx, h.bus, runID)

	writeSSEEvents(w, rc, ctx, ch)
}

func setupSSE(w http.ResponseWriter) (*http.ResponseController, error) {
	rc := http.NewResponseController(w)
	if rc == nil {
		http.Error(w, `{"error":"streaming not supported"}`, http.StatusInternalServerError)
		return nil, fmt.Errorf("streaming not supported")
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	rc.Flush()
	return rc, nil
}

func writeSSEEvents(w http.ResponseWriter, rc *http.ResponseController, ctx context.Context, ch <-chan event.Event) {
	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", evt.Type, evt.Data)
			rc.Flush()
			if evt.Type == event.TypeDone || evt.Type == event.TypeError {
				return
			}
		}
	}
}

func runMockAgent(ctx context.Context, bus *event.Bus, runID string) {
	planData, _ := json.Marshal(map[string]interface{}{
		"plan": "search",
	})
	bus.Publish(event.Event{Type: event.TypePlan, RunID: runID, Data: planData})

	toolStartData, _ := json.Marshal(map[string]string{"name": "search_songs", "input": "searching"})
	bus.Publish(event.Event{Type: event.TypeToolStart, RunID: runID, Data: toolStartData})

	toolDoneData, _ := json.Marshal(map[string]interface{}{
		"tool":  "search_songs",
		"songs": []map[string]string{
			{"id": "1", "title": "晴天", "artist": "周杰伦"},
			{"id": "2", "title": "七里香", "artist": "周杰伦"},
			{"id": "3", "title": "稻香", "artist": "周杰伦"},
		},
	})
	bus.Publish(event.Event{Type: event.TypeToolDone, RunID: runID, Data: toolDoneData})

	deltaData, _ := json.Marshal(map[string]string{"message": "为你找到了周杰伦的热门歌曲：晴天、七里香、稻香 🎵"})
	bus.Publish(event.Event{Type: event.TypeDelta, RunID: runID, Data: deltaData})

	bus.Publish(event.Event{Type: event.TypeDone, RunID: runID})
}
