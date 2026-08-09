package repository

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/auth"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/database"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/domain"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func integrationStore(t *testing.T) (*Store, context.Context) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	pool, err := database.Open(ctx, url)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(pool.Close)
	migrations, err := filepath.Abs(filepath.Join("..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	if err = database.Migrate(ctx, pool, migrations); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return New(pool), ctx
}
func createFirm(t *testing.T, store *Store, ctx context.Context, label string) (domain.User, string) {
	t.Helper()
	suffix := uuid.NewString()[:8]
	passwordHash, err := auth.HashPassword("integration-password")
	if err != nil {
		t.Fatal(err)
	}
	firm := domain.Firm{LegalName: label + " Legal", DisplayName: label, Slug: "firm-" + suffix, Email: "owner-" + suffix + "@example.test", Timezone: "America/Sao_Paulo", Locale: "pt-BR"}
	user, err := store.Setup(ctx, firm, "Owner "+label, firm.Email, passwordHash)
	if err != nil {
		t.Fatalf("setup firm: %v", err)
	}
	return user, firm.Slug
}
func TestFirmIsolationMatterAccessAndDocuments(t *testing.T) {
	store, ctx := integrationStore(t)
	ownerA, _ := createFirm(t, store, ctx, "Alpha")
	ownerB, _ := createFirm(t, store, ctx, "Beta")
	clientA, err := store.CreateClient(ctx, ownerA.FirmID, domain.Client{Type: "person", Name: "Client Alpha"})
	if err != nil {
		t.Fatal(err)
	}
	clientB, err := store.CreateClient(ctx, ownerB.FirmID, domain.Client{Type: "person", Name: "Client Beta"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.Client(ctx, ownerA.FirmID, clientB.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-firm client lookup should fail, got %v", err)
	}
	if _, err = store.Client(ctx, ownerA.FirmID, clientA.ID); err != nil {
		t.Fatalf("own client missing: %v", err)
	}
	clientA.Name = "Client Alpha Updated"
	if _, err = store.UpdateClient(ctx, ownerB.FirmID, clientA.ID, clientA); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-firm client update should fail, got %v", err)
	}
	if clientA, err = store.UpdateClient(ctx, ownerA.FirmID, clientA.ID, clientA); err != nil || clientA.Name != "Client Alpha Updated" {
		t.Fatalf("own client update failed: client=%+v err=%v", clientA, err)
	}
	passwordHash, _ := auth.HashPassword("secondary-password")
	other, err := store.CreateUser(ctx, ownerA.FirmID, "Other Lawyer", "other-"+uuid.NewString()[:8]+"@example.test", passwordHash, nil)
	if err != nil {
		t.Fatal(err)
	}
	matter, err := store.CreateMatter(ctx, ownerA.FirmID, ownerA.ID, domain.Matter{Type: "legal_consultation", InternalNumber: "MAT-" + uuid.NewString()[:8], Title: "Restricted advice", Status: "active", Priority: "high", Confidentiality: "restricted", OpenedAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.Matter(ctx, ownerB.FirmID, matter.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-firm Matter lookup should fail, got %v", err)
	}
	allowed, err := store.CanAccessMatter(ctx, ownerA.FirmID, other.ID, matter.ID, "read")
	if err != nil || allowed {
		t.Fatalf("restricted Matter unexpectedly accessible: allowed=%v err=%v", allowed, err)
	}
	if err = store.GrantMatterAccess(ctx, ownerA.FirmID, matter.ID, &other.ID, nil, "read"); err != nil {
		t.Fatal(err)
	}
	allowed, err = store.CanAccessMatter(ctx, ownerA.FirmID, other.ID, matter.ID, "read")
	if err != nil || !allowed {
		t.Fatalf("explicit Matter access not honored: allowed=%v err=%v", allowed, err)
	}
	doc, err := store.CreateDocument(ctx, ownerA.FirmID, ownerA.ID, domain.Document{MatterID: &matter.ID, Title: "Restricted evidence", Category: "evidence", OriginalFileName: "evidence.pdf", MimeType: "application/pdf", SizeBytes: 3, Checksum: "abc"}, uuid.NewString()+".pdf")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = store.DocumentVersion(ctx, ownerB.FirmID, ownerB.ID, doc.ID, nil); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-firm document should be hidden, got %v", err)
	}
	if _, _, err = store.DocumentVersion(ctx, ownerA.FirmID, other.ID, doc.ID, nil); err != nil {
		t.Fatalf("authorized document lookup failed: %v", err)
	}
	task, err := store.CreateTask(ctx, ownerA.FirmID, ownerA.ID, domain.Task{MatterID: &matter.ID, Title: "Review restricted advice", Status: "todo", Priority: "normal"})
	if err != nil {
		t.Fatal(err)
	}
	if err = store.UpdateTaskStatus(ctx, ownerB.FirmID, ownerB.ID, task.ID, "done"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-firm task update should fail, got %v", err)
	}
	if err = store.UpdateTaskStatus(ctx, ownerA.FirmID, ownerA.ID, task.ID, "done"); err != nil {
		t.Fatalf("own task update failed: %v", err)
	}
}
