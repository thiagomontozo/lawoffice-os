package config

import (
	"errors"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Environment, Port, DatabaseURL, WebOrigin, SessionSecret, MetricsToken, StoragePath, MigrationsDir, Locale, Timezone string
	MaxUpload                                                                                                            int64
	LogLevel                                                                                                             slog.Level
	SessionTTL                                                                                                           time.Duration
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
func value(k, d string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return d
}
