package jobs

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/mailer"
)

type Queue struct {
	pool       *pgxpool.Pool
	sender     mailer.Sender
	aead       cipher.AEAD
	workerID   string
	logger     *slog.Logger
	pollPeriod time.Duration
}

type claimedJob struct {
	ID               string
	EncryptedPayload []byte
	Attempts         int
	MaxAttempts      int
}

func New(pool *pgxpool.Pool, secret string, sender mailer.Sender, logger *slog.Logger) (*Queue, error) {
	key := sha256.Sum256([]byte("lawoffice-outbound-jobs:" + secret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Queue{pool: pool, sender: sender, aead: aead, workerID: uuid.NewString(), logger: logger, pollPeriod: 10 * time.Second}, nil
}

func (q *Queue) Enabled() bool { return q.sender != nil }

func (q *Queue) EnqueueEmail(ctx context.Context, firmID string, message mailer.Message) (bool, error) {
	if q.sender == nil {
		return false, nil
	}
	payload, err := json.Marshal(message)
	if err != nil {
		return false, err
	}
	encrypted, err := q.seal(payload)
	if err != nil {
		return false, err
	}
	_, err = q.pool.Exec(ctx, `INSERT INTO outbound_jobs(firm_id,job_type,encrypted_payload) VALUES($1,'email.send',$2)`, firmID, encrypted)
	return err == nil, err
}

func (q *Queue) Run(ctx context.Context) {
	q.process(ctx)
	ticker := time.NewTicker(q.pollPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			q.logger.Info("outbound job worker stopped")
			return
		case <-ticker.C:
			q.process(ctx)
		}
	}
}

func (q *Queue) process(ctx context.Context) {
	if q.sender == nil {
		return
	}
	workContext, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	jobs, err := q.claim(workContext, 10)
	if err != nil {
		if ctx.Err() == nil {
			q.logger.Error("outbound job claim failed", "error", err)
		}
		return
	}
	for _, job := range jobs {
		if err = q.deliver(workContext, job); err != nil {
			q.fail(workContext, job, err)
			continue
		}
		if _, err = q.pool.Exec(workContext, `UPDATE outbound_jobs SET status='completed',encrypted_payload=NULL,completed_at=now(),locked_at=NULL,locked_by=NULL,last_error=NULL WHERE id=$1 AND locked_by=$2`, job.ID, q.workerID); err != nil && ctx.Err() == nil {
			q.logger.Error("outbound job completion failed", "job_id", job.ID, "error", err)
		}
	}
	_, _ = q.pool.Exec(workContext, `DELETE FROM outbound_jobs WHERE status='completed' AND completed_at<now()-interval '30 days'`)
}

func (q *Queue) claim(ctx context.Context, limit int) ([]claimedJob, error) {
	rows, err := q.pool.Query(ctx, `WITH candidates AS (
        SELECT id FROM outbound_jobs
        WHERE (status='pending' AND available_at<=now()) OR (status='processing' AND locked_at<now()-interval '5 minutes')
        ORDER BY available_at,created_at
        FOR UPDATE SKIP LOCKED LIMIT $1
    )
    UPDATE outbound_jobs j SET status='processing',attempts=j.attempts+1,locked_at=now(),locked_by=$2
    FROM candidates c WHERE j.id=c.id
    RETURNING j.id,j.encrypted_payload,j.attempts,j.max_attempts`, limit, q.workerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []claimedJob{}
	for rows.Next() {
		var job claimedJob
		if err = rows.Scan(&job.ID, &job.EncryptedPayload, &job.Attempts, &job.MaxAttempts); err != nil {
			return nil, err
		}
		result = append(result, job)
	}
	return result, rows.Err()
}

func (q *Queue) deliver(ctx context.Context, job claimedJob) error {
	plaintext, err := q.open(job.EncryptedPayload)
	if err != nil {
		return err
	}
	var message mailer.Message
	if err = json.Unmarshal(plaintext, &message); err != nil {
		return err
	}
	return q.sender.Send(ctx, message)
}

func (q *Queue) fail(ctx context.Context, job claimedJob, deliveryErr error) {
	status := "pending"
	if job.Attempts >= job.MaxAttempts {
		status = "failed"
	}
	exponent := job.Attempts
	if exponent > 6 {
		exponent = 6
	}
	delay := time.Duration(1<<exponent) * 30 * time.Second
	message := fmt.Sprintf("delivery failed (%T)", deliveryErr)
	_, updateErr := q.pool.Exec(ctx, `UPDATE outbound_jobs SET status=$3,available_at=now()+($4 * interval '1 second'),locked_at=NULL,locked_by=NULL,last_error=$5,encrypted_payload=CASE WHEN $3='failed' THEN NULL ELSE encrypted_payload END WHERE id=$1 AND locked_by=$2`, job.ID, q.workerID, status, delay.Seconds(), message)
	if updateErr != nil {
		q.logger.Error("outbound job retry update failed", "job_id", job.ID, "error", updateErr)
		return
	}
	q.logger.Warn("outbound job delivery failed", "job_id", job.ID, "attempt", job.Attempts, "terminal", status == "failed")
}

func (q *Queue) seal(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, q.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return q.aead.Seal(nonce, nonce, plaintext, nil), nil
}

func (q *Queue) open(encrypted []byte) ([]byte, error) {
	if len(encrypted) < q.aead.NonceSize() {
		return nil, errors.New("invalid encrypted job payload")
	}
	nonce := encrypted[:q.aead.NonceSize()]
	return q.aead.Open(nil, nonce, encrypted[q.aead.NonceSize():], nil)
}
