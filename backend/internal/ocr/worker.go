package ocr

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/domain"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/repository"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/storage"
)

type Worker struct {
	store      *repository.Store
	storage    storage.ObjectStorage
	provider   Provider
	workerID   string
	logger     *slog.Logger
	pollPeriod time.Duration
}

func NewWorker(store *repository.Store, objects storage.ObjectStorage, provider Provider, logger *slog.Logger) *Worker {
	return &Worker{store: store, storage: objects, provider: provider, workerID: uuid.NewString(), logger: logger, pollPeriod: 10 * time.Second}
}

func (worker *Worker) Run(ctx context.Context) {
	worker.process(ctx)
	ticker := time.NewTicker(worker.pollPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			worker.logger.Info("document extraction worker stopped")
			return
		case <-ticker.C:
			worker.process(ctx)
		}
	}
}

func (worker *Worker) process(ctx context.Context) {
	workContext, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	items, err := worker.store.ClaimDocumentExtractions(workContext, worker.workerID, 3)
	if err != nil {
		if ctx.Err() == nil {
			worker.logger.Error("document extraction claim failed", "error", err)
		}
		return
	}
	for _, item := range items {
		if err = worker.extract(workContext, item); err != nil && ctx.Err() == nil {
			worker.logger.Warn("document extraction failed", "extraction_id", item.ID, "attempt", item.Attempts, "error_type", fmt.Sprintf("%T", err))
		}
	}
}

func (worker *Worker) extract(ctx context.Context, item domain.ClaimedDocumentExtraction) error {
	reader, err := worker.storage.Open(ctx, item.StorageKey)
	if err != nil {
		_ = worker.store.FailDocumentExtraction(ctx, worker.workerID, item, "STORAGE_OPEN_FAILED", false)
		return err
	}
	result, extractionErr := worker.provider.Extract(ctx, Document{Reader: reader, MimeType: item.MimeType, FileName: item.OriginalFileName})
	closeErr := reader.Close()
	if extractionErr != nil {
		unsupported := errors.Is(extractionErr, ErrUnsupported)
		code := "PROVIDER_FAILED"
		if unsupported {
			code = "UNSUPPORTED_TYPE"
		}
		if failErr := worker.store.FailDocumentExtraction(ctx, worker.workerID, item, code, unsupported); failErr != nil {
			return errors.Join(extractionErr, failErr)
		}
		return extractionErr
	}
	if closeErr != nil {
		_ = worker.store.FailDocumentExtraction(ctx, worker.workerID, item, "STORAGE_CLOSE_FAILED", false)
		return closeErr
	}
	pages := make([]domain.DocumentExtractionPage, 0, len(result.Pages))
	for _, page := range result.Pages {
		pages = append(pages, domain.DocumentExtractionPage{PageNumber: page.Number, Content: page.Text, Confidence: page.Confidence})
	}
	return worker.store.CompleteDocumentExtraction(ctx, worker.workerID, item, result.Provider, result.Language, pages)
}
