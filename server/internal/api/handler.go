package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/music-agent/music-agent/internal/agent"
	"github.com/music-agent/music-agent/internal/auth"
	"github.com/music-agent/music-agent/internal/event"
	"github.com/music-agent/music-agent/internal/llm"
	"github.com/music-agent/music-agent/internal/tme"
)

type AgentRunner interface {
	Run(ctx context.Context, state agent.LoopState) <-chan event.Event
}

type Handler struct {
	bus        *event.Bus
	jwtSecret  []byte
	db         *pgxpool.Pool
	agent      AgentRunner
	credStore  *tme.CredentialStore
	tmeClient  *tme.Client
	activeRuns map[string]string
	runConvMap map[string]string
	mu         sync.RWMutex
	maxSteps   int
	logger     *slog.Logger
}

func NewHandler(bus *event.Bus, jwtSecret []byte, db *pgxpool.Pool, logger *slog.Logger) *Handler {
	return &Handler{
		bus:        bus,
		jwtSecret:  jwtSecret,
		db:         db,
		activeRuns: make(map[string]string),
		runConvMap: make(map[string]string),
		logger:     logger,
	}
}

func (h *Handler) SetAgent(a AgentRunner) {
	h.agent = a
}

func (h *Handler) logCtx(ctx context.Context) *slog.Logger {
	var attrs []slog.Attr
	if reqID := middleware.GetReqID(ctx); reqID != "" {
		attrs = append(attrs, slog.String("requestID", reqID))
	}
	if user := auth.UserFromContext(ctx); user != nil {
		attrs = append(attrs, slog.String("userID", user.UserID))
	}
	if len(attrs) == 0 {
		return h.logger
	}
	return h.logger.With(attrsToAny(attrs)...)
}

func attrsToAny(attrs []slog.Attr) []any {
	out := make([]any, len(attrs)*2)
	for i, a := range attrs {
		out[i*2] = a.Key
		out[i*2+1] = a.Value
	}
	return out
}

func (h *Handler) SetCredentialStore(cs *tme.CredentialStore) {
	h.credStore = cs
}

func (h *Handler) SetTMEClient(c *tme.Client) {
	h.tmeClient = c
}

func (h *Handler) SetMaxSteps(n int) {
	if n < 1 {
		n = 5
	}
	h.maxSteps = n
}

func (h *Handler) StoreRunMessage(runID, message string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.activeRuns[runID] = message
}

func (h *Handler) PopRunMessage(runID string) (string, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	msg, ok := h.activeRuns[runID]
	if ok {
		delete(h.activeRuns, runID)
	}
	return msg, ok
}

func (h *Handler) SetRunConversation(runID, convID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.runConvMap[runID] = convID
}

func (h *Handler) convIDForRun(runID string) string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.runConvMap[runID]
}

func (h *Handler) JWTSecret() []byte {
	return h.jwtSecret
}

func (h *Handler) Bus() *event.Bus {
	return h.bus
}

func (h *Handler) DB() *pgxpool.Pool {
	return h.db
}

func (h *Handler) saveMessage(ctx context.Context, conversationID, role, content, metadata string) {
	if h.db == nil || conversationID == "" {
		return
	}
	_, err := h.db.Exec(ctx,
		`INSERT INTO messages (conversation_id, role, content, metadata) VALUES ($1,$2,$3,$4)`,
		conversationID, role, content, metadata)
	if err != nil {
		h.logger.ErrorContext(ctx, "saveMessage failed", "error", err, "conversationID", conversationID)
		return
	}
	_, err = h.db.Exec(ctx,
		`UPDATE conversations SET updated_at=now() WHERE id=$1`, conversationID)
	if err != nil {
		h.logger.WarnContext(ctx, "failed to bump conversation updated_at", "error", err, "conversationID", conversationID)
	}
}

func (h *Handler) LogBehavior(ctx context.Context, userID, eventType, payload string) {
	if h.db == nil {
		return
	}
	
	_, err := h.db.Exec(ctx, `INSERT INTO behavior_events (user_id, event_type, payload) VALUES ($1, $2, $3)`,
		userID, eventType, payload)
	if err != nil {
		h.logger.WarnContext(ctx, "LogBehavior failed", "error", err, "userID", userID)
	}
}

func (h *Handler) saveTaskMemory(ctx context.Context, userID, query string) {
	if h.db == nil {
		return
	}
	
	_, err := h.db.Exec(ctx, `INSERT INTO task_memory (user_id, query) VALUES ($1, $2)`, userID, query)
	if err != nil {
		h.logger.WarnContext(ctx, "saveTaskMemory failed", "error", err, "userID", userID)
	}
}

func (h *Handler) updatePreferences(ctx context.Context, userID, query string) {
	if h.db == nil {
		return
	}
	
	_, err := h.db.Exec(ctx,
		`INSERT INTO user_preferences (user_id, key, polarity, confidence, updated_at)
		 VALUES ($1, $2, 'positive', 0.1, now())
		 ON CONFLICT (user_id, key) DO UPDATE SET confidence = user_preferences.confidence + 0.05, updated_at=now()`,
		userID, query)
	if err != nil {
		h.logger.WarnContext(ctx, "updatePreferences failed", "error", err, "userID", userID)
	}
}

func (h *Handler) cleanExpiredPreferences(ctx context.Context) {
	if h.db == nil {
		return
	}
	
	if _, err := h.db.Exec(ctx, `DELETE FROM task_memory WHERE created_at < now() - INTERVAL '30 days'`); err != nil {
		h.logger.WarnContext(ctx, "cleanExpiredPreferences: task_memory cleanup failed", "error", err)
	}
	if _, err := h.db.Exec(ctx, `DELETE FROM behavior_events WHERE created_at < now() - INTERVAL '90 days'`); err != nil {
		h.logger.WarnContext(ctx, "cleanExpiredPreferences: behavior_events cleanup failed", "error", err)
	}
}

func (h *Handler) loadConversationHistory(ctx context.Context, convID string) []llm.Message {
	if h.db == nil || convID == "" {
		return nil
	}
	rows, err := h.db.Query(ctx,
		`SELECT role, content, metadata FROM messages WHERE conversation_id=$1 ORDER BY created_at DESC LIMIT 6`, convID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var msgs []llm.Message
	for rows.Next() {
		var role, content, meta string
		if rows.Scan(&role, &content, &meta) != nil {
			continue
		}
		llmRole := "user"
		if role == "agent" {
			llmRole = "assistant"
		}
		if meta != "" && meta != "{}" {
			if songs := extractSongsFromMeta(meta); songs != "" {
				content += "\n\nRecommended songs: " + songs
			}
		}
		msgs = append([]llm.Message{{Role: llmRole, Content: content}}, msgs...)
	}
	return msgs
}

func extractSongsFromMeta(meta string) string {
	var parsed map[string]any
	if json.Unmarshal([]byte(meta), &parsed) != nil {
		return ""
	}
	songs, ok := parsed["songs"]
	if !ok {
		return ""
	}
	arr, ok := songs.([]any)
	if !ok {
		return ""
	}
	var titles []string
	for i, s := range arr {
		if i >= 5 {
			break
		}
		if m, ok := s.(map[string]any); ok {
			if t := getStr(m, "title"); t != "" {
				titles = append(titles, t)
			}
		}
	}
	return strings.Join(titles, ", ")
}

func getStr(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
