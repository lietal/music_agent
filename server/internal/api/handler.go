package api

import (
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/music-agent/music-agent/internal/agent"
	"github.com/music-agent/music-agent/internal/event"
)

type Handler struct {
	bus       *event.Bus
	jwtSecret []byte
	db        *pgxpool.Pool
	agent     *agent.AgentLoop
	activeRuns map[string]string
	mu        sync.RWMutex
}

func NewHandler(bus *event.Bus, jwtSecret []byte, db *pgxpool.Pool) *Handler {
	return &Handler{
		bus:        bus,
		jwtSecret:  jwtSecret,
		db:         db,
		activeRuns: make(map[string]string),
	}
}

func (h *Handler) SetAgent(a *agent.AgentLoop) {
	h.agent = a
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

func (h *Handler) JWTSecret() []byte {
	return h.jwtSecret
}

func (h *Handler) Bus() *event.Bus {
	return h.bus
}

func (h *Handler) DB() *pgxpool.Pool {
	return h.db
}
