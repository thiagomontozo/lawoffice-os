package middleware

import (
	"context"
	"crypto/subtle"
	"github.com/google/uuid"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/auth"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/domain"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/repository"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"
)

type key string

const (
	userKey    key = "user"
	requestKey key = "request"
)

type ErrorWriter func(http.ResponseWriter, *http.Request, int, string, string)

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if id == "" || len(id) > 128 {
			id = uuid.NewString()
		}
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestKey, id)))
	})
}
func RequestIDValue(r *http.Request) string { v, _ := r.Context().Value(requestKey).(string); return v }
func User(ctx context.Context) (domain.User, bool) {
	u, ok := ctx.Value(userKey).(domain.User)
	return u, ok
}
func CORS(origin string, write ErrorWriter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			o := r.Header.Get("Origin")
			if o != "" && o != origin && r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
				write(w, r, http.StatusForbidden, "ORIGIN_NOT_ALLOWED", "Origin not allowed")
				return
			}
			if o == origin {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Add("Vary", "Origin")
			}
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Request-ID")
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Referrer-Policy", "same-origin")
			w.Header().Set("Permissions-Policy", "camera=(), geolocation=(), microphone=()")
			w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; base-uri 'none'")
			if r.Method == "OPTIONS" {
				if o != origin {
					write(w, r, 403, "ORIGIN_NOT_ALLOWED", "Origin not allowed")
					return
				}
				w.WriteHeader(204)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

type rateEntry struct {
	count int
	reset time.Time
}

// RateLimit creates a bounded, in-memory limiter for sensitive endpoints. It is
// intentionally per instance; distributed deployments should use a shared edge
// limiter in addition to this defense-in-depth control.
func RateLimit(max int, window time.Duration, write ErrorWriter) func(http.Handler) http.Handler {
	var mu sync.Mutex
	entries := make(map[string]rateEntry)
	lastSweep := time.Now()
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			now := time.Now()
			host, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				host = r.RemoteAddr
			}
			mu.Lock()
			if now.Sub(lastSweep) >= window {
				for key, entry := range entries {
					if !now.Before(entry.reset) {
						delete(entries, key)
					}
				}
				lastSweep = now
			}
			entry := entries[host]
			if entry.reset.IsZero() || !now.Before(entry.reset) {
				entry = rateEntry{reset: now.Add(window)}
			}
			entry.count++
			entries[host] = entry
			limited := entry.count > max
			retryAfter := time.Until(entry.reset)
			mu.Unlock()
			if limited {
				seconds := maxInt64(1, int64(retryAfter.Round(time.Second)/time.Second))
				w.Header().Set("Retry-After", strconv.FormatInt(seconds, 10))
				write(w, r, http.StatusTooManyRequests, "RATE_LIMITED", "Too many attempts; try again later")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
func Recover(logger *slog.Logger, write ErrorWriter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if v := recover(); v != nil {
					logger.Error("panic recovered", "request_id", RequestIDValue(r), "error", v, "stack", string(debug.Stack()))
					write(w, r, 500, "INTERNAL_ERROR", "Request could not be completed")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			logger.Info("http request", "request_id", RequestIDValue(r), "method", r.Method, "path", r.URL.Path, "duration_ms", time.Since(start).Milliseconds())
		})
	}
}
func Authenticate(store *repository.Store, cookie, secret string, write ErrorWriter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, e := r.Cookie(cookie)
			if e != nil || c.Value == "" {
				write(w, r, 401, "UNAUTHORIZED", "Authentication required")
				return
			}
			u, e := store.UserBySession(r.Context(), auth.TokenHash(c.Value, secret))
			if e != nil {
				write(w, r, 401, "UNAUTHORIZED", "Session invalid or expired")
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userKey, u)))
		})
	}
}
func Permission(key string, write ErrorWriter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u, ok := User(r.Context())
			if !ok || !contains(u.Permissions, key) {
				write(w, r, 403, "FORBIDDEN", "Permission required: "+key)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
func BearerToken(expected string, write ErrorWriter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			provided := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if expected == "" || len(provided) != len(expected) || subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) != 1 {
				write(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Valid bearer token required")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
func contains(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}
