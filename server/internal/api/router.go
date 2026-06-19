package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(h *Handler) chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)

	r.Get("/health", healthHandler)

	r.Post("/api/auth/register", h.authRegisterHandler)
	r.Post("/api/auth/login", h.authLoginHandler)

	r.Group(func(r chi.Router) {
		r.Use(JWTAuthMiddleware(h.jwtSecret))

		r.Get("/api/auth/me", h.authMeHandler)

		r.Post("/api/conversations", createConversationHandler)
		r.Get("/api/conversations", listConversationsHandler)
		r.Get("/api/conversations/{id}", getConversationHandler)

		r.Post("/api/chat", h.createChatHandler)
		r.Get("/api/chat/{runId}/events", h.chatEventsHandler)
	})

	return r
}

func SetupPlayerRoutes(r chi.Router, ph *PlayerHandler, sh *StreamHandler, lh *LoginHandler) {
	r.Get("/api/player/url/*", ph.HandleGetPlayURL)
	r.Get("/api/player/stream/*", sh.HandleStream)
	r.Get("/api/player/lyrics/*", ph.HandleGetLyrics)
	r.Post("/api/qqmusic/login/qrcode", lh.HandleGetQRCode)
	r.Get("/api/qqmusic/login/status", lh.HandleGetStatus)
	r.Get("/api/qqmusic/login/status/*", lh.HandleCheckQRStatus)
}
