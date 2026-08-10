package config

import "testing"

func validEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv("APP_ENV", "development")
	t.Setenv("API_PORT", "8080")
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("WEB_ORIGIN", "http://localhost:5173")
	t.Setenv("SESSION_SECRET", "development-secret")
	t.Setenv("MAX_UPLOAD_MB", "25")
	t.Setenv("LOG_LEVEL", "info")
}
func TestLoadValidConfiguration(t *testing.T) {
	validEnvironment(t)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Port != "8080" || cfg.MaxUpload != 25*1024*1024 {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}
func TestProductionRejectsPlaceholderSecret(t *testing.T) {
	validEnvironment(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("SESSION_SECRET", "change-me")
	if _, err := Load(); err == nil {
		t.Fatal("production placeholder secret should be rejected")
	}
}
func TestProductionRejectsWeakMetricsTokenWhenEnabled(t *testing.T) {
	validEnvironment(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("SESSION_SECRET", "a-production-session-secret-with-entropy")
	t.Setenv("METRICS_TOKEN", "short")
	if _, err := Load(); err == nil {
		t.Fatal("production weak metrics token should be rejected")
	}
}
func TestOriginMustBeExact(t *testing.T) {
	validEnvironment(t)
	t.Setenv("WEB_ORIGIN", "http://localhost:5173/path")
	if _, err := Load(); err == nil {
		t.Fatal("origin with path should be rejected")
	}
}
func TestS3ConfigurationRequiresCredentialsAndSchemeFreeEndpoint(t *testing.T) {
	validEnvironment(t)
	t.Setenv("STORAGE_DRIVER", "s3")
	t.Setenv("S3_ENDPOINT", "https://objects.example")
	t.Setenv("S3_BUCKET", "documents")
	t.Setenv("S3_ACCESS_KEY", "access")
	t.Setenv("S3_SECRET_KEY", "secret")
	if _, err := Load(); err == nil {
		t.Fatal("S3 endpoint with scheme should be rejected")
	}
}
func TestRequiredUploadScannerNeedsAddress(t *testing.T) {
	validEnvironment(t)
	t.Setenv("UPLOAD_SCAN_MODE", "required")
	if _, err := Load(); err == nil {
		t.Fatal("required scanner without CLAMAV_ADDRESS should be rejected")
	}
	t.Setenv("CLAMAV_ADDRESS", "clamav:3310")
	if _, err := Load(); err != nil {
		t.Fatalf("valid scanner configuration rejected: %v", err)
	}
}
func TestProductionSMTPRequiresTLSAndStrongJobSecret(t *testing.T) {
	validEnvironment(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("SESSION_SECRET", "a-production-session-secret-with-entropy")
	t.Setenv("SMTP_MODE", "enabled")
	t.Setenv("SMTP_ADDRESS", "smtp.example.test:587")
	t.Setenv("SMTP_FROM", "portal@example.test")
	t.Setenv("SMTP_REQUIRE_TLS", "false")
	if _, err := Load(); err == nil {
		t.Fatal("production SMTP without TLS should be rejected")
	}
	t.Setenv("SMTP_REQUIRE_TLS", "true")
	t.Setenv("JOB_ENCRYPTION_SECRET", "short")
	if _, err := Load(); err == nil {
		t.Fatal("production SMTP with weak job encryption secret should be rejected")
	}
}
