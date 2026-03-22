package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/naoki-watanabe/casual-talker/backend/internal/config"
	"github.com/naoki-watanabe/casual-talker/backend/internal/handler"
	"github.com/naoki-watanabe/casual-talker/backend/internal/middleware"
	oai "github.com/naoki-watanabe/casual-talker/backend/internal/openai"
	"github.com/naoki-watanabe/casual-talker/backend/internal/repository"
	"github.com/naoki-watanabe/casual-talker/backend/internal/service"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	// --- Database ---
	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to create database connection pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		slog.Error("failed to reach database", "error", err)
		os.Exit(1)
	}

	// --- Dependency wiring ---
	authRepo := repository.NewPgxAuthRepository(pool)
	authSvc := service.NewAuthService(authRepo, cfg.JWTSecret)
	authHandler := handler.NewAuthHandler(authSvc)

	sessionRepo := repository.NewPgxSessionRepository(pool)

	openaiClient := oai.NewClient(cfg.OpenAIAPIKey)
	sessionHandler := handler.NewSessionHandler(sessionRepo, authRepo, openaiClient)
	feedbackHandler := handler.NewFeedbackHandler(sessionRepo, openaiClient)
	speechHandler := handler.NewSpeechHandler(openaiClient)
	chatHandler := handler.NewChatHandler(openaiClient, sessionRepo)

	authMiddleware := middleware.Auth(middleware.AuthConfig{
		JWTSecret: []byte(cfg.JWTSecret),
	})

	// Rate limiter with a cancellable context tied to server lifetime.
	rlCtx, rlCancel := context.WithCancel(context.Background())
	defer rlCancel()
	rateLimiter := middleware.NewRateLimiter(rlCtx)

	// --- Router ---
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(rateLimiter.RateLimit)
	r.Use(chimiddleware.RequestLogger(&chimiddleware.DefaultLogFormatter{
		Logger:  slog.NewLogLogger(logger.Handler(), slog.LevelInfo),
		NoColor: true,
	}))
	r.Use(chimiddleware.Recoverer)

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Public routes
		r.Post("/auth/register", authHandler.Register)
		r.Post("/auth/login", authHandler.Login)
		r.Post("/auth/refresh", authHandler.Refresh)
		r.Post("/auth/logout", authHandler.Logout)
		r.Get("/health", handler.Health)

		// Protected routes (require a valid access token)
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)
			r.Get("/users/me", authHandler.Me)
			r.Get("/users/me/stats", sessionHandler.Stats)

			// Courses & themes
			r.Get("/courses", sessionHandler.ListCourses)
			r.Get("/courses/{courseID}/themes", sessionHandler.ListThemes)
			r.Get("/themes/{id}", sessionHandler.GetTheme)

			// Sessions
			r.Post("/sessions", sessionHandler.Create)
			r.Get("/sessions", sessionHandler.List)
			r.Get("/sessions/{id}", sessionHandler.Get)
			r.Put("/sessions/{id}/complete", sessionHandler.Complete)
			r.Get("/sessions/{id}/turns", sessionHandler.ListTurns)
			r.Get("/sessions/{id}/feedback", sessionHandler.GetFeedback)

			// Speech
			r.Post("/speech/stt", speechHandler.STT)
			r.Post("/speech/tts", speechHandler.TTS)

			// Chat (AI conversation)
			r.Post("/chat/stream", chatHandler.Stream)
			r.Post("/chat/hint", chatHandler.Hint)
			r.Post("/chat/interpret", chatHandler.Interpret)

			// Feedback generation
			r.Post("/feedback/generate", feedbackHandler.Generate)
		})
	})

	// Static file serving with SPA fallback
	r.Mount("/", staticHandler(cfg.StaticDir, logger))

	addr := fmt.Sprintf(":%s", cfg.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine so that graceful shutdown can proceed.
	serverErr := make(chan error, 1)
	go func() {
		slog.Info("server starting", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// Wait for an OS signal or a fatal server error.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		slog.Info("shutdown signal received", "signal", sig)
	case err := <-serverErr:
		slog.Error("server error", "error", err)
	}

	// Allow up to 30 seconds for in-flight requests to complete.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}

	slog.Info("server stopped")
}

// staticHandler serves files from staticDir and falls back to index.html for
// paths that do not match an existing file, enabling SPA client-side routing.
// filepath.Clean prevents path traversal attacks.
func staticHandler(staticDir string, logger *slog.Logger) http.Handler {
	absStatic, err := filepath.Abs(staticDir)
	if err != nil {
		absStatic = staticDir
	}
	indexPath := filepath.Join(absStatic, "index.html")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Clean and join to prevent path traversal (e.g. "../../etc/passwd").
		cleaned := filepath.Clean("/" + r.URL.Path)
		path := filepath.Join(absStatic, cleaned)

		info, err := os.Stat(path)
		if err != nil {
			// File not found — serve index.html for SPA client-side routing.
			http.ServeFile(w, r, indexPath)
			return
		}

		// Prevent directory listing.
		if info.IsDir() {
			dirIndex := filepath.Join(path, "index.html")
			if _, err := os.Stat(dirIndex); err != nil {
				http.ServeFile(w, r, indexPath)
				return
			}
		}

		http.ServeFile(w, r, path)
	})
}
