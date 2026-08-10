package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/domain"
)

const (
	chunkSizeRunes    = 1800
	chunkOverlapRunes = 250
)

func splitExtractionPage(content string) []string {
	runes := []rune(strings.TrimSpace(content))
	if len(runes) == 0 {
		return nil
	}
	chunks := make([]string, 0, len(runes)/chunkSizeRunes+1)
	for start := 0; start < len(runes); {
		end := start + chunkSizeRunes
		if end > len(runes) {
			end = len(runes)
		} else {
			minimum := end - 300
			for cursor := end; cursor > minimum; cursor-- {
				if runes[cursor-1] == '\n' || runes[cursor-1] == '.' || runes[cursor-1] == ';' {
					end = cursor
					break
				}
			}
		}
		chunk := strings.TrimSpace(string(runes[start:end]))
		if chunk != "" {
			chunks = append(chunks, chunk)
		}
		if end == len(runes) {
			break
		}
		next := end - chunkOverlapRunes
		if next <= start {
			next = end
		}
		start = next
	}
	return chunks
}

func replaceDocumentChunks(ctx context.Context, tx pgx.Tx, item domain.ClaimedDocumentExtraction, pages []domain.DocumentExtractionPage) error {
	if _, err := tx.Exec(ctx, `DELETE FROM document_chunks WHERE firm_id=$1 AND document_version_id=$2`, item.FirmID, item.DocumentVersionID); err != nil {
		return err
	}
	for _, page := range pages {
		for index, content := range splitExtractionPage(page.Content) {
			digest := sha256.Sum256([]byte(content))
			_, err := tx.Exec(ctx, `INSERT INTO document_chunks(
				firm_id,matter_id,document_id,document_version_id,extraction_id,page_number,chunk_index,content,content_hash,character_count
			) SELECT d.firm_id,d.matter_id,d.id,$3,$4,$5,$6,$7,$8,$9
			  FROM documents d WHERE d.firm_id=$1 AND d.id=$2 AND d.deleted_at IS NULL`,
				item.FirmID, item.DocumentID, item.DocumentVersionID, item.ID, page.PageNumber, index, content, hex.EncodeToString(digest[:]), len([]rune(content)))
			if err != nil {
				return mapError(err)
			}
		}
	}
	return nil
}

func (s *Store) MatterRetrievalCandidates(ctx context.Context, firmID, userID, matterID, query string, documentIDs []string, limit int) ([]domain.DocumentChunk, error) {
	if limit < 1 || limit > 300 {
		return nil, ErrInvalid
	}
	allowed, err := s.CanAccessMatter(ctx, firmID, userID, matterID, "read")
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrForbidden
	}
	rows, err := s.Pool.Query(ctx, `SELECT c.id,c.firm_id,c.matter_id,c.document_id,c.document_version_id,d.title,
		c.page_number,c.chunk_index,c.character_count,c.content,c.content_hash,
		CASE WHEN btrim($4)='' THEN 0 ELSE ts_rank_cd(c.search_vector,websearch_to_tsquery('portuguese',$4)) END,
		COALESCE(c.embedding,'null'::jsonb),COALESCE(c.embedding_model,'')
	FROM document_chunks c
	JOIN documents d ON d.id=c.document_id AND d.firm_id=c.firm_id
	WHERE c.firm_id=$1 AND c.matter_id=$2 AND d.deleted_at IS NULL AND d.current_version_id=c.document_version_id
	  AND (cardinality($3::uuid[])=0 OR c.document_id=ANY($3::uuid[]))
	ORDER BY CASE WHEN btrim($4)='' THEN 0 ELSE ts_rank_cd(c.search_vector,websearch_to_tsquery('portuguese',$4)) END DESC,
	         c.document_id,c.page_number,c.chunk_index
	LIMIT $5`, firmID, matterID, documentIDs, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.DocumentChunk{}
	for rows.Next() {
		var item domain.DocumentChunk
		var embeddingJSON []byte
		if err = rows.Scan(&item.ID, &item.FirmID, &item.MatterID, &item.DocumentID, &item.DocumentVersionID, &item.DocumentTitle,
			&item.PageNumber, &item.ChunkIndex, &item.CharacterCount, &item.Content, &item.ContentHash, &item.KeywordScore, &embeddingJSON, &item.EmbeddingModel); err != nil {
			return nil, err
		}
		if string(embeddingJSON) != "null" {
			if err = json.Unmarshal(embeddingJSON, &item.Embedding); err != nil {
				return nil, err
			}
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) ClaimEmbeddingChunks(ctx context.Context, workerID, model string, limit int) ([]domain.ClaimedEmbeddingChunk, error) {
	if limit < 1 || limit > 100 {
		return nil, ErrInvalid
	}
	rows, err := s.Pool.Query(ctx, `WITH candidates AS (
		SELECT c.id FROM document_chunks c
		JOIN documents d ON d.id=c.document_id AND d.firm_id=c.firm_id
		WHERE d.deleted_at IS NULL AND d.current_version_id=c.document_version_id
		  AND ((c.embedding_status='pending' AND c.embedding_available_at<=now())
		    OR (c.embedding_status='processing' AND c.embedding_locked_at<now()-interval '10 minutes')
		    OR (c.embedding_status='succeeded' AND c.embedding_model IS DISTINCT FROM $3))
		ORDER BY c.embedding_available_at,c.created_at
		FOR UPDATE OF c SKIP LOCKED LIMIT $1
	)
	UPDATE document_chunks c SET embedding_status='processing',
		embedding_attempts=CASE WHEN c.embedding_model IS DISTINCT FROM $3 THEN 1 ELSE c.embedding_attempts+1 END,
		embedding_locked_at=now(),embedding_locked_by=$2,updated_at=now()
	FROM candidates q WHERE c.id=q.id
	RETURNING c.id,c.firm_id,c.content,c.embedding_attempts`, limit, workerID, model)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.ClaimedEmbeddingChunk{}
	for rows.Next() {
		var item domain.ClaimedEmbeddingChunk
		if err = rows.Scan(&item.ID, &item.FirmID, &item.Content, &item.Attempts); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) CompleteEmbeddingChunk(ctx context.Context, workerID, id, firmID, model string, embedding []float32) error {
	encoded, err := json.Marshal(embedding)
	if err != nil {
		return err
	}
	result, err := s.Pool.Exec(ctx, `UPDATE document_chunks SET embedding=$5::jsonb,embedding_model=$4,
		embedding_status='succeeded',embedding_error_code=NULL,embedding_locked_at=NULL,embedding_locked_by=NULL,updated_at=now()
		WHERE id=$1 AND firm_id=$2 AND embedding_locked_by=$3`, id, firmID, workerID, model, encoded)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrConflict
	}
	return nil
}

func (s *Store) FailEmbeddingChunk(ctx context.Context, workerID string, item domain.ClaimedEmbeddingChunk, code string) error {
	status := "pending"
	if item.Attempts >= 5 {
		status = "failed"
	}
	exponent := item.Attempts
	if exponent > 6 {
		exponent = 6
	}
	delay := time.Duration(1<<exponent) * time.Minute
	_, err := s.Pool.Exec(ctx, `UPDATE document_chunks SET embedding_status=$4,
		embedding_available_at=now()+($5 * interval '1 second'),embedding_error_code=$6,
		embedding_locked_at=NULL,embedding_locked_by=NULL,updated_at=now()
		WHERE id=$1 AND firm_id=$2 AND embedding_locked_by=$3`, item.ID, item.FirmID, workerID, status, delay.Seconds(), code)
	return err
}

func (s *Store) SaveAIFeedback(ctx context.Context, firmID, userID, matterID, responseID, rating string, reason *string) error {
	result, err := s.Pool.Exec(ctx, `INSERT INTO ai_feedback(firm_id,user_id,matter_id,response_id,rating,reason)
		SELECT $1,$2,r.matter_id,r.id,$5,$6 FROM ai_responses r
		JOIN matters m ON m.id=r.matter_id AND m.firm_id=r.firm_id
		WHERE r.id=$4 AND r.firm_id=$1 AND r.user_id=$2 AND r.matter_id=$3 AND m.deleted_at IS NULL
		ON CONFLICT(firm_id,user_id,response_id) DO UPDATE SET rating=excluded.rating,reason=excluded.reason`,
		firmID, userID, matterID, responseID, rating, reason)
	if err != nil {
		return mapError(err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) SaveAIResponse(ctx context.Context, firmID, userID, matterID, responseID, model, retrieval string, citationCount int) error {
	result, err := s.Pool.Exec(ctx, `INSERT INTO ai_responses(id,firm_id,user_id,matter_id,model,retrieval,citation_count)
		SELECT $4,$1,$2,m.id,$5,$6,$7 FROM matters m
		WHERE m.firm_id=$1 AND m.id=$3 AND m.deleted_at IS NULL`,
		firmID, userID, matterID, responseID, model, retrieval, citationCount)
	if err != nil {
		return mapError(err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) EmbeddingQueueCounts(ctx context.Context) (pending, failed int64, err error) {
	err = s.Pool.QueryRow(ctx, `SELECT count(*) FILTER(WHERE embedding_status IN('pending','processing')),
		count(*) FILTER(WHERE embedding_status='failed') FROM document_chunks`).Scan(&pending, &failed)
	if errors.Is(err, pgx.ErrNoRows) {
		err = nil
	}
	return
}
