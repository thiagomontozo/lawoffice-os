package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/thiagomontozo/lawoffice-os/backend/internal/config"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/database"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/http/handlers"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/http/router"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/realtime"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/repository"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/scheduler"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/service"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("configuration rejected", "error", err)
		os.Exit(1)
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	db, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	if err = database.Migrate(ctx, db, cfg.MigrationsDir); err != nil {
		logger.Error("migration failed", "error", err)
		os.Exit(1)
	}
	objects, err := storage.NewLocal(cfg.StoragePath)
	if err != nil {
		logger.Error("storage initialization failed", "error", err)
		os.Exit(1)
	}
	store := repository.New(db)
	services := service.New(store, objects, cfg.MaxUpload)
	hub := realtime.New()
	if err = realtime.StartPostgres(ctx, db, hub, logger); err != nil {
		logger.Warn("database realtime unavailable; using local delivery", "error", err)
	}
	handler := handlers.New(store, services, objects, db, cfg, logger, hub)
	server := &http.Server{Addr: ":" + cfg.Port, Handler: router.New(handler, store, cfg), ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 60 * time.Second, IdleTimeout: 90 * time.Second}
	go scheduler.New(store, logger).Run(ctx)
	done := make(chan error, 1)
	go func() {
		logger.Info("LawOffice OS API started", "port", cfg.Port, "environment", cfg.Environment)
		done <- server.ListenAndServe()
	}()
	select {
	case err = <-done:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server stopped unexpectedly", "error", err)
		}
	case <-ctx.Done():
		logger.Info("shutdown requested")
	}
	hub.Close()
	shutdown, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err = server.Shutdown(shutdown); err != nil {
		logger.Error("graceful shutdown timed out", "error", err)
	}
	logger.Info("LawOffice OS API stopped")
}
