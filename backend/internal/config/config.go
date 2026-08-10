package config

import (
	"errors"
	"log/slog"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Environment, Port, DatabaseURL, WebOrigin, SessionSecret, MetricsToken, StoragePath, MigrationsDir, Locale, Timezone string
	StorageDriver, S3Endpoint, S3Bucket, S3AccessKey, S3SecretKey, S3Region, UploadScanMode, ClamAVAddress               string
	MaxUpload                                                                                                            int64
	LogLevel                                                                                                             slog.Level
	SessionTTL                                                                                                           time.Duration
	S3UseTLS, S3CreateBucket                                                                                             bool
}

func Load() (Config, error) {
	c := Config{Environment: value("APP_ENV", "development"), Port: value("API_PORT", "8080"), DatabaseURL: strings.TrimSpace(os.Getenv("DATABASE_URL")), WebOrigin: value("WEB_ORIGIN", "http://localhost:5173"), SessionSecret: strings.TrimSpace(os.Getenv("SESSION_SECRET")), MetricsToken: strings.TrimSpace(os.Getenv("METRICS_TOKEN")), StoragePath: value("STORAGE_PATH", "./data/storage"), MigrationsDir: value("MIGRATIONS_DIR", "./migrations"), Locale: value("DEFAULT_LOCALE", "pt-BR"), Timezone: value("DEFAULT_TIMEZONE", "America/Sao_Paulo"), SessionTTL: 30 * 24 * time.Hour}
	if c.DatabaseURL == "" || c.SessionSecret == "" {
		return c, errors.New("DATABASE_URL and SESSION_SECRET are required")
	}
	if c.Environment != "development" && c.Environment != "production" {
		return c, errors.New("APP_ENV must be development or production")
	}
	p, e := strconv.Atoi(c.Port)
	if e != nil || p < 1 || p > 65535 {
		return c, errors.New("API_PORT must be between 1 and 65535")
	}
	u, e := url.Parse(c.WebOrigin)
	if e != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.Path != "" {
		return c, errors.New("WEB_ORIGIN must be an exact HTTP(S) origin")
	}
	if c.Environment == "production" && (c.SessionSecret == "change-me" || len(c.SessionSecret) < 32) {
		return c, errors.New("SESSION_SECRET must contain at least 32 characters in production")
	}
	if c.Environment == "production" && c.MetricsToken != "" && (c.MetricsToken == "change-me" || len(c.MetricsToken) < 32) {
		return c, errors.New("METRICS_TOKEN must contain at least 32 characters in production when metrics are enabled")
	}
	mb, e := strconv.ParseInt(value("MAX_UPLOAD_MB", "25"), 10, 64)
	if e != nil || mb < 1 || mb > 100 {
		return c, errors.New("MAX_UPLOAD_MB must be 1..100")
	}
	c.MaxUpload = mb * 1024 * 1024
	c.StorageDriver = strings.ToLower(value("STORAGE_DRIVER", "local"))
	c.S3Endpoint = strings.TrimSpace(os.Getenv("S3_ENDPOINT"))
	c.S3Bucket = strings.TrimSpace(os.Getenv("S3_BUCKET"))
	c.S3AccessKey = strings.TrimSpace(os.Getenv("S3_ACCESS_KEY"))
	c.S3SecretKey = strings.TrimSpace(os.Getenv("S3_SECRET_KEY"))
	c.S3Region = value("S3_REGION", "us-east-1")
	c.S3UseTLS, e = boolean("S3_USE_TLS", true)
	if e != nil {
		return c, e
	}
	c.S3CreateBucket, e = boolean("S3_CREATE_BUCKET", false)
	if e != nil {
		return c, e
	}
	if c.StorageDriver != "local" && c.StorageDriver != "s3" {
		return c, errors.New("STORAGE_DRIVER must be local or s3")
	}
	if c.StorageDriver == "s3" && (c.S3Endpoint == "" || c.S3Bucket == "" || c.S3AccessKey == "" || c.S3SecretKey == "" || strings.Contains(c.S3Endpoint, "://")) {
		return c, errors.New("S3_ENDPOINT without scheme, S3_BUCKET, S3_ACCESS_KEY and S3_SECRET_KEY are required for S3 storage")
	}
	c.UploadScanMode = strings.ToLower(value("UPLOAD_SCAN_MODE", "off"))
	c.ClamAVAddress = strings.TrimSpace(os.Getenv("CLAMAV_ADDRESS"))
	if c.UploadScanMode != "off" && c.UploadScanMode != "required" {
		return c, errors.New("UPLOAD_SCAN_MODE must be off or required")
	}
	if c.UploadScanMode == "required" {
		if _, _, splitErr := net.SplitHostPort(c.ClamAVAddress); splitErr != nil {
			return c, errors.New("CLAMAV_ADDRESS must be a host:port when upload scanning is required")
		}
	}
	c.StoragePath, e = filepath.Abs(c.StoragePath)
	if e != nil {
		return c, e
	}
	levels := map[string]slog.Level{"debug": slog.LevelDebug, "info": slog.LevelInfo, "warn": slog.LevelWarn, "error": slog.LevelError}
	level, ok := levels[strings.ToLower(value("LOG_LEVEL", "info"))]
	if !ok {
		return c, errors.New("invalid LOG_LEVEL")
	}
	c.LogLevel = level
	return c, nil
}

func boolean(key string, fallback bool) (bool, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(raw)
	if err != nil {
		return false, errors.New(key + " must be true or false")
	}
	return parsed, nil
}
func value(k, d string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return d
}
