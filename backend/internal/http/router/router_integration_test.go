package router

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/config"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/database"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/http/handlers"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/jobs"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/observability"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/realtime"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/repository"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/service"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/storage"
)

type integrationApp struct {
	handler http.Handler
	pool    *pgxpool.Pool
	hub     *realtime.Hub
}

func newIntegrationApp(t *testing.T) *integrationApp {
	t.Helper()
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	pool, err := database.Open(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(pool.Close)
	migrations, err := filepath.Abs(filepath.Join("..", "..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	if err = database.Migrate(ctx, pool, migrations); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	objects, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("storage: %v", err)
	}
	store := repository.New(pool)
	hub := realtime.New()
	t.Cleanup(hub.Close)
	cfg := config.Config{
		Environment:   "development",
		WebOrigin:     "https://office.example",
		SessionSecret: "http-integration-secret-with-sufficient-entropy",
		SessionTTL:    time.Hour,
		MaxUpload:     1024 * 1024,
		Locale:        "pt-BR",
		Timezone:      "America/Sao_Paulo",
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	jobQueue, err := jobs.New(pool, cfg.SessionSecret, nil, logger)
	if err != nil {
		t.Fatalf("jobs: %v", err)
	}
	handler := handlers.New(store, service.New(store, objects, nil, cfg.MaxUpload), objects, pool, cfg, logger, hub, jobQueue)
	return &integrationApp{handler: New(handler, store, cfg, observability.NewMetrics()), pool: pool, hub: hub}
}

func (a *integrationApp) request(t *testing.T, method, path string, body any, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	var payload io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		payload = bytes.NewReader(encoded)
	}
	request := httptest.NewRequest(method, path, payload)
	request.RemoteAddr = "192.0.2.44:4242"
	request.Header.Set("Origin", "https://office.example")
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	a.handler.ServeHTTP(response, request)
	return response
}

func responseCookie(t *testing.T, response *httptest.ResponseRecorder, name string) *http.Cookie {
	t.Helper()
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}
	t.Fatalf("cookie %q was not issued", name)
	return nil
}

func responseObject(t *testing.T, response *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var result map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode response %d: %v\n%s", response.Code, err, response.Body.String())
	}
	return result
}

func setupFirm(t *testing.T, app *integrationApp, suffix string) *http.Cookie {
	t.Helper()
	response := app.request(t, http.MethodPost, "/api/v1/setup", map[string]any{
		"legalName":   "Integration " + suffix + " Legal",
		"displayName": "Integration " + suffix,
		"slug":        "integration-" + strings.ToLower(suffix),
		"email":       strings.ToLower(suffix) + "@firm.example",
		"adminName":   "Owner " + suffix,
		"adminEmail":  strings.ToLower(suffix) + "@owner.example",
		"password":    "correct-horse-battery-staple",
	})
	if response.Code != http.StatusCreated {
		t.Fatalf("setup %s: status=%d body=%s", suffix, response.Code, response.Body.String())
	}
	return responseCookie(t, response, handlers.SessionCookie)
}

func TestHTTPAuthenticationFirmIsolationPortalAndSessionRevocation(t *testing.T) {
	app := newIntegrationApp(t)
	suffix := strings.ReplaceAll(time.Now().UTC().Format("150405.000000000"), ".", "")
	alpha := setupFirm(t, app, "alpha"+suffix)
	beta := setupFirm(t, app, "beta"+suffix)

	unauthorized := app.request(t, http.MethodGet, "/api/v1/clients", nil)
	if unauthorized.Code != http.StatusUnauthorized || unauthorized.Header().Get("X-Request-ID") == "" {
		t.Fatalf("protected route response: status=%d request-id=%q", unauthorized.Code, unauthorized.Header().Get("X-Request-ID"))
	}

	clientResponse := app.request(t, http.MethodPost, "/api/v1/clients", map[string]any{"type": "person", "name": "Portal Client"}, alpha)
	if clientResponse.Code != http.StatusCreated {
		t.Fatalf("create client: status=%d body=%s", clientResponse.Code, clientResponse.Body.String())
	}
	clientID, _ := responseObject(t, clientResponse)["id"].(string)
	if clientID == "" {
		t.Fatal("create client did not return an ID")
	}

	crossFirm := app.request(t, http.MethodGet, "/api/v1/clients/"+clientID, nil, beta)
	if crossFirm.Code != http.StatusNotFound {
		t.Fatalf("cross-firm client leaked: status=%d body=%s", crossFirm.Code, crossFirm.Body.String())
	}

	matterResponse := app.request(t, http.MethodPost, "/api/v1/matters", map[string]any{
		"type": "legal_consultation", "internalNumber": "INT-" + suffix, "title": "Confidential HTTP Matter", "status": "active", "confidentiality": "restricted",
	}, alpha)
	if matterResponse.Code != http.StatusCreated {
		t.Fatalf("create Matter: status=%d body=%s", matterResponse.Code, matterResponse.Body.String())
	}
	matterID, _ := responseObject(t, matterResponse)["id"].(string)
	if matterID == "" {
		t.Fatal("create Matter did not return an ID")
	}
	crossFirmMatter := app.request(t, http.MethodGet, "/api/v1/matters/"+matterID, nil, beta)
	if crossFirmMatter.Code != http.StatusNotFound {
		t.Fatalf("cross-firm Matter leaked: status=%d body=%s", crossFirmMatter.Code, crossFirmMatter.Body.String())
	}

	invitation := app.request(t, http.MethodPost, "/api/v1/portal/invitations", map[string]any{
		"clientId": clientID, "email": "portal." + suffix + "@example.test", "matterIds": []string{matterID},
	}, alpha)
	if invitation.Code != http.StatusCreated {
		t.Fatalf("create portal invitation: status=%d body=%s", invitation.Code, invitation.Body.String())
	}
	invitationURL, _ := responseObject(t, invitation)["invitationUrl"].(string)
	parsed, err := url.Parse(invitationURL)
	if err != nil || parsed.Query().Get("token") == "" {
		t.Fatalf("invalid invitation URL: %q", invitationURL)
	}
	portalEmail := "portal." + suffix + "@example.test"
	accepted := app.request(t, http.MethodPost, "/api/v1/portal/invitations/accept", map[string]any{"token": parsed.Query().Get("token"), "password": "portal-password-strong"})
	if accepted.Code != http.StatusOK {
		t.Fatalf("accept invitation: status=%d body=%s", accepted.Code, accepted.Body.String())
	}
	portalLogin := app.request(t, http.MethodPost, "/api/v1/portal/login", map[string]any{
		"firmSlug": "integration-alpha" + suffix, "email": portalEmail, "password": "portal-password-strong",
	})
	if portalLogin.Code != http.StatusOK {
		t.Fatalf("portal login: status=%d body=%s", portalLogin.Code, portalLogin.Body.String())
	}
	portalSession := responseCookie(t, portalLogin, "lawoffice_portal_session")
	portalMatter := app.request(t, http.MethodGet, "/api/v1/portal/matters/"+matterID, nil, portalSession)
	if portalMatter.Code != http.StatusOK {
		t.Fatalf("shared portal Matter unavailable: status=%d body=%s", portalMatter.Code, portalMatter.Body.String())
	}
	knownRecovery := app.request(t, http.MethodPost, "/api/v1/portal/password/forgot", map[string]any{"firmSlug": "integration-alpha" + suffix, "email": portalEmail})
	unknownRecovery := app.request(t, http.MethodPost, "/api/v1/portal/password/forgot", map[string]any{"firmSlug": "integration-alpha" + suffix, "email": "unknown." + suffix + "@example.test"})
	if knownRecovery.Code != http.StatusAccepted || unknownRecovery.Code != http.StatusAccepted || knownRecovery.Body.String() != unknownRecovery.Body.String() {
		t.Fatalf("password recovery disclosed account existence: known=%d/%s unknown=%d/%s", knownRecovery.Code, knownRecovery.Body.String(), unknownRecovery.Code, unknownRecovery.Body.String())
	}

	changed := app.request(t, http.MethodPost, "/api/v1/auth/change-password", map[string]any{
		"currentPassword": "correct-horse-battery-staple", "newPassword": "a-new-secure-administrator-password",
	}, alpha)
	if changed.Code != http.StatusNoContent {
		t.Fatalf("change password: status=%d body=%s", changed.Code, changed.Body.String())
	}
	revoked := app.request(t, http.MethodGet, "/api/v1/auth/me", nil, alpha)
	if revoked.Code != http.StatusUnauthorized {
		t.Fatalf("old session survived password change: status=%d", revoked.Code)
	}
	relogin := app.request(t, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"firmSlug": "integration-alpha" + suffix, "email": "alpha" + suffix + "@owner.example", "password": "a-new-secure-administrator-password",
	})
	if relogin.Code != http.StatusOK {
		t.Fatalf("login with changed password: status=%d body=%s", relogin.Code, relogin.Body.String())
	}
}
