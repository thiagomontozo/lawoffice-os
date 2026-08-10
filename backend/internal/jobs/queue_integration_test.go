package jobs

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/database"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/mailer"
)

type recordingSender struct {
	mu       sync.Mutex
	messages []mailer.Message
}

func (s *recordingSender) Send(_ context.Context, message mailer.Message) error {
	s.mu.Lock()
	s.messages = append(s.messages, message)
	s.mu.Unlock()
	return nil
}

func TestPostgresQueueClaimsJobOnceAcrossWorkers(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := database.Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	migrations, err := filepath.Abs(filepath.Join("..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	if err = database.Migrate(ctx, pool, migrations); err != nil {
		t.Fatal(err)
	}
	var firmID string
	suffix := uuid.NewString()
	if err = pool.QueryRow(ctx, `INSERT INTO firms(legal_name,display_name,slug,email) VALUES($1,$2,$3,$4) RETURNING id`, "Queue Legal", "Queue", "queue-"+suffix, suffix+"@example.test").Scan(&firmID); err != nil {
		t.Fatal(err)
	}
	sender := &recordingSender{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	first, err := New(pool, "durable-job-secret", sender, logger)
	if err != nil {
		t.Fatal(err)
	}
	second, err := New(pool, "durable-job-secret", sender, logger)
	if err != nil {
		t.Fatal(err)
	}
	queued, err := first.EnsureEmail(ctx, firmID, "test-notification:"+suffix, mailer.Message{To: "client@example.test", Subject: "Portal", Text: "one-time-link"})
	if err != nil || !queued {
		t.Fatalf("enqueue: queued=%v err=%v", queued, err)
	}
	queued, err = second.EnsureEmail(ctx, firmID, "test-notification:"+suffix, mailer.Message{To: "client@example.test", Subject: "Portal duplicate", Text: "duplicate"})
	if err != nil || !queued {
		t.Fatalf("deduplicated enqueue: queued=%v err=%v", queued, err)
	}
	var workers sync.WaitGroup
	workers.Add(2)
	go func() { defer workers.Done(); first.process(ctx) }()
	go func() { defer workers.Done(); second.process(ctx) }()
	workers.Wait()
	sender.mu.Lock()
	messageCount := len(sender.messages)
	sender.mu.Unlock()
	if messageCount != 1 {
		t.Fatalf("job delivered %d times", messageCount)
	}
	var status string
	var payloadMissing bool
	var jobCount int
	if err = pool.QueryRow(ctx, `SELECT count(*),min(status),bool_and(encrypted_payload IS NULL) FROM outbound_jobs WHERE firm_id=$1 AND deduplication_key=$2`, firmID, "test-notification:"+suffix).Scan(&jobCount, &status, &payloadMissing); err != nil {
		t.Fatal(err)
	}
	if jobCount != 1 || status != "completed" || !payloadMissing {
		t.Fatalf("unexpected terminal job state: count=%d status=%s payloadMissing=%v", jobCount, status, payloadMissing)
	}
}
