package realtime

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/database"
)

func TestPostgresPersistsAndIsolatesReplay(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx, cancel := context.WithCancel(context.Background())
	pool, err := database.Open(ctx, databaseURL)
	if err != nil {
		cancel()
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(pool.Close)
	t.Cleanup(cancel)
	migrations, err := filepath.Abs(filepath.Join("..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	if err = database.Migrate(ctx, pool, migrations); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	createFirm := func(label string) string {
		t.Helper()
		var id string
		suffix := uuid.NewString()
		err = pool.QueryRow(ctx, `INSERT INTO firms(legal_name,display_name,slug,email) VALUES($1,$2,$3,$4) RETURNING id`, label+" Legal", label, "realtime-"+suffix, suffix+"@example.test").Scan(&id)
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	firmID := createFirm("Realtime Alpha")
	otherFirmID := createFirm("Realtime Beta")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	firstHub := New()
	t.Cleanup(firstHub.Close)
	if err = StartPostgres(ctx, pool, firstHub, logger); err != nil {
		t.Fatalf("start first realtime hub: %v", err)
	}
	stream, _, err := firstHub.Subscribe(ctx, firmID, "")
	if err != nil {
		t.Fatal(err)
	}
	firstHub.Publish(firmID, Event{Type: "matter.updated", ResourceType: "matter", ResourceID: uuid.NewString()})
	var first Event
	select {
	case first = <-stream:
	case <-time.After(3 * time.Second):
		t.Fatal("database event was not delivered")
	}
	if first.ID == "" || first.PublishedAt.IsZero() {
		t.Fatalf("persistent event metadata missing: %+v", first)
	}

	secondResource := uuid.NewString()
	firstHub.Publish(firmID, Event{Type: "task.updated", ResourceType: "task", ResourceID: secondResource})
	select {
	case <-stream:
	case <-time.After(3 * time.Second):
		t.Fatal("second database event was not committed")
	}

	secondHub := New()
	t.Cleanup(secondHub.Close)
	if err = StartPostgres(ctx, pool, secondHub, logger); err != nil {
		t.Fatalf("start second realtime hub: %v", err)
	}
	_, initialReplay, err := secondHub.Subscribe(ctx, firmID, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(initialReplay) != 0 {
		t.Fatalf("initial connection unexpectedly replayed history: %+v", initialReplay)
	}
	_, replay, err := secondHub.Subscribe(ctx, firmID, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(replay) != 1 || replay[0].ResourceID != secondResource {
		t.Fatalf("durable cross-instance replay mismatch: %+v", replay)
	}
	_, isolatedReplay, err := secondHub.Subscribe(ctx, otherFirmID, "0")
	if err != nil {
		t.Fatal(err)
	}
	if len(isolatedReplay) != 0 {
		t.Fatalf("cross-firm events leaked: %+v", isolatedReplay)
	}
}
