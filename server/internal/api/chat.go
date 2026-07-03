package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

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
	ctx := r.Context()
	user := auth.UserFromContext(ctx)
	if user == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	if h.credStore != nil {
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

	h.LogBehavior(ctx, user.UserID, "search", req.Message)
	h.saveTaskMemory(ctx, user.UserID, req.Message)
	h.updatePreferences(ctx, user.UserID, req.Message)

	runID := uuid.New().String()
	ctx = withRunID(ctx, runID)
	h.StoreRunMessage(runID, req.Message)

	if req.ConversationID != "" {
		h.SetRunConversation(runID, req.ConversationID)
		h.saveMessage(ctx, req.ConversationID, "user", req.Message, "{}")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"runId":          runID,
			"conversationId": req.ConversationID,
		})
		return
	}

	convID := uuid.New().String()
	title := req.Message
	if len([]rune(title)) > 20 {
		title = string([]rune(title)[:20])
	}
	if h.db != nil {
		now := time.Now()
		_, err := h.db.Exec(ctx,
			`INSERT INTO conversations (id, user_id, title, status, created_at, updated_at) VALUES ($1,$2,$3,'active',$4,$5)`,
			convID, user.UserID, title, now, now)
		if err != nil {
			h.logger.ErrorContext(ctx, "failed to create conversation", "error", err)
			http.Error(w, `{"error":"failed to create conversation"}`, http.StatusInternalServerError)
			return
		}
		h.saveMessage(ctx, convID, "user", req.Message, "{}")
	}
	h.SetRunConversation(runID, convID)
	go h.summarizeConversationTitle(convID, req.Message)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"runId":          runID,
		"conversationId": convID,
	})
}

func (h *Handler) chatEventsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
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

	runCtx := withRunID(ctx, runID)

	eventCh := h.startAgentEventCh(runCtx, runID, message)
	h.writeSSEWithSave(w, rc, runCtx, eventCh, runID)
}

func (h *Handler) startAgentEventCh(ctx context.Context, runID, message string) <-chan event.Event {
	convID := h.convIDForRun(runID)
	state := agent.LoopState{
		RunID:            runID,
		UserID:           "",
		Goal:             tool.AgentGoal{Intent: message, TaskType: "chat"},
		MaxSteps:         h.maxSteps,
		ExecutedCalls:    make(map[string]bool),
		MessageHistory:   h.loadConversationHistory(ctx, convID),
	}

	if user := auth.UserFromContext(ctx); user != nil {
		state.UserID = user.UserID
	}

	agentCtx, cancel := context.WithCancel(ctx)
	if state.UserID != "" {
		agentCtx = tool.WithUserID(agentCtx, state.UserID)
	}
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

func (h *Handler) writeSSEWithSave(w http.ResponseWriter, rc *http.ResponseController, ctx context.Context, ch <-chan event.Event, runID string) {
	var agentContent string
	var songsJSON string
	h.logger.DebugContext(ctx, "writeSSEWithSave started", "runID", runID)
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

			if evt.Type == event.TypeDelta {
				var delta map[string]string
				if json.Unmarshal(evt.Data, &delta) == nil {
					agentContent += delta["message"]
				}
			} else if evt.Type == event.TypeToolDone {
				var obs map[string]any
				if json.Unmarshal(evt.Data, &obs) == nil {
					if result, ok := obs["Result"]; ok {
						if r, ok := result.(map[string]any); ok {
							if d, ok := r["data"]; ok {
								if s, ok := d.(string); ok {
									songsJSON = s
								}
							}
						}
					}
				}
			} else if evt.Type == event.TypeDone {
				meta := "{}"
				if songsJSON != "" {
					meta = `{"songs":` + songsJSON + `}`
				}
				convID := h.convIDForRun(runID)
				h.logger.DebugContext(ctx, "saving agent message", "runID", runID, "convID", convID, "contentLen", len(agentContent))
				h.saveMessage(ctx, convID, "agent", agentContent, meta)
				return
			} else if evt.Type == event.TypeError {
				return
			}
		}
	}
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

func (h *Handler) summarizeConversationTitle(convID, message string) {
	if h.db == nil || h.agent == nil {
		return
	}
	if planner, ok := h.agent.(interface {
		SummarizeConversation(ctx context.Context, msg string) (string, error)
	}); ok {
		bgCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		title, err := planner.SummarizeConversation(bgCtx, message)
		if err == nil && title != "" {
			_, err := h.db.Exec(bgCtx, `UPDATE conversations SET title=$1 WHERE id=$2`, title, convID)
			if err != nil {
			h.logCtx(bgCtx).WarnContext(bgCtx, "failed to update conversation title", "error", err, "convID", convID)
			}
		}
	}
}
