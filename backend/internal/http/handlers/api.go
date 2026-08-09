package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/auth"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/domain"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/realtime"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/repository"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/service"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{1,61}[a-z0-9])?$`)
var emailPattern = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

const portalCookie = "lawoffice_portal_session"

func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := contextWithTimeout(r, 2*time.Second)
	defer cancel()
	if err := h.DB.Ping(ctx); err != nil {
		WriteError(w, r, 503, "NOT_READY", "Database unavailable")
		return
	}
	if err := h.Storage.Health(ctx); err != nil {
		WriteError(w, r, 503, "NOT_READY", "Object storage unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

type setupInput struct {
	LegalName      string  `json:"legalName"`
	DisplayName    string  `json:"displayName"`
	Slug           string  `json:"slug"`
	Email          string  `json:"email"`
	Phone          *string `json:"phone"`
	Website        *string `json:"website"`
	Timezone       string  `json:"timezone"`
	Locale         string  `json:"locale"`
	AdminName      string  `json:"adminName"`
	AdminEmail     string  `json:"adminEmail"`
	Password       string  `json:"password"`
	PrimaryColor   string  `json:"primaryColor"`
	SecondaryColor string  `json:"secondaryColor"`
	AccentColor    string  `json:"accentColor"`
}

func (h *Handler) Setup(w http.ResponseWriter, r *http.Request) {
	var in setupInput
	if err := decode(w, r, &in); err != nil {
		bad(w, r, "Invalid setup payload")
		return
	}
	legal, ok := required(in.LegalName, 160)
	if !ok {
		bad(w, r, "Legal name is required")
		return
	}
	display, ok := required(in.DisplayName, 120)
	if !ok {
		bad(w, r, "Display name is required")
		return
	}
	slug := strings.ToLower(strings.TrimSpace(in.Slug))
	if !slugPattern.MatchString(slug) {
		bad(w, r, "Slug must use lowercase letters, numbers and hyphens")
		return
	}
	admin, ok := required(in.AdminName, 120)
	if !ok || !emailPattern.MatchString(in.AdminEmail) || !emailPattern.MatchString(in.Email) {
		bad(w, r, "Administrator name and valid email are required")
		return
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		bad(w, r, err.Error())
		return
	}
	if in.Timezone == "" {
		in.Timezone = h.Config.Timezone
	}
	if in.Locale == "" {
		in.Locale = h.Config.Locale
	}
	firm := domain.Firm{LegalName: legal, DisplayName: display, Slug: slug, Email: strings.ToLower(strings.TrimSpace(in.Email)), Phone: in.Phone, Website: in.Website, Timezone: in.Timezone, Locale: in.Locale}
	u, err := h.Store.Setup(r.Context(), firm, admin, strings.ToLower(strings.TrimSpace(in.AdminEmail)), hash)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	if err = h.issueSession(w, r, u); err != nil {
		h.fail(w, r, err)
		return
	}
	colors := []string{in.PrimaryColor, in.SecondaryColor, in.AccentColor}
	fallback := []string{"#17324D", "#334E68", "#C9A227"}
	for i := range colors {
		if colors[i] == "" {
			colors[i] = fallback[i]
		}
	}
	_, _ = h.Store.UpdateBranding(r.Context(), u.FirmID, u.ID, domain.Branding{SystemTitle: "Legal Workspace", FirmDisplayName: display, PrimaryColor: colors[0], SecondaryColor: colors[1], AccentColor: colors[2], SidebarStyle: "dark", BorderRadiusStyle: "soft", ClientPortalTitle: "Portal do Cliente", ClientPortalWelcomeText: "Acompanhe as informações compartilhadas pelo escritório."})
	writeJSON(w, http.StatusCreated, map[string]any{"user": u, "next": "/app"})
}

type loginInput struct {
	FirmSlug string `json:"firmSlug"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var in loginInput
	if decode(w, r, &in) != nil {
		bad(w, r, "Invalid credentials payload")
		return
	}
	u, hash, err := h.Store.Credentials(r.Context(), strings.ToLower(strings.TrimSpace(in.FirmSlug)), strings.ToLower(strings.TrimSpace(in.Email)))
	if err != nil || !auth.CheckPassword(hash, in.Password) {
		WriteError(w, r, 401, "INVALID_CREDENTIALS", "Firm, email or password is incorrect")
		return
	}
	if err = h.issueSession(w, r, u); err != nil {
		h.fail(w, r, err)
		return
	}
	ip, ua := auditContext(r)
	_ = h.Store.Audit(r.Context(), u.FirmID, u.ID, "login.success", "session", nil, map[string]any{}, ip, ua)
	writeJSON(w, 200, map[string]any{"user": u})
}
func (h *Handler) issueSession(w http.ResponseWriter, r *http.Request, u domain.User) error {
	token, hash, err := auth.NewToken(h.Config.SessionSecret)
	if err != nil {
		return err
	}
	if err = h.Store.CreateSession(r.Context(), u, hash, time.Now().Add(h.Config.SessionTTL)); err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{Name: SessionCookie, Value: token, Path: "/", HttpOnly: true, Secure: h.Config.Environment == "production", SameSite: http.SameSiteLaxMode, MaxAge: int(h.Config.SessionTTL.Seconds())})
	return nil
}
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if c, e := r.Cookie(SessionCookie); e == nil {
		_ = h.Store.DeleteSession(r.Context(), auth.TokenHash(c.Value, h.Config.SessionSecret))
	}
	http.SetCookie(w, &http.Cookie{Name: SessionCookie, Path: "/", HttpOnly: true, Secure: h.Config.Environment == "production", SameSite: http.SameSiteLaxMode, MaxAge: -1})
	w.WriteHeader(204)
}
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	u := user(r)
	firm, err := h.Store.Firm(r.Context(), u.FirmID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	branding, _ := h.Store.Branding(r.Context(), u.FirmID)
	writeJSON(w, 200, map[string]any{"user": u, "firm": firm, "branding": branding})
}
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Current string `json:"currentPassword"`
		Next    string `json:"newPassword"`
	}
	if decode(w, r, &in) != nil {
		bad(w, r, "Invalid payload")
		return
	}
	u := user(r)
	hash, e := h.Store.PasswordHash(r.Context(), u.FirmID, u.ID)
	if e != nil || !auth.CheckPassword(hash, in.Current) {
		WriteError(w, r, 401, "INVALID_CREDENTIALS", "Current password is incorrect")
		return
	}
	next, e := auth.HashPassword(in.Next)
	if e != nil {
		bad(w, r, e.Error())
		return
	}
	if e = h.Store.ChangePassword(r.Context(), u.FirmID, u.ID, next); e != nil {
		h.fail(w, r, e)
		return
	}
	_ = h.Store.RevokeUserSessions(r.Context(), u.FirmID, u.ID)
	w.WriteHeader(204)
}

func (h *Handler) PublicBranding(w http.ResponseWriter, r *http.Request) {
	b, e := h.Store.BrandingBySlug(r.Context(), chi.URLParam(r, "slug"))
	if e != nil {
		h.fail(w, r, e)
		return
	}
	writeJSON(w, 200, b)
}
func (h *Handler) PublicBrandAsset(w http.ResponseWriter, r *http.Request) {
	key, e := h.Store.BrandAssetKeyBySlug(r.Context(), chi.URLParam(r, "slug"), chi.URLParam(r, "kind"))
	if e != nil {
		h.fail(w, r, e)
		return
	}
	reader, e := h.Storage.Open(r.Context(), key)
	if e != nil {
		h.fail(w, r, e)
		return
	}
	defer reader.Close()
	w.Header().Set("Content-Type", mime.TypeByExtension(filepath.Ext(key)))
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = io.Copy(w, reader)
}
func (h *Handler) Branding(w http.ResponseWriter, r *http.Request) {
	b, e := h.Store.Branding(r.Context(), user(r).FirmID)
	if e != nil {
		h.fail(w, r, e)
		return
	}
	writeJSON(w, 200, b)
}
func (h *Handler) UpdateBranding(w http.ResponseWriter, r *http.Request) {
	var b domain.Branding
	if decode(w, r, &b) != nil {
		bad(w, r, "Invalid branding")
		return
	}
	u := user(r)
	x, e := h.Service.UpdateBranding(r.Context(), u.FirmID, u.ID, b)
	if e != nil {
		h.fail(w, r, e)
		return
	}
	h.audit(r, "branding.updated", "branding", &u.FirmID, map[string]any{"colorsChanged": true})
	h.Hub.Publish(u.FirmID, realtime.Event{Type: "branding.updated", ResourceType: "firm", ResourceID: u.FirmID})
	writeJSON(w, 200, x)
}
func (h *Handler) UploadBrandAsset(w http.ResponseWriter, r *http.Request) {
	if e := r.ParseMultipartForm(h.Config.MaxUpload); e != nil {
		bad(w, r, "Invalid image upload")
		return
	}
	f, head, e := r.FormFile("file")
	if e != nil {
		bad(w, r, "File is required")
		return
	}
	_ = f.Close()
	u := user(r)
	if e = h.Service.UploadBrandAsset(r.Context(), u.FirmID, u.ID, chi.URLParam(r, "kind"), head); e != nil {
		h.fail(w, r, e)
		return
	}
	h.audit(r, "branding.asset_updated", "branding", &u.FirmID, map[string]any{"kind": chi.URLParam(r, "kind")})
	w.WriteHeader(204)
}
func (h *Handler) BrandAsset(w http.ResponseWriter, r *http.Request) {
	firmID := user(r).FirmID
	key, e := h.Store.BrandAssetKey(r.Context(), firmID, chi.URLParam(r, "kind"))
	if e != nil {
		h.fail(w, r, e)
		return
	}
	reader, e := h.Storage.Open(r.Context(), key)
	if e != nil {
		h.fail(w, r, e)
		return
	}
	defer reader.Close()
	w.Header().Set("Content-Type", mime.TypeByExtension(filepath.Ext(key)))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "private, max-age=300")
	_, _ = io.Copy(w, reader)
}

func (h *Handler) Users(w http.ResponseWriter, r *http.Request) {
	items, e := h.Store.Users(r.Context(), user(r).FirmID)
	if e != nil {
		h.fail(w, r, e)
		return
	}
	writeJSON(w, 200, map[string]any{"items": items})
}
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name     string   `json:"name"`
		Email    string   `json:"email"`
		Password string   `json:"password"`
		RoleIDs  []string `json:"roleIds"`
	}
	if decode(w, r, &in) != nil {
		bad(w, r, "Invalid user")
		return
	}
	name, ok := required(in.Name, 120)
	if !ok || !emailPattern.MatchString(in.Email) {
		bad(w, r, "Valid name and email are required")
		return
	}
	hash, e := auth.HashPassword(in.Password)
	if e != nil {
		bad(w, r, e.Error())
		return
	}
	u := user(r)
	created, e := h.Store.CreateUser(r.Context(), u.FirmID, name, strings.ToLower(in.Email), hash, in.RoleIDs)
	if e != nil {
		h.fail(w, r, e)
		return
	}
	h.audit(r, "user.created", "user", &created.ID, map[string]any{"email": created.Email})
	writeJSON(w, 201, created)
}
func (h *Handler) SetUserActive(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !validID(id) {
		bad(w, r, "Invalid user ID")
		return
	}
	var in struct {
		Active bool `json:"active"`
	}
	if decode(w, r, &in) != nil {
		bad(w, r, "Invalid status")
		return
	}
	u := user(r)
	if id == u.ID && !in.Active {
		bad(w, r, "You cannot disable your own account")
		return
	}
	if e := h.Store.SetUserActive(r.Context(), u.FirmID, id, in.Active); e != nil {
		h.fail(w, r, e)
		return
	}
	h.audit(r, map[bool]string{true: "user.reactivated", false: "user.disabled"}[in.Active], "user", &id, map[string]any{})
	w.WriteHeader(204)
}
func (h *Handler) Roles(w http.ResponseWriter, r *http.Request) {
	x, e := h.Store.Roles(r.Context(), user(r).FirmID)
	if e != nil {
		h.fail(w, r, e)
		return
	}
	writeJSON(w, 200, map[string]any{"items": x})
}
func (h *Handler) CreateRole(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name        string   `json:"name"`
		Description *string  `json:"description"`
		Permissions []string `json:"permissions"`
	}
	if decode(w, r, &in) != nil {
		bad(w, r, "Invalid role")
		return
	}
	name, ok := required(in.Name, 80)
	if !ok {
		bad(w, r, "Role name is required")
		return
	}
	u := user(r)
	x, e := h.Store.CreateRole(r.Context(), u.FirmID, name, in.Description, in.Permissions)
	if e != nil {
		h.fail(w, r, e)
		return
	}
	h.audit(r, "role.created", "role", &x.ID, map[string]any{"name": x.Name})
	writeJSON(w, 201, x)
}

func (h *Handler) Permissions(w http.ResponseWriter, r *http.Request) {
	items, err := h.Store.Permissions(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var in struct {
		Name        string   `json:"name"`
		Description *string  `json:"description"`
		Permissions []string `json:"permissions"`
	}
	if !validID(id) || decode(w, r, &in) != nil {
		bad(w, r, "Invalid role")
		return
	}
	name, ok := required(in.Name, 80)
	if !ok || len(in.Permissions) > 100 {
		bad(w, r, "Role name and a valid permission set are required")
		return
	}
	u := user(r)
	role, err := h.Store.UpdateRole(r.Context(), u.FirmID, id, name, in.Description, in.Permissions)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	h.audit(r, "role.updated", "role", &id, map[string]any{"permissionCount": len(in.Permissions)})
	writeJSON(w, http.StatusOK, role)
}

func (h *Handler) UpdateUserRoles(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var in struct {
		RoleIDs []string `json:"roleIds"`
	}
	if !validID(id) || decode(w, r, &in) != nil || len(in.RoleIDs) == 0 || len(in.RoleIDs) > 20 {
		bad(w, r, "At least one valid role is required")
		return
	}
	for _, roleID := range in.RoleIDs {
		if !validID(roleID) {
			bad(w, r, "Invalid role ID")
			return
		}
	}
	u := user(r)
	if id == u.ID {
		bad(w, r, "You cannot change your own roles")
		return
	}
	if err := h.Store.UpdateUserRoles(r.Context(), u.FirmID, id, in.RoleIDs); err != nil {
		h.fail(w, r, err)
		return
	}
	h.audit(r, "user.roles_updated", "user", &id, map[string]any{"roleCount": len(in.RoleIDs)})
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Clients(w http.ResponseWriter, r *http.Request) {
	p, s, ok := page(r)
	if !ok {
		bad(w, r, "Invalid pagination")
		return
	}
	x, e := h.Store.Clients(r.Context(), user(r).FirmID, strings.TrimSpace(r.URL.Query().Get("q")), s, (p-1)*s)
	if e != nil {
		h.fail(w, r, e)
		return
	}
	writeJSON(w, 200, map[string]any{"items": x, "page": p, "pageSize": s})
}
func (h *Handler) CreateClient(w http.ResponseWriter, r *http.Request) {
	var x domain.Client
	if decode(w, r, &x) != nil {
		bad(w, r, "Invalid client")
		return
	}
	name, ok := required(x.Name, 180)
	if !ok || !(x.Type == "person" || x.Type == "company") {
		bad(w, r, "Valid name and type are required")
		return
	}
	x.Name = name
	u := user(r)
	created, e := h.Store.CreateClient(r.Context(), u.FirmID, x)
	if e != nil {
		h.fail(w, r, e)
		return
	}
	h.audit(r, "client.created", "client", &created.ID, map[string]any{"type": created.Type})
	writeJSON(w, 201, created)
}

func (h *Handler) UpdateClient(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var x domain.Client
	if !validID(id) || decode(w, r, &x) != nil {
		bad(w, r, "Invalid client")
		return
	}
	name, ok := required(x.Name, 180)
	if !ok || (x.Type != "person" && x.Type != "company") {
		bad(w, r, "Valid name and type are required")
		return
	}
	x.Name = name
	u := user(r)
	updated, err := h.Store.UpdateClient(r.Context(), u.FirmID, id, x)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	h.audit(r, "client.updated", "client", &id, map[string]any{"type": updated.Type})
	writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) ArchiveClient(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !validID(id) {
		bad(w, r, "Invalid client ID")
		return
	}
	u := user(r)
	if err := h.Store.ArchiveClient(r.Context(), u.FirmID, u.ID, id); err != nil {
		h.fail(w, r, err)
		return
	}
	h.audit(r, "client.archived", "client", &id, map[string]any{})
	w.WriteHeader(http.StatusNoContent)
}
func (h *Handler) Client(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !validID(id) {
		bad(w, r, "Invalid client ID")
		return
	}
	x, e := h.Store.Client(r.Context(), user(r).FirmID, id)
	if e != nil {
		h.fail(w, r, e)
		return
	}
	writeJSON(w, 200, x)
}

func (h *Handler) Matters(w http.ResponseWriter, r *http.Request) {
	p, s, ok := page(r)
	if !ok {
		bad(w, r, "Invalid pagination")
		return
	}
	u := user(r)
	x, e := h.Store.Matters(r.Context(), u.FirmID, u.ID, r.URL.Query().Get("q"), r.URL.Query().Get("status"), r.URL.Query().Get("priority"), r.URL.Query().Get("type"), r.URL.Query().Get("archived") == "true", s, (p-1)*s)
	if e != nil {
		h.fail(w, r, e)
		return
	}
	writeJSON(w, 200, map[string]any{"items": x, "page": p, "pageSize": s})
}
func (h *Handler) CreateMatter(w http.ResponseWriter, r *http.Request) {
	var x domain.Matter
	if decode(w, r, &x) != nil {
		bad(w, r, "Invalid matter")
		return
	}
	title, ok := required(x.Title, 220)
	internal, internalOK := required(x.InternalNumber, 80)
	validTypes := map[string]bool{"judicial_process": true, "administrative_process": true, "legal_consultation": true, "contract": true, "advisory": true, "arbitration": true, "extrajudicial": true, "internal_legal_project": true, "other": true}
	if !ok || !internalOK || !validTypes[x.Type] {
		bad(w, r, "Title, internal number and valid type are required")
		return
	}
	x.Title = title
	x.InternalNumber = internal
	if x.Priority == "" {
		x.Priority = "normal"
	}
	if x.Confidentiality == "" {
		x.Confidentiality = "normal"
	}
	if x.Status == "" {
		x.Status = "draft"
	}
	if x.OpenedAt.IsZero() {
		x.OpenedAt = time.Now()
	}
	if !map[string]bool{"draft": true, "active": true, "on_hold": true, "closing": true}[x.Status] || !map[string]bool{"low": true, "normal": true, "high": true, "critical": true}[x.Priority] || !map[string]bool{"normal": true, "team_only": true, "partners_only": true, "restricted": true}[x.Confidentiality] {
		bad(w, r, "Invalid status, priority or confidentiality")
		return
	}
	u := user(r)
	created, e := h.Store.CreateMatter(r.Context(), u.FirmID, u.ID, x)
	if e != nil {
		h.fail(w, r, e)
		return
	}
	h.audit(r, "matter.created", "matter", &created.ID, map[string]any{"type": created.Type})
	h.Hub.Publish(u.FirmID, realtime.Event{Type: "matter.created", ResourceType: "matter", ResourceID: created.ID})
	writeJSON(w, 201, created)
}
func (h *Handler) Matter(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !validID(id) {
		bad(w, r, "Invalid matter ID")
		return
	}
	u := user(r)
	x, e := h.Service.MatterDetail(r.Context(), u.FirmID, u.ID, id)
	if e != nil {
		h.fail(w, r, e)
		return
	}
	writeJSON(w, 200, x)
}
func (h *Handler) ConflictCheck(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name         string   `json:"name"`
		Document     string   `json:"document"`
		RelatedNames []string `json:"relatedNames"`
	}
	if decode(w, r, &in) != nil {
		bad(w, r, "Invalid conflict query")
		return
	}
	term := strings.TrimSpace(strings.Join(append([]string{in.Name}, in.RelatedNames...), " "))
	if len(term) < 3 && len(strings.TrimSpace(in.Document)) < 3 {
		bad(w, r, "Provide a name or document")
		return
	}
	x, e := h.Store.ConflictCheck(r.Context(), user(r).FirmID, term, strings.TrimSpace(in.Document))
	if e != nil {
		h.fail(w, r, e)
		return
	}
	writeJSON(w, 200, x)
}

func (h *Handler) Documents(w http.ResponseWriter, r *http.Request) {
	p, s, ok := page(r)
	if !ok {
		bad(w, r, "Invalid pagination")
		return
	}
	mid, ok := optionalID(r.URL.Query().Get("matterId"))
	if !ok {
		bad(w, r, "Invalid Matter ID")
		return
	}
	u := user(r)
	x, e := h.Store.Documents(r.Context(), u.FirmID, u.ID, r.URL.Query().Get("q"), mid, s, (p-1)*s)
	if e != nil {
		h.fail(w, r, e)
		return
	}
	writeJSON(w, 200, map[string]any{"items": x})
}
func (h *Handler) DeletedDocuments(w http.ResponseWriter, r *http.Request) {
	u := user(r)
	items, err := h.Store.DeletedDocuments(r.Context(), u.FirmID, u.ID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}
func (h *Handler) UploadDocument(w http.ResponseWriter, r *http.Request) {
	if e := r.ParseMultipartForm(h.Config.MaxUpload); e != nil {
		bad(w, r, "Invalid upload")
		return
	}
	f, head, e := r.FormFile("file")
	if e != nil {
		bad(w, r, "File is required")
		return
	}
	_ = f.Close()
	matterID, ok := optionalID(r.FormValue("matterId"))
	if !ok {
		bad(w, r, "Invalid Matter ID")
		return
	}
	clientID, ok := optionalID(r.FormValue("clientId"))
	if !ok {
		bad(w, r, "Invalid client ID")
		return
	}
	u := user(r)
	x, e := h.Service.UploadDocument(r.Context(), service.Upload{FirmID: u.FirmID, UserID: u.ID, MatterID: matterID, ClientID: clientID, Title: r.FormValue("title"), Description: optionalForm(r.FormValue("description")), Category: r.FormValue("category"), ClientVisible: r.FormValue("clientVisible") == "true", Header: head})
	if e != nil {
		h.fail(w, r, e)
		return
	}
	h.audit(r, "document.uploaded", "document", &x.ID, map[string]any{"matterId": matterID, "mime": x.MimeType})
	h.Hub.Publish(u.FirmID, realtime.Event{Type: "document.uploaded", ResourceType: "document", ResourceID: x.ID})
	writeJSON(w, 201, x)
}
func (h *Handler) DownloadDocument(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !validID(id) {
		bad(w, r, "Invalid document ID")
		return
	}
	var version *int
	if v := r.URL.Query().Get("version"); v != "" {
		n, e := strconv.Atoi(v)
		if e != nil || n < 1 {
			bad(w, r, "Invalid version")
			return
		}
		version = &n
	}
	u := user(r)
	d, reader, e := h.Service.OpenDocument(r.Context(), u.FirmID, u.ID, id, version)
	if e != nil {
		h.fail(w, r, e)
		return
	}
	defer reader.Close()
	h.audit(r, "document.downloaded", "document", &id, map[string]any{"version": d.VersionNumber})
	w.Header().Set("Content-Type", d.MimeType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", safeHeaderName(d.OriginalFileName)))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = io.Copy(w, reader)
}

func (h *Handler) DeleteDocument(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !validID(id) {
		bad(w, r, "Invalid document ID")
		return
	}
	u := user(r)
	if err := h.Service.DeleteDocument(r.Context(), u.FirmID, u.ID, id); err != nil {
		h.fail(w, r, err)
		return
	}
	h.audit(r, "document.deleted", "document", &id, map[string]any{"retention": "metadata and versions preserved"})
	h.Hub.Publish(u.FirmID, realtime.Event{Type: "document.deleted", ResourceType: "document", ResourceID: id})
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) RestoreDocument(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !validID(id) {
		bad(w, r, "Invalid document ID")
		return
	}
	u := user(r)
	if err := h.Service.RestoreDocument(r.Context(), u.FirmID, u.ID, id); err != nil {
		h.fail(w, r, err)
		return
	}
	h.audit(r, "document.restored", "document", &id, map[string]any{})
	h.Hub.Publish(u.FirmID, realtime.Event{Type: "document.restored", ResourceType: "document", ResourceID: id})
	w.WriteHeader(http.StatusNoContent)
}
func (h *Handler) Versions(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !validID(id) {
		bad(w, r, "Invalid document ID")
		return
	}
	u := user(r)
	if _, _, e := h.Store.DocumentVersion(r.Context(), u.FirmID, u.ID, id, nil); e != nil {
		h.fail(w, r, e)
		return
	}
	x, e := h.Store.Versions(r.Context(), u.FirmID, id)
	if e != nil {
		h.fail(w, r, e)
		return
	}
	writeJSON(w, 200, map[string]any{"items": x})
}
func (h *Handler) AddVersion(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !validID(id) {
		bad(w, r, "Invalid document ID")
		return
	}
	if e := r.ParseMultipartForm(h.Config.MaxUpload); e != nil {
		bad(w, r, "Invalid upload")
		return
	}
	f, head, e := r.FormFile("file")
	if e != nil {
		bad(w, r, "File is required")
		return
	}
	_ = f.Close()
	u := user(r)
	x, e := h.Service.AddVersion(r.Context(), u.FirmID, u.ID, id, head, optionalForm(r.FormValue("notes")))
	if e != nil {
		h.fail(w, r, e)
		return
	}
	h.audit(r, "document.version_added", "document", &id, map[string]any{"version": x.VersionNumber})
	writeJSON(w, 201, x)
}

func (h *Handler) Deadlines(w http.ResponseWriter, r *http.Request) {
	u := user(r)
	mid, ok := optionalID(r.URL.Query().Get("matterId"))
	if !ok {
		bad(w, r, "Invalid Matter ID")
		return
	}
	x, e := h.Store.Deadlines(r.Context(), u.FirmID, u.ID, mid, r.URL.Query().Get("mine") == "true")
	if e != nil {
		h.fail(w, r, e)
		return
	}
	writeJSON(w, 200, map[string]any{"items": x})
}
func (h *Handler) CreateDeadline(w http.ResponseWriter, r *http.Request) {
	var x domain.Deadline
	if decode(w, r, &x) != nil || !validID(x.MatterID) {
		bad(w, r, "Invalid deadline")
		return
	}
	if _, ok := required(x.Title, 200); !ok || x.DueAt.IsZero() {
		bad(w, r, "Title and due date are required")
		return
	}
	if x.Priority == "" {
		x.Priority = "normal"
	}
	u := user(r)
	allowed, e := h.Store.CanAccessMatter(r.Context(), u.FirmID, u.ID, x.MatterID, "write")
	if e != nil || !allowed {
		h.fail(w, r, repository.ErrForbidden)
		return
	}
	created, e := h.Store.CreateDeadline(r.Context(), u.FirmID, u.ID, x)
	if e != nil {
		h.fail(w, r, e)
		return
	}
	h.Hub.Publish(u.FirmID, realtime.Event{Type: "deadline.created", ResourceType: "deadline", ResourceID: created.ID})
	writeJSON(w, 201, created)
}

func (h *Handler) UpdateDeadlineStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var in struct {
		Status string `json:"status"`
	}
	if !validID(id) || decode(w, r, &in) != nil || !map[string]bool{"open": true, "completed": true, "cancelled": true}[in.Status] {
		bad(w, r, "Invalid deadline status")
		return
	}
	u := user(r)
	matterID, err := h.Store.DeadlineMatter(r.Context(), u.FirmID, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	if !h.canMatter(w, r, u, matterID, "write") {
		return
	}
	if err = h.Store.UpdateDeadlineStatus(r.Context(), u.FirmID, u.ID, id, in.Status); err != nil {
		h.fail(w, r, err)
		return
	}
	h.audit(r, "deadline.updated", "deadline", &id, map[string]any{"status": in.Status})
	h.Hub.Publish(u.FirmID, realtime.Event{Type: "deadline.updated", ResourceType: "deadline", ResourceID: id})
	w.WriteHeader(http.StatusNoContent)
}
func (h *Handler) Tasks(w http.ResponseWriter, r *http.Request) {
	u := user(r)
	mid, ok := optionalID(r.URL.Query().Get("matterId"))
	if !ok {
		bad(w, r, "Invalid Matter ID")
		return
	}
	x, e := h.Store.Tasks(r.Context(), u.FirmID, u.ID, mid, r.URL.Query().Get("mine") == "true")
	if e != nil {
		h.fail(w, r, e)
		return
	}
	writeJSON(w, 200, map[string]any{"items": x})
}
func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	var x domain.Task
	if decode(w, r, &x) != nil {
		bad(w, r, "Invalid task")
		return
	}
	if _, ok := required(x.Title, 200); !ok {
		bad(w, r, "Title is required")
		return
	}
	if x.Status == "" {
		x.Status = "todo"
	}
	if x.Priority == "" {
		x.Priority = "normal"
	}
	u := user(r)
	if x.MatterID != nil {
		allowed, e := h.Store.CanAccessMatter(r.Context(), u.FirmID, u.ID, *x.MatterID, "write")
		if e != nil || !allowed {
			h.fail(w, r, repository.ErrForbidden)
			return
		}
	}
	created, e := h.Store.CreateTask(r.Context(), u.FirmID, u.ID, x)
	if e != nil {
		h.fail(w, r, e)
		return
	}
	h.Hub.Publish(u.FirmID, realtime.Event{Type: "task.created", ResourceType: "task", ResourceID: created.ID})
	writeJSON(w, 201, created)
}

func (h *Handler) UpdateTaskStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var in struct {
		Status string `json:"status"`
	}
	if !validID(id) || decode(w, r, &in) != nil || !map[string]bool{"todo": true, "in_progress": true, "blocked": true, "done": true, "cancelled": true}[in.Status] {
		bad(w, r, "Invalid task status")
		return
	}
	u := user(r)
	matterID, err := h.Store.TaskMatter(r.Context(), u.FirmID, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	if matterID != nil && !h.canMatter(w, r, u, *matterID, "write") {
		return
	}
	if err = h.Store.UpdateTaskStatus(r.Context(), u.FirmID, u.ID, id, in.Status); err != nil {
		h.fail(w, r, err)
		return
	}
	h.audit(r, "task.updated", "task", &id, map[string]any{"status": in.Status})
	h.Hub.Publish(u.FirmID, realtime.Event{Type: "task.updated", ResourceType: "task", ResourceID: id})
	w.WriteHeader(http.StatusNoContent)
}
func (h *Handler) Calendar(w http.ResponseWriter, r *http.Request) {
	from := time.Now().AddDate(0, -1, 0)
	to := time.Now().AddDate(0, 3, 0)
	if v := r.URL.Query().Get("from"); v != "" {
		if x, ok := date(v); ok {
			from = x
		} else {
			bad(w, r, "Invalid from date")
			return
		}
	}
	if v := r.URL.Query().Get("to"); v != "" {
		if x, ok := date(v); ok {
			to = x.Add(24 * time.Hour)
		} else {
			bad(w, r, "Invalid to date")
			return
		}
	}
	u := user(r)
	x, e := h.Store.Hearings(r.Context(), u.FirmID, u.ID, from, to)
	if e != nil {
		h.fail(w, r, e)
		return
	}
	writeJSON(w, 200, map[string]any{"hearings": x})
}
func (h *Handler) Workflows(w http.ResponseWriter, r *http.Request) {
	x, e := h.Store.Workflows(r.Context(), user(r).FirmID)
	if e != nil {
		h.fail(w, r, e)
		return
	}
	writeJSON(w, 200, map[string]any{"items": x})
}
func (h *Handler) Archive(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var in struct {
		Reason  string `json:"reason"`
		Outcome string `json:"outcome"`
		Summary string `json:"summary"`
		Force   bool   `json:"force"`
	}
	if !validID(id) || decode(w, r, &in) != nil {
		bad(w, r, "Invalid archive request")
		return
	}
	u := user(r)
	x, e := h.Service.Archive(r.Context(), u.FirmID, u.ID, id, in.Reason, in.Outcome, in.Summary, in.Force)
	if e != nil {
		h.fail(w, r, e)
		return
	}
	h.audit(r, "matter.archived", "matter", &id, map[string]any{"reason": in.Reason})
	writeJSON(w, 200, x)
}
func (h *Handler) Reopen(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var in struct {
		Reason string `json:"reason"`
	}
	if !validID(id) || decode(w, r, &in) != nil || strings.TrimSpace(in.Reason) == "" {
		bad(w, r, "Reason is required")
		return
	}
	u := user(r)
	if e := h.Store.ReopenMatter(r.Context(), u.FirmID, u.ID, id, in.Reason); e != nil {
		h.fail(w, r, e)
		return
	}
	h.audit(r, "matter.reopened", "matter", &id, map[string]any{"reason": in.Reason})
	w.WriteHeader(204)
}
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(q) < 2 || len([]rune(q)) > 120 {
		bad(w, r, "Search requires at least two characters")
		return
	}
	kind := strings.TrimSpace(r.URL.Query().Get("type"))
	if !map[string]bool{"": true, "matter": true, "client": true, "contact": true, "document": true}[kind] {
		bad(w, r, "Invalid search type")
		return
	}
	limit := 30
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 50 {
			bad(w, r, "Invalid search limit")
			return
		}
		limit = parsed
	}
	u := user(r)
	x, e := h.Store.Search(r.Context(), u.FirmID, u.ID, q, kind, limit)
	if e != nil {
		h.fail(w, r, e)
		return
	}
	writeJSON(w, 200, map[string]any{"items": x})
}
func (h *Handler) Notifications(w http.ResponseWriter, r *http.Request) {
	u := user(r)
	x, e := h.Store.Notifications(r.Context(), u.FirmID, u.ID)
	if e != nil {
		h.fail(w, r, e)
		return
	}
	writeJSON(w, 200, map[string]any{"items": x})
}
func (h *Handler) ReadNotification(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !validID(id) {
		bad(w, r, "Invalid notification ID")
		return
	}
	u := user(r)
	if e := h.Store.ReadNotification(r.Context(), u.FirmID, u.ID, id); e != nil {
		h.fail(w, r, e)
		return
	}
	w.WriteHeader(204)
}
func (h *Handler) AuditEvents(w http.ResponseWriter, r *http.Request) {
	x, e := h.Store.AuditEvents(r.Context(), user(r).FirmID)
	if e != nil {
		h.fail(w, r, e)
		return
	}
	writeJSON(w, 200, map[string]any{"items": x})
}
func (h *Handler) CommandCenter(w http.ResponseWriter, r *http.Request) {
	u := user(r)
	x, e := h.Store.CommandCenter(r.Context(), u.FirmID, u.ID)
	if e != nil {
		h.fail(w, r, e)
		return
	}
	writeJSON(w, 200, x)
}
func (h *Handler) Stream(w http.ResponseWriter, r *http.Request) {
	u := user(r)
	flusher, ok := w.(http.Flusher)
	if !ok {
		WriteError(w, r, 500, "SSE_UNAVAILABLE", "Streaming unsupported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	_ = http.NewResponseController(w).SetWriteDeadline(time.Time{})
	events, replay := h.Hub.Subscribe(r.Context(), u.FirmID, r.Header.Get("Last-Event-ID"))
	_, _ = fmt.Fprint(w, "event: connected\ndata: {}\n\n")
	flusher.Flush()
	writeEvent := func(event realtime.Event) bool {
		payload, err := json.Marshal(event)
		if err != nil {
			return false
		}
		_, err = fmt.Fprintf(w, "id: %s\nevent: %s\ndata: %s\n\n", event.ID, event.Type, payload)
		if err == nil {
			flusher.Flush()
		}
		return err == nil
	}
	for _, event := range replay {
		if !writeEvent(event) {
			return
		}
	}
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case event, open := <-events:
			if !open {
				return
			}
			if !writeEvent(event) {
				return
			}
		case <-heartbeat.C:
			if _, err := fmt.Fprint(w, ": keepalive\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func (h *Handler) PortalLogin(w http.ResponseWriter, r *http.Request) {
	var in loginInput
	if decode(w, r, &in) != nil {
		bad(w, r, "Invalid credentials payload")
		return
	}
	firmID, portalID, passwordHash, e := h.Store.PortalCredentials(r.Context(), strings.ToLower(strings.TrimSpace(in.FirmSlug)), strings.ToLower(strings.TrimSpace(in.Email)))
	if e != nil || !auth.CheckPassword(passwordHash, in.Password) {
		WriteError(w, r, 401, "INVALID_CREDENTIALS", "Firm, email or password is incorrect")
		return
	}
	token, tokenHash, e := auth.NewToken(h.Config.SessionSecret)
	if e != nil {
		h.fail(w, r, e)
		return
	}
	if e = h.Store.CreatePortalSession(r.Context(), firmID, portalID, tokenHash, time.Now().Add(h.Config.SessionTTL)); e != nil {
		h.fail(w, r, e)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: portalCookie, Value: token, Path: "/api/v1/portal", HttpOnly: true, Secure: h.Config.Environment == "production", SameSite: http.SameSiteLaxMode, MaxAge: int(h.Config.SessionTTL.Seconds())})
	writeJSON(w, 200, map[string]string{"status": "authenticated"})
}
func (h *Handler) PortalLogout(w http.ResponseWriter, r *http.Request) {
	if c, e := r.Cookie(portalCookie); e == nil {
		_ = h.Store.DeletePortalSession(r.Context(), auth.TokenHash(c.Value, h.Config.SessionSecret))
	}
	http.SetCookie(w, &http.Cookie{Name: portalCookie, Path: "/api/v1/portal", HttpOnly: true, Secure: h.Config.Environment == "production", SameSite: http.SameSiteLaxMode, MaxAge: -1})
	w.WriteHeader(204)
}
func (h *Handler) portalIdentity(r *http.Request) (string, string, error) {
	c, e := r.Cookie(portalCookie)
	if e != nil {
		return "", "", repository.ErrForbidden
	}
	return h.Store.PortalBySession(r.Context(), auth.TokenHash(c.Value, h.Config.SessionSecret))
}
func (h *Handler) PortalMatters(w http.ResponseWriter, r *http.Request) {
	firmID, portalID, e := h.portalIdentity(r)
	if e != nil {
		WriteError(w, r, 401, "UNAUTHORIZED", "Portal authentication required")
		return
	}
	items, e := h.Store.PortalMatters(r.Context(), firmID, portalID)
	if e != nil {
		h.fail(w, r, e)
		return
	}
	branding, _ := h.Store.Branding(r.Context(), firmID)
	writeJSON(w, 200, map[string]any{"items": items, "branding": branding})
}
func (h *Handler) PortalMatter(w http.ResponseWriter, r *http.Request) {
	firmID, portalID, e := h.portalIdentity(r)
	if e != nil {
		WriteError(w, r, 401, "UNAUTHORIZED", "Portal authentication required")
		return
	}
	id := chi.URLParam(r, "id")
	if !validID(id) {
		bad(w, r, "Invalid Matter ID")
		return
	}
	matter, e := h.Store.PortalMatter(r.Context(), firmID, portalID, id)
	if e != nil {
		h.fail(w, r, e)
		return
	}
	writeJSON(w, 200, matter)
}

func (h *Handler) CreatePortalUser(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ClientID  string   `json:"clientId"`
		Email     string   `json:"email"`
		Password  string   `json:"password"`
		MatterIDs []string `json:"matterIds"`
	}
	if decode(w, r, &in) != nil || !validID(in.ClientID) || !emailPattern.MatchString(in.Email) {
		bad(w, r, "Valid client, email and password are required")
		return
	}
	for _, id := range in.MatterIDs {
		if !validID(id) {
			bad(w, r, "Invalid Matter ID")
			return
		}
	}
	hash, e := auth.HashPassword(in.Password)
	if e != nil {
		bad(w, r, e.Error())
		return
	}
	u := user(r)
	id, e := h.Store.CreatePortalUser(r.Context(), u.FirmID, in.ClientID, strings.ToLower(in.Email), hash, in.MatterIDs)
	if e != nil {
		h.fail(w, r, e)
		return
	}
	h.audit(r, "portal.user_created", "portal_user", &id, map[string]any{"clientId": in.ClientID, "matterCount": len(in.MatterIDs)})
	writeJSON(w, 201, map[string]string{"id": id})
}

func (h *Handler) audit(r *http.Request, action, resourceType string, resourceID *string, metadata any) {
	u := user(r)
	ip, ua := auditContext(r)
	if e := h.Store.Audit(r.Context(), u.FirmID, u.ID, action, resourceType, resourceID, metadata, ip, ua); e != nil {
		h.Logger.Error("audit write failed", "action", action, "error", e)
	}
}
func optionalForm(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return &v
}
func safeHeaderName(v string) string {
	return strings.Map(func(r rune) rune {
		if r == '\r' || r == '\n' || r == '"' {
			return '_'
		}
		return r
	}, v)
}
func contextWithTimeout(r *http.Request, d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), d)
}
