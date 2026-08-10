package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/ai"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/config"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/domain"
	appmw "github.com/thiagomontozo/lawoffice-os/backend/internal/http/middleware"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/jobs"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/realtime"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/repository"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/service"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/storage"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const SessionCookie = "lawoffice_session"

type Handler struct {
	Store   *repository.Store
	Service *service.Service
	Storage storage.ObjectStorage
	DB      *pgxpool.Pool
	Config  config.Config
	Logger  *slog.Logger
	Hub     *realtime.Hub
	Jobs    *jobs.Queue
	AI      *ai.Workspace
}

func New(r *repository.Store, s *service.Service, o storage.ObjectStorage, db *pgxpool.Pool, c config.Config, l *slog.Logger, h *realtime.Hub, queue *jobs.Queue, workspaces ...*ai.Workspace) *Handler {
	var workspace *ai.Workspace
	if len(workspaces) > 0 {
		workspace = workspaces[0]
	}
	return &Handler{Store: r, Service: s, Storage: o, DB: db, Config: c, Logger: l, Hub: h, Jobs: queue, AI: workspace}
}

type envelope struct {
	Error struct {
		Code      string `json:"code"`
		Message   string `json:"message"`
		RequestID string `json:"requestId"`
	} `json:"error"`
}

func WriteError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	var e envelope
	e.Error.Code = code
	e.Error.Message = message
	e.Error.RequestID = appmw.RequestIDValue(r)
	writeJSON(w, status, e)
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func decode(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		return e
	}
	if e := d.Decode(&struct{}{}); !errors.Is(e, io.EOF) {
		return errors.New("one JSON object required")
	}
	return nil
}
func (h *Handler) fail(w http.ResponseWriter, r *http.Request, e error) {
	switch {
	case errors.Is(e, service.ErrValidation), errors.Is(e, repository.ErrInvalid):
		WriteError(w, r, 422, "VALIDATION_ERROR", e.Error())
	case errors.Is(e, repository.ErrNotFound):
		WriteError(w, r, 404, "NOT_FOUND", "Resource not found")
	case errors.Is(e, repository.ErrConflict):
		WriteError(w, r, 409, "CONFLICT", "Resource already exists or is in use")
	case errors.Is(e, repository.ErrForbidden):
		WriteError(w, r, 403, "FORBIDDEN", "Access denied")
	case errors.Is(e, ai.ErrDisabled):
		WriteError(w, r, 503, "AI_DISABLED", "Matter AI Workspace is not enabled")
	case errors.Is(e, ai.ErrNoSources):
		WriteError(w, r, 409, "AI_SOURCES_UNAVAILABLE", "No extracted document sources are available for this Matter")
	case ai.IsOperationalError(e):
		h.Logger.Warn("AI provider unavailable", "request_id", appmw.RequestIDValue(r), "error_type", fmt.Sprintf("%T", e))
		WriteError(w, r, 502, "AI_PROVIDER_UNAVAILABLE", "The AI provider could not complete the request")
	default:
		h.Logger.Error("request failed", "request_id", appmw.RequestIDValue(r), "error", e)
		WriteError(w, r, 500, "INTERNAL_ERROR", "Request could not be completed")
	}
}
func required(v string, max int) (string, bool) {
	v = strings.TrimSpace(v)
	return v, v != "" && len([]rune(v)) <= max
}
func optional(v *string, max int) (*string, bool) {
	if v == nil {
		return nil, true
	}
	x := strings.TrimSpace(*v)
	if x == "" {
		return nil, true
	}
	return &x, len([]rune(x)) <= max
}
func validID(v string) bool { _, e := uuid.Parse(v); return e == nil }
func optionalID(v string) (*string, bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil, true
	}
	return &v, validID(v)
}
func page(r *http.Request) (int, int, bool) {
	p := number(r, "page", 1)
	s := number(r, "pageSize", 30)
	return p, s, p > 0 && s > 0 && s <= 100
}
func number(r *http.Request, key string, d int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return d
	}
	n, e := strconv.Atoi(v)
	if e != nil {
		return -1
	}
	return n
}
func user(r *http.Request) domain.User { u, _ := appmw.User(r.Context()); return u }
func auditContext(r *http.Request) (string, string) {
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	return ip, r.UserAgent()
}
func date(v string) (time.Time, bool)     { x, e := time.Parse("2006-01-02", v); return x, e == nil }
func dateTime(v string) (time.Time, bool) { x, e := time.Parse(time.RFC3339, v); return x, e == nil }
func bad(w http.ResponseWriter, r *http.Request, m string) {
	WriteError(w, r, 400, "VALIDATION_ERROR", m)
}
