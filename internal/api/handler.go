package api

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/music-agent/music-agent/internal/event"
)

type Handler struct {
	bus       *event.Bus
	jwtSecret []byte
	db        *pgxpool.Pool
}

func NewHandler(bus *event.Bus, jwtSecret []byte, db *pgxpool.Pool) *Handler {
	return &Handler{
		bus:       bus,
		jwtSecret: jwtSecret,
		db:        db,
	}
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
