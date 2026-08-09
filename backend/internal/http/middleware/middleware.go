package middleware

import (
	"context"
	"github.com/google/uuid"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/auth"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/domain"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/repository"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
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
func contains(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}
