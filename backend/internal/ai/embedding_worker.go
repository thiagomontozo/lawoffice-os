package ai

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/repository"
)

type EmbeddingWorker struct {
	store      *repository.Store
	embedder   Embedder
	workerID   string
	logger     *slog.Logger
	pollPeriod time.Duration
}

func NewEmbeddingWorker(store *repository.Store, embedder Embedder, logger *slog.Logger) *EmbeddingWorker {
	return &EmbeddingWorker{store: store, embedder: embedder, workerID: uuid.NewString(), logger: logger, pollPeriod: 10 * time.Second}
}

func (worker *EmbeddingWorker) Run(ctx context.Context) {
	worker.process(ctx)
	ticker := time.NewTicker(worker.pollPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			worker.logger.Info("embedding worker stopped")
			return
		case <-ticker.C:
			worker.process(ctx)
		}
	}
}

func (worker *EmbeddingWorker) process(ctx context.Context) {
	workContext, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	items, err := worker.store.ClaimEmbeddingChunks(workContext, worker.workerID, worker.embedder.EmbeddingModel(), 32)
	if err != nil {
		if ctx.Err() == nil {
			worker.logger.Error("embedding claim failed", "error", err)
		}
		return
	}
	if len(items) == 0 {
		return
	}
	input := make([]string, len(items))
	for index, item := range items {
		input[index] = item.Content
	}
	embeddings, err := worker.embedder.Embed(workContext, input)
	if err != nil || len(embeddings) != len(items) {
		for _, item := range items {
			_ = worker.store.FailEmbeddingChunk(workContext, worker.workerID, item, "PROVIDER_FAILED")
		}
		if ctx.Err() == nil {
			worker.logger.Warn("embedding batch failed", "count", len(items), "error_type", fmt.Sprintf("%T", err))
		}
		return
	}
	for index, item := range items {
		if completeErr := worker.store.CompleteEmbeddingChunk(workContext, worker.workerID, item.ID, item.FirmID, worker.embedder.EmbeddingModel(), embeddings[index]); completeErr != nil {
			_ = worker.store.FailEmbeddingChunk(workContext, worker.workerID, item, "PERSIST_FAILED")
			worker.logger.Warn("embedding persistence failed", "chunk_id", item.ID, "error_type", fmt.Sprintf("%T", completeErr))
		}
	}
}
