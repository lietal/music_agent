package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/music-agent/music-agent/internal/api"
	"github.com/music-agent/music-agent/internal/auth"
	"github.com/music-agent/music-agent/internal/event"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	jwtSecret := []byte(os.Getenv("JWT_SECRET"))
	if len(jwtSecret) == 0 {
		jwtSecret = []byte("dev-secret-change-in-production")
	}

	var provider auth.OAuthProvider
	if appID := os.Getenv("WECHAT_APP_ID"); appID != "" {
		provider = auth.NewWeChatProvider(
			appID,
			os.Getenv("WECHAT_APP_SECRET"),
			os.Getenv("WECHAT_REDIRECT_URI"),
		)
	} else {
		provider = auth.NewMockProvider()
		logger.Info("using mock auth provider (no WECHAT_APP_ID set)")
	}

	bus := event.NewBus()
	handler := api.NewHandler(bus, jwtSecret, provider)
	r := api.NewRouter(handler)

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info("server starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	<-quit
	logger.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", "error", err)
	}

	logger.Info("server stopped")
}
