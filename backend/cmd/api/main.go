package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/thiagomontozo/lawoffice-os/backend/internal/config"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/database"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/http/handlers"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/http/router"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/jobs"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/mailer"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/observability"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/ocr"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/realtime"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/repository"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/scanner"
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
	var objects storage.ObjectStorage
	if cfg.StorageDriver == "s3" {
		objects, err = storage.NewS3(ctx, storage.S3Config{Endpoint: cfg.S3Endpoint, Bucket: cfg.S3Bucket, AccessKey: cfg.S3AccessKey, SecretKey: cfg.S3SecretKey, Region: cfg.S3Region, UseTLS: cfg.S3UseTLS, CreateBucket: cfg.S3CreateBucket})
	} else {
		objects, err = storage.NewLocal(cfg.StoragePath)
	}
	if err != nil {
		logger.Error("storage initialization failed", "driver", cfg.StorageDriver, "error", err)
		os.Exit(1)
	}
	var uploadScanner scanner.Scanner = scanner.Disabled{}
	if cfg.UploadScanMode == "required" {
		uploadScanner = scanner.NewClamAV(cfg.ClamAVAddress)
	}
	store := repository.New(db)
	services := service.New(store, objects, uploadScanner, cfg.MaxUpload)
	var extractionWorker *ocr.Worker
	if cfg.OCRMode != "off" {
		var provider ocr.Provider = ocr.Builtin{MaxBytes: cfg.OCRMaxInput}
		if cfg.OCRMode == "http" {
			provider = &ocr.HTTPProvider{Endpoint: cfg.OCREndpoint, Token: cfg.OCRToken, Language: cfg.OCRLanguage, Client: &http.Client{Timeout: cfg.OCRTimeout}, MaxBytes: cfg.OCRMaxInput, MaxPages: cfg.OCRMaxPages, MaxCharacters: cfg.OCRMaxCharacters}
		}
		extractionWorker = ocr.NewWorker(store, objects, provider, logger)
	}
	var emailSender mailer.Sender
	if cfg.SMTPMode == "enabled" {
		emailSender, err = mailer.NewSMTP(mailer.SMTPConfig{Address: cfg.SMTPAddress, Username: cfg.SMTPUsername, Password: cfg.SMTPPassword, From: cfg.SMTPFrom, FromName: cfg.SMTPFromName, RequireTLS: cfg.SMTPRequireTLS})
		if err != nil {
			logger.Error("email delivery initialization failed", "error", err)
			os.Exit(1)
		}
	}
	jobQueue, err := jobs.New(db, cfg.JobEncryptionSecret, emailSender, logger)
	if err != nil {
		logger.Error("outbound job queue initialization failed", "error", err)
		os.Exit(1)
	}
	hub := realtime.New()
	if err = realtime.StartPostgres(ctx, db, hub, logger); err != nil {
		logger.Warn("database realtime unavailable; using local delivery", "error", err)
	}
	handler := handlers.New(store, services, objects, db, cfg, logger, hub, jobQueue)
	metrics := observability.NewMetrics()
	server := &http.Server{Addr: ":" + cfg.Port, Handler: router.New(handler, store, cfg, metrics), ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 60 * time.Second, IdleTimeout: 90 * time.Second}
	var background sync.WaitGroup
	background.Add(2)
	if extractionWorker != nil {
		background.Add(1)
		go func() {
			defer background.Done()
			extractionWorker.Run(ctx)
		}()
	}
	go func() {
		defer background.Done()
		scheduler.New(store, jobQueue, logger).Run(ctx)
	}()
	go func() {
		defer background.Done()
		jobQueue.Run(ctx)
	}()
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
	stop()
	hub.Close()
	shutdown, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err = server.Shutdown(shutdown); err != nil {
		logger.Error("graceful shutdown timed out", "error", err)
	}
	backgroundDone := make(chan struct{})
	go func() {
		background.Wait()
		close(backgroundDone)
	}()
	select {
	case <-backgroundDone:
	case <-shutdown.Done():
		logger.Error("background workers did not stop before shutdown deadline")
	}
	logger.Info("LawOffice OS API stopped")
}
