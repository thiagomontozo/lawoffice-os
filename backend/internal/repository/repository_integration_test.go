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
	ownerA, slugA := createFirm(t, store, ctx, "Alpha")
	ownerB, _ := createFirm(t, store, ctx, "Beta")
	preferences, err := store.NotificationPreferences(ctx, ownerA.FirmID, ownerA.ID)
	if err != nil || preferences.EmailDeadlines || preferences.EmailTasks {
		t.Fatalf("unexpected default notification preferences: preferences=%+v err=%v", preferences, err)
	}
	preferences, err = store.UpdateNotificationPreferences(ctx, ownerA.FirmID, ownerA.ID, domain.NotificationPreferences{EmailDeadlines: true, EmailTasks: true})
	if err != nil || !preferences.EmailDeadlines || !preferences.EmailTasks {
		t.Fatalf("update notification preferences: preferences=%+v err=%v", preferences, err)
	}
	var notificationID string
	if err = store.Pool.QueryRow(ctx, `INSERT INTO notifications(firm_id,user_id,type,title,message,resource_type) VALUES($1,$2,'deadline.approaching','Deadline notice','Deadline approaching','deadline') RETURNING id`, ownerA.FirmID, ownerA.ID).Scan(&notificationID); err != nil {
		t.Fatal(err)
	}
	pendingEmails, err := store.PendingEmailNotifications(ctx, 100)
	if err != nil {
		t.Fatal(err)
	}
	foundPending := false
	for _, pending := range pendingEmails {
		if pending.ID == notificationID {
			foundPending = pending.FirmID == ownerA.FirmID && pending.Email == ownerA.Email
		}
	}
	if !foundPending {
		t.Fatal("opted-in notification was not returned for email delivery")
	}
	if err = store.MarkNotificationEmailQueued(ctx, ownerA.FirmID, notificationID); err != nil {
		t.Fatal(err)
	}
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
	search, err := store.Search(ctx, ownerA.FirmID, other.ID, "Restricted", "matter", 20)
	if err != nil || len(search) != 0 {
		t.Fatalf("restricted Matter leaked through search: results=%+v err=%v", search, err)
	}
	if err = store.GrantMatterAccess(ctx, ownerA.FirmID, matter.ID, &other.ID, nil, "read"); err != nil {
		t.Fatal(err)
	}
	allowed, err = store.CanAccessMatter(ctx, ownerA.FirmID, other.ID, matter.ID, "read")
	if err != nil || !allowed {
		t.Fatalf("explicit Matter access not honored: allowed=%v err=%v", allowed, err)
	}
	search, err = store.Search(ctx, ownerA.FirmID, other.ID, "Restricted", "matter", 20)
	if err != nil || len(search) != 1 || search[0].ID != matter.ID {
		t.Fatalf("authorized ranked search failed: results=%+v err=%v", search, err)
	}
	search, err = store.Search(ctx, ownerB.FirmID, ownerB.ID, "Alpha Updated", "client", 20)
	if err != nil || len(search) != 0 {
		t.Fatalf("cross-firm client leaked through search: results=%+v err=%v", search, err)
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
	if err = store.SoftDeleteDocument(ctx, ownerB.FirmID, ownerB.ID, doc.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-firm document deletion should fail, got %v", err)
	}
	if err = store.SoftDeleteDocument(ctx, ownerA.FirmID, ownerA.ID, doc.ID); err != nil {
		t.Fatalf("soft delete failed: %v", err)
	}
	if _, _, err = store.DocumentVersion(ctx, ownerA.FirmID, ownerA.ID, doc.ID, nil); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted document should be hidden, got %v", err)
	}
	if err = store.RestoreDocument(ctx, ownerA.FirmID, doc.ID); err != nil {
		t.Fatalf("restore failed: %v", err)
	}
	portalToken, portalTokenHash, err := auth.NewToken("portal-test-secret")
	if err != nil || portalToken == "" {
		t.Fatalf("create portal token: %v", err)
	}
	portalID, err := store.CreatePortalInvitation(ctx, ownerA.FirmID, ownerA.ID, clientA.ID, "portal@example.test", []string{matter.ID}, portalTokenHash, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("create portal invitation: %v", err)
	}
	portalPassword, err := auth.HashPassword("portal-password-strong")
	if err != nil {
		t.Fatal(err)
	}
	acceptedID, err := store.AcceptPortalInvitation(ctx, portalTokenHash, portalPassword)
	if err != nil || acceptedID != portalID {
		t.Fatalf("accept portal invitation: id=%s err=%v", acceptedID, err)
	}
	if _, err = store.AcceptPortalInvitation(ctx, portalTokenHash, portalPassword); !errors.Is(err, ErrNotFound) {
		t.Fatalf("portal invitation should be single use, got %v", err)
	}
	firmID, authenticatedPortalID, storedHash, err := store.PortalCredentials(ctx, slugA, "portal@example.test")
	if err != nil || firmID != ownerA.FirmID || authenticatedPortalID != portalID || !auth.CheckPassword(storedHash, "portal-password-strong") {
		t.Fatalf("portal credentials unavailable: firm=%s portal=%s err=%v", firmID, authenticatedPortalID, err)
	}
	_, portalSessionHash, err := auth.NewToken("portal-test-secret")
	if err != nil {
		t.Fatal(err)
	}
	if err = store.CreatePortalSession(ctx, ownerA.FirmID, portalID, portalSessionHash, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	_, resetHash, err := auth.NewToken("portal-test-secret")
	if err != nil {
		t.Fatal(err)
	}
	resetFirmID, err := store.CreatePortalPasswordReset(ctx, slugA, "portal@example.test", resetHash, time.Now().Add(time.Hour))
	if err != nil || resetFirmID != ownerA.FirmID {
		t.Fatalf("create portal password reset: firm=%s err=%v", resetFirmID, err)
	}
	resetPassword, err := auth.HashPassword("portal-password-replaced")
	if err != nil {
		t.Fatal(err)
	}
	resetFirmID, resetPortalID, err := store.ResetPortalPassword(ctx, resetHash, resetPassword)
	if err != nil || resetFirmID != ownerA.FirmID || resetPortalID != portalID {
		t.Fatalf("reset portal password: firm=%s portal=%s err=%v", resetFirmID, resetPortalID, err)
	}
	if _, _, err = store.PortalBySession(ctx, portalSessionHash); !errors.Is(err, ErrNotFound) {
		t.Fatalf("portal password reset should revoke sessions, got %v", err)
	}
	_, _, replacedHash, err := store.PortalCredentials(ctx, slugA, "portal@example.test")
	if err != nil || !auth.CheckPassword(replacedHash, "portal-password-replaced") {
		t.Fatalf("portal password was not replaced: %v", err)
	}
	if _, _, err = store.ResetPortalPassword(ctx, resetHash, resetPassword); !errors.Is(err, ErrNotFound) {
		t.Fatalf("portal password reset should be single use, got %v", err)
	}
	if _, _, err = store.PortalDocument(ctx, ownerA.FirmID, portalID, doc.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("private document leaked to portal, got %v", err)
	}
	publicDoc, err := store.CreateDocument(ctx, ownerA.FirmID, ownerA.ID, domain.Document{MatterID: &matter.ID, Title: "Shared evidence", Category: "evidence", OriginalFileName: "shared.pdf", MimeType: "application/pdf", SizeBytes: 3, Checksum: "def", ClientVisible: true}, uuid.NewString()+".pdf")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = store.PortalDocument(ctx, ownerA.FirmID, portalID, publicDoc.ID); err != nil {
		t.Fatalf("shared document unavailable to portal: %v", err)
	}
	if err = store.SetPortalUserActive(ctx, ownerA.FirmID, portalID, false); err != nil {
		t.Fatalf("revoke portal user: %v", err)
	}
	if _, _, _, err = store.PortalCredentials(ctx, slugA, "portal@example.test"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("revoked portal user should not authenticate, got %v", err)
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
