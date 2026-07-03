package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/music-agent/music-agent/internal/agent"
	"github.com/music-agent/music-agent/internal/api"
	"github.com/music-agent/music-agent/internal/config"
	"github.com/music-agent/music-agent/internal/crypto"
	"github.com/music-agent/music-agent/internal/db"
	"github.com/music-agent/music-agent/internal/event"
	"github.com/music-agent/music-agent/internal/llm"
	"github.com/music-agent/music-agent/internal/tme"
	"github.com/music-agent/music-agent/internal/tool"
)

func main() {
	logger := slog.New(api.NewContextHandler(slog.NewJSONHandler(os.Stdout, nil)))

	cfg, err := config.Load("config.toml")
	if err != nil {
		logger.Warn("config load failed, using defaults", "error", err)
		cfg = &config.Config{
			Server: config.ServerConfig{Host: "0.0.0.0", Port: 8080},
			LLM:    config.LLMConfig{Provider: "deepseek", Model: "deepseek-chat"},
		}
	}

	jwtSecret := []byte(os.Getenv("JWT_SECRET"))
	if len(jwtSecret) == 0 {
		jwtSecret = []byte(cfg.Auth.JWTSecret)
	}
	if len(jwtSecret) == 0 {
		jwtSecret = []byte("dev-secret-change-in-production")
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = cfg.Database.URL
	}
	if databaseURL == "" {
		databaseURL = "postgres://music_agent:music_agent@127.0.0.1:5432/music_agent?sslmode=disable"
	}

	ctx := context.Background()
	pool, err := db.NewPool(ctx, databaseURL, db.DefaultPoolConfig())
	if err != nil {
		logger.Warn("database not available, using in-memory storage", "error", err)
	} else {
		defer pool.Close()
		if err := db.RunMigrations(ctx, databaseURL); err != nil {
			logger.Error("failed to run migrations", "error", err)
			os.Exit(1)
		}
	}

	apiKey := os.Getenv("LLM_API_KEY")
	if apiKey == "" {
		apiKey = cfg.LLM.APIKey
	}
	baseURL := os.Getenv("LLM_BASE_URL")
	if baseURL == "" {
		baseURL = cfg.LLM.BaseURL
	}
	model := os.Getenv("LLM_MODEL")
	if model == "" {
		model = cfg.LLM.Model
	}

	bus := event.NewBus()
	handler := api.NewHandler(bus, jwtSecret, pool, logger)
	handler.SetMaxSteps(5)

	if apiKey != "" && baseURL != "" && model != "" {
		llmClient := llm.NewOpenAI(baseURL, apiKey, nil)

		tmeSearch := tool.NewTMESearchSongs()
		checkCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		useReal := tmeSearch.IsAvailable(checkCtx)
		cancel()

		var searchTool tool.Tool
		if useReal {
			searchTool = tmeSearch
			logger.Info("TME direct API connected")
		} else {
			searchTool = tool.NewMockSearchSongs()
			logger.Warn("TME API unavailable, using mock search")
		}

		tools := map[string]tool.Tool{
			"search_songs":    searchTool,
			"recommend_songs": tool.NewMockRecommendSongs(),
		}
		if pool != nil {
			for name, t := range tool.NewPlaylistTools(pool) {
				tools[name] = t
			}
		}

		executor := agent.NewDefaultExecutor()
		prompts := agent.DefaultPrompts()

		turnPlanner := agent.NewLLMTurnPlanner(llmClient, model, prompts.IntentRouter)
		pipeline := agent.NewAgentPipeline(turnPlanner, llmClient, model, prompts, tools, executor, 5)
		handler.SetAgent(pipeline)
		logger.Info("agent initialized with pipeline", "model", model)
	} else {
		logger.Warn("LLM not configured, using mock agent")
	}

	r := api.NewRouter(handler)

	// Player and QQ Music login routes
	tmeClient := tme.NewClient()
	encryptor := crypto.NewAES(string(jwtSecret))
	credStore := tme.NewCredentialStoreWithPool(pool, encryptor)
	credStore.LoadCredentials()
	handler.SetCredentialStore(credStore)
	handler.SetTMEClient(tmeClient)
	if credStore.IsLoggedIn() {
		mid, mk := credStore.Get()
		tmeClient.SetCredential(mid, mk)
	}
	playerH := api.NewPlayerHandler(tmeClient, credStore)
	playerH.SetLogger(logger)
	streamH := api.NewStreamHandler(tmeClient)
	streamH.SetLogger(logger)
	loginH := api.NewLoginHandler(tmeClient, credStore, jwtSecret, pool)
	loginH.SetLogger(logger)
	api.SetupPlayerRoutes(r, playerH, streamH, loginH)

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 0,
		IdleTimeout:  120 * time.Second,
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

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server forced to shutdown", "error", err)
	}

	logger.Info("server stopped")
}
