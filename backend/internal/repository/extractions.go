package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/domain"
)

func (s *Store) ClaimDocumentExtractions(ctx context.Context, workerID string, limit int) ([]domain.ClaimedDocumentExtraction, error) {
	if limit < 1 || limit > 20 {
		return nil, ErrInvalid
	}
	rows, err := s.Pool.Query(ctx, `WITH candidates AS (
        SELECT e.id FROM document_extractions e
        JOIN documents d ON d.id=e.document_id AND d.firm_id=e.firm_id
        WHERE d.deleted_at IS NULL AND ((e.status='pending' AND e.available_at<=now()) OR (e.status='processing' AND e.locked_at<now()-interval '10 minutes'))
        ORDER BY e.available_at,e.created_at
        FOR UPDATE OF e SKIP LOCKED LIMIT $1
    )
    UPDATE document_extractions e SET status='processing',attempts=e.attempts+1,locked_at=now(),locked_by=$2,started_at=COALESCE(started_at,now()),updated_at=now()
    FROM candidates c,document_versions v
    WHERE e.id=c.id AND v.id=e.document_version_id AND v.firm_id=e.firm_id
    RETURNING e.id,e.firm_id,e.document_id,e.document_version_id,v.storage_key,v.mime_type,v.original_file_name,e.attempts,e.max_attempts`, limit, workerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.ClaimedDocumentExtraction{}
	for rows.Next() {
		var item domain.ClaimedDocumentExtraction
		if err = rows.Scan(&item.ID, &item.FirmID, &item.DocumentID, &item.DocumentVersionID, &item.StorageKey, &item.MimeType, &item.OriginalFileName, &item.Attempts, &item.MaxAttempts); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) CompleteDocumentExtraction(ctx context.Context, workerID string, item domain.ClaimedDocumentExtraction, provider, language string, pages []domain.DocumentExtractionPage) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `DELETE FROM document_extraction_pages WHERE extraction_id=$1 AND firm_id=$2`, item.ID, item.FirmID); err != nil {
		return err
	}
	confidenceTotal := 0.0
	confidenceCount := 0
	for _, page := range pages {
		if _, err = tx.Exec(ctx, `INSERT INTO document_extraction_pages(extraction_id,firm_id,page_number,content,confidence) VALUES($1,$2,$3,$4,$5)`, item.ID, item.FirmID, page.PageNumber, page.Content, page.Confidence); err != nil {
			return mapError(err)
		}
		if page.Confidence != nil {
			confidenceTotal += *page.Confidence
			confidenceCount++
		}
	}
	var average *float64
	if confidenceCount > 0 {
		value := confidenceTotal / float64(confidenceCount)
		average = &value
	}
	result, err := tx.Exec(ctx, `UPDATE document_extractions SET status='succeeded',provider=$4,language=$5,page_count=$6,average_confidence=$7,error_code=NULL,locked_at=NULL,locked_by=NULL,completed_at=now(),updated_at=now() WHERE id=$1 AND firm_id=$2 AND locked_by=$3`, item.ID, item.FirmID, workerID, provider, language, len(pages), average)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrConflict
	}
	return tx.Commit(ctx)
}

func (s *Store) FailDocumentExtraction(ctx context.Context, workerID string, item domain.ClaimedDocumentExtraction, errorCode string, unsupported bool) error {
	status := "pending"
	if unsupported {
		status = "unsupported"
	} else if item.Attempts >= item.MaxAttempts {
		status = "failed"
	}
	exponent := item.Attempts
	if exponent > 6 {
		exponent = 6
	}
	delay := time.Duration(1<<exponent) * 30 * time.Second
	_, err := s.Pool.Exec(ctx, `UPDATE document_extractions SET status=$4,available_at=now()+($5 * interval '1 second'),error_code=$6,locked_at=NULL,locked_by=NULL,completed_at=CASE WHEN $4 IN('failed','unsupported') THEN now() ELSE NULL END,updated_at=now() WHERE id=$1 AND firm_id=$2 AND locked_by=$3`, item.ID, item.FirmID, workerID, status, delay.Seconds(), errorCode)
	return err
}

func (s *Store) DocumentExtraction(ctx context.Context, firmID, documentID string, pageSize, offset int) (domain.DocumentExtraction, error) {
	var extraction domain.DocumentExtraction
	err := s.Pool.QueryRow(ctx, `SELECT e.id,e.document_id,e.document_version_id,e.status,e.provider,e.language,e.page_count,e.average_confidence,e.attempts,e.error_code,e.created_at,e.started_at,e.completed_at
        FROM documents d JOIN document_extractions e ON e.document_version_id=d.current_version_id AND e.firm_id=d.firm_id
        WHERE d.firm_id=$1 AND d.id=$2 AND d.deleted_at IS NULL`, firmID, documentID).Scan(&extraction.ID, &extraction.DocumentID, &extraction.DocumentVersionID, &extraction.Status, &extraction.Provider, &extraction.Language, &extraction.PageCount, &extraction.AverageConfidence, &extraction.Attempts, &extraction.ErrorCode, &extraction.CreatedAt, &extraction.StartedAt, &extraction.CompletedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return extraction, ErrNotFound
	}
	if err != nil {
		return extraction, err
	}
	extraction.Pages = []domain.DocumentExtractionPage{}
	if extraction.Status != "succeeded" {
		return extraction, nil
	}
	rows, err := s.Pool.Query(ctx, `SELECT page_number,content,confidence FROM document_extraction_pages WHERE extraction_id=$1 AND firm_id=$2 ORDER BY page_number LIMIT $3 OFFSET $4`, extraction.ID, firmID, pageSize, offset)
	if err != nil {
		return extraction, err
	}
	defer rows.Close()
	for rows.Next() {
		var page domain.DocumentExtractionPage
		if err = rows.Scan(&page.PageNumber, &page.Content, &page.Confidence); err != nil {
			return extraction, err
		}
		extraction.Pages = append(extraction.Pages, page)
	}
	return extraction, rows.Err()
}

func (s *Store) RequeueDocumentExtraction(ctx context.Context, firmID, documentID string) error {
	result, err := s.Pool.Exec(ctx, `UPDATE document_extractions e SET status='pending',attempts=0,available_at=now(),locked_at=NULL,locked_by=NULL,error_code=NULL,started_at=NULL,completed_at=NULL,updated_at=now() FROM documents d WHERE d.id=$2 AND d.firm_id=$1 AND d.deleted_at IS NULL AND e.document_version_id=d.current_version_id AND e.firm_id=d.firm_id`, firmID, documentID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
