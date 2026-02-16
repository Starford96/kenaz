// Package internal provides the main application initialization and runtime logic.
package internal

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"golang.org/x/sync/errgroup"

	"github.com/starford/kenaz/internal/api"
	"github.com/starford/kenaz/internal/index"
	"github.com/starford/kenaz/internal/sse"
	"github.com/starford/kenaz/internal/storage"
)

// Run starts the application with the given options.
func Run(ctx context.Context, opts ...Option) error {
	app := &application{}

	for _, opt := range opts {
		opt(app)
	}

	if app.config == nil {
		return fmt.Errorf("config is required")
	}

	cfg := app.config

	// Initialize structured JSON logger.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.App.LogLevel,
	}))
	slog.SetDefault(logger)

	logger.Info("Configuration loaded",
		slog.String("http_address", cfg.App.HTTP.Address()),
		slog.String("vault_path", cfg.Vault.Path),
		slog.String("sqlite_path", cfg.SQLite.Path),
		slog.String("log_level", cfg.App.LogLevel.String()))

	// Ensure vault directory exists.
	if err := os.MkdirAll(cfg.Vault.Path, 0o755); err != nil {
		return fmt.Errorf("create vault dir: %w", err)
	}

	// Initialize storage.
	store, err := storage.NewFS(cfg.Vault.Path)
	if err != nil {
		return fmt.Errorf("init storage: %w", err)
	}

	// Initialize SQLite index.
	db, err := index.Open(cfg.SQLite.Path)
	if err != nil {
		return fmt.Errorf("init index: %w", err)
	}
	defer db.Close()

	// Run initial sync.
	if err := index.Sync(db, store, logger); err != nil {
		logger.Warn("initial sync failed", slog.String("error", err.Error()))
	}

	// SSE broker.
	broker := sse.NewBroker(2 * time.Second)

	// Build API service and router.
	svc := api.NewService(store, db)
	apiRouter := api.NewRouter(svc, cfg.Auth.Token)

	// Build chi router.
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Health check endpoints (unauthenticated).
	r.Get("/health/live", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	r.Get("/health/ready", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// Mount API routes under /api.
	r.Mount("/api", apiRouter)

	// SSE endpoint.
	r.Get("/api/events", broker.ServeHTTP)

	httpServer := &http.Server{
		Addr:    cfg.App.HTTP.Address(),
		Handler: r,
	}

	logger.Info("Server starting...", slog.String("http_address", cfg.App.HTTP.Address()))

	g, gCtx := errgroup.WithContext(ctx)

	// Start file watcher with SSE callback.
	g.Go(func() error {
		index.Watch(gCtx, db, store, cfg.Vault.Path, logger, func(kind, path string) {
			broker.PublishNoteEvent(kind, path)
		})
		return nil
	})

	// Start HTTP server.
	g.Go(func() error {
		logger.Info("Starting HTTP server", slog.String("address", cfg.App.HTTP.Address()))
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("HTTP server error: %w", err)
		}
		return nil
	})

	// Handle shutdown signals.
	g.Go(func() error {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		select {
		case sig := <-quit:
			logger.Info("Received shutdown signal", slog.String("signal", sig.String()))
		case <-gCtx.Done():
			logger.Info("Context cancelled, initiating shutdown")
		}

		logger.Info("Shutting down server...")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			logger.Error("HTTP server shutdown error", slog.String("error", err.Error()))
		}

		return nil
	})

	if err := g.Wait(); err != nil {
		logger.Error("Application error", slog.String("error", err.Error()))
		return err
	}

	logger.Info("Server stopped successfully")
	return nil
}
