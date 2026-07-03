package api

import (
	"context"
	"log/slog"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/music-agent/music-agent/internal/auth"
)

type contextHandler struct {
	handler slog.Handler
}

func (h *contextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h *contextHandler) Handle(ctx context.Context, r slog.Record) error {
	if reqID := middleware.GetReqID(ctx); reqID != "" {
		r.AddAttrs(slog.String("requestID", reqID))
	}
	if user := auth.UserFromContext(ctx); user != nil {
		r.AddAttrs(slog.String("userID", user.UserID))
	}
	if runID := runIDFromContext(ctx); runID != "" {
		r.AddAttrs(slog.String("runID", runID))
	}
	if convID := convIDFromContext(ctx); convID != "" {
		r.AddAttrs(slog.String("convID", convID))
	}
	return h.handler.Handle(ctx, r)
}

func (h *contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &contextHandler{handler: h.handler.WithAttrs(attrs)}
}

func (h *contextHandler) WithGroup(name string) slog.Handler {
	return &contextHandler{handler: h.handler.WithGroup(name)}
}

func NewContextHandler(handler slog.Handler) slog.Handler {
	return &contextHandler{handler: handler}
}
