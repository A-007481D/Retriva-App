// Command retriva is the entry point for the Retriva server.
// It loads configuration, initializes all services, and starts the HTTP server.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/A-007481D/retriva/server/internal/api"
	"github.com/A-007481D/retriva/server/internal/config"
	"github.com/A-007481D/retriva/server/internal/database"
	"github.com/A-007481D/retriva/server/internal/downloader"
	"github.com/A-007481D/retriva/server/internal/history"
	"github.com/A-007481D/retriva/server/internal/jobs"
	"github.com/A-007481D/retriva/server/internal/media"
	"github.com/A-007481D/retriva/server/internal/resolver"
	"github.com/A-007481D/retriva/server/internal/resolver/direct"
	"github.com/A-007481D/retriva/server/internal/storage/filesystem"
)

// version is injected at build time via:
//
//	go build -ldflags "-X main.version=v0.1.0"
var version = "dev"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "retriva: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	logger := newLogger(cfg.LogLevel)

	db, err := database.Open(cfg.Database, logger)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	store, err := filesystem.New(cfg.DataDir)
	if err != nil {
		return fmt.Errorf("init storage: %w", err)
	}

	// Initialize repositories
	jobsRepo := jobs.NewSQLiteRepository(db.DB)
	mediaRepo := media.NewSQLiteRepository(db.DB)
	historyRepo := history.NewSQLiteRepository(db.DB)

	// Initialize resolver and downloader
	resRegistry := resolver.NewRegistry(direct.New())
	dl := downloader.NewHTTPDownloader(store, 30*time.Second, 1024*1024*1024, 5) // 1GB max

	// Initialize executor and worker pool
	exec := jobs.NewExecutor(resRegistry, dl, jobsRepo, mediaRepo, historyRepo, logger)
	pool := jobs.NewWorkerPool(cfg.Workers, 1000, exec, logger)

	// Propagate version to API handler.
	api.Version = version

	handler := api.New(logger, db, store, jobsRepo, mediaRepo, historyRepo, pool, cfg.AuthToken)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      handler,
		ReadTimeout:  cfg.RequestTimeout,
		WriteTimeout: cfg.RequestTimeout + 10*time.Second,
		IdleTimeout:  120 * time.Second,
	}

	logger.Info("retriva starting",
		slog.String("version", version),
		slog.String("addr", srv.Addr),
		slog.String("data_dir", cfg.DataDir),
		slog.Int("workers", cfg.Workers),
		slog.Bool("auth_enabled", cfg.AuthToken != ""),
	)

	pool.Start()
	logger.Info("worker pool started", slog.Int("workers", cfg.Workers))

	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return fmt.Errorf("server error: %w", err)
	case sig := <-quit:
		logger.Info("shutdown signal received", slog.String("signal", sig.String()))

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			return fmt.Errorf("graceful shutdown failed: %w", err)
		}
		
		pool.Stop()
		logger.Info("worker pool stopped")

		logger.Info("shutdown complete")
		return nil
	}
}

func newLogger(level string) *slog.Logger {
	var l slog.Level
	switch level {
	case "debug":
		l = slog.LevelDebug
	case "warn":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	default:
		l = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     l,
		AddSource: level == "debug",
	}))
}
