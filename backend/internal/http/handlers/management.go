package handlers

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/domain"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/repository"
)

func (h *Handler) Contacts(w http.ResponseWriter, r *http.Request) {
	items, e := h.Store.Contacts(r.Context(), user(r).FirmID)
	if e != nil {
		h.fail(w, r, e)
		return
	}
	writeJSON(w, 200, map[string]any{"items": items})
}
func (h *Handler) CreateContact(w http.ResponseWriter, r *http.Request) {
	var x domain.Contact
	if decode(w, r, &x) != nil {
		bad(w, r, "Invalid contact")
		return
	}
	name, ok := required(x.Name, 180)
	if !ok {
		bad(w, r, "Contact name is required")
		return
	}
	x.Name = name
	if x.ClientID != nil && !validID(*x.ClientID) {
		bad(w, r, "Invalid client ID")
		return
	}
	if x.Type == "" {
		x.Type = "other"
	}
	u := user(r)
	created, e := h.Store.CreateContact(r.Context(), u.FirmID, x)
	if e != nil {
		h.fail(w, r, e)
		return
	}
	h.audit(r, "contact.created", "contact", &created.ID, map[string]any{"type": created.Type})
	writeJSON(w, 201, created)
}

func (h *Handler) UpdateContact(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var x domain.Contact
	if !validID(id) || decode(w, r, &x) != nil {
		bad(w, r, "Invalid contact")
		return
	}
	name, ok := required(x.Name, 180)
	if !ok {
		bad(w, r, "Contact name is required")
		return
	}
	if x.ClientID != nil && !validID(*x.ClientID) {
		bad(w, r, "Invalid client ID")
		return
	}
	x.Name = name
	if x.Type == "" {
		x.Type = "other"
	}
	u := user(r)
	updated, err := h.Store.UpdateContact(r.Context(), u.FirmID, id, x)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	h.audit(r, "contact.updated", "contact", &id, map[string]any{"type": updated.Type})
	writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) ArchiveContact(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !validID(id) {
		bad(w, r, "Invalid contact ID")
		return
	}
	if err := h.Store.ArchiveContact(r.Context(), user(r).FirmID, id); err != nil {
		h.fail(w, r, err)
		return
	}
	h.audit(r, "contact.archived", "contact", &id, map[string]any{})
	w.WriteHeader(http.StatusNoContent)
}
func (h *Handler) AddParty(w http.ResponseWriter, r *http.Request) {
	matterID := chi.URLParam(r, "id")
	var x domain.Party
	if !validID(matterID) || decode(w, r, &x) != nil {
		bad(w, r, "Invalid party")
		return
	}
	if _, ok := required(x.Name, 180); !ok || !map[string]bool{"client": true, "opposing": true, "third_party": true, "neutral": true}[x.Side] {
		bad(w, r, "Name and valid side are required")
		return
	}
	u := user(r)
	if !h.canMatter(w, r, u, matterID, "write") {
		return
	}
	created, e := h.Store.AddParty(r.Context(), u.FirmID, u.ID, matterID, x)
	if e != nil {
		h.fail(w, r, e)
		return
	}
	h.audit(r, "party.added", "matter", &matterID, map[string]any{"partyId": created.ID, "side": created.Side})
	writeJSON(w, 201, created)
}
func (h *Handler) AddNote(w http.ResponseWriter, r *http.Request) {
	matterID := chi.URLParam(r, "id")
	var x domain.Note
	if !validID(matterID) || decode(w, r, &x) != nil {
		bad(w, r, "Invalid note")
		return
	}
	x.Content = strings.TrimSpace(x.Content)
	if x.Content == "" || len([]rune(x.Content)) > 10000 || !map[string]bool{"team": true, "private": true}[x.Visibility] {
		bad(w, r, "Content and valid visibility are required")
		return
	}
	u := user(r)
	if !h.canMatter(w, r, u, matterID, "write") {
		return
	}
	created, e := h.Store.AddNote(r.Context(), u.FirmID, u.ID, matterID, x)
	if e != nil {
		h.fail(w, r, e)
		return
	}
	writeJSON(w, 201, created)
}
func (h *Handler) GrantMatterAccess(w http.ResponseWriter, r *http.Request) {
	matterID := chi.URLParam(r, "id")
	var in struct {
		UserID      *string `json:"userId"`
		RoleID      *string `json:"roleId"`
		AccessLevel string  `json:"accessLevel"`
	}
	if !validID(matterID) || decode(w, r, &in) != nil {
		bad(w, r, "Invalid access grant")
		return
	}
	if (in.UserID == nil) == (in.RoleID == nil) || !map[string]bool{"read": true, "write": true, "manage": true}[in.AccessLevel] {
		bad(w, r, "Choose exactly one user or role and an access level")
		return
	}
	if in.UserID != nil && !validID(*in.UserID) || in.RoleID != nil && !validID(*in.RoleID) {
		bad(w, r, "Invalid user or role ID")
		return
	}
	u := user(r)
	if !h.canMatter(w, r, u, matterID, "manage") {
		return
	}
	if e := h.Store.GrantMatterAccess(r.Context(), u.FirmID, matterID, in.UserID, in.RoleID, in.AccessLevel); e != nil {
		h.fail(w, r, e)
		return
	}
	h.audit(r, "matter.access_granted", "matter", &matterID, map[string]any{"accessLevel": in.AccessLevel})
	w.WriteHeader(204)
}
func (h *Handler) UpdateMatterStatus(w http.ResponseWriter, r *http.Request) {
	matterID := chi.URLParam(r, "id")
	var in struct {
		Status string `json:"status"`
	}
	if !validID(matterID) || decode(w, r, &in) != nil || !map[string]bool{"draft": true, "active": true, "on_hold": true, "closing": true}[in.Status] {
		bad(w, r, "Invalid status")
		return
	}
	u := user(r)
	if !h.canMatter(w, r, u, matterID, "write") {
		return
	}
	if e := h.Store.UpdateMatterStatus(r.Context(), u.FirmID, u.ID, matterID, in.Status); e != nil {
		h.fail(w, r, e)
		return
	}
	h.audit(r, "matter.status_changed", "matter", &matterID, map[string]any{"status": in.Status})
	w.WriteHeader(204)
}
func (h *Handler) CreateHearing(w http.ResponseWriter, r *http.Request) {
	var x domain.Hearing
	if decode(w, r, &x) != nil || !validID(x.MatterID) || x.ScheduledAt.IsZero() {
		bad(w, r, "Invalid hearing")
		return
	}
	if _, ok := required(x.Title, 180); !ok {
		bad(w, r, "Title is required")
		return
	}
	if x.Status == "" {
		x.Status = "scheduled"
	}
	u := user(r)
	if !h.canMatter(w, r, u, x.MatterID, "write") {
		return
	}
	created, e := h.Store.CreateHearing(r.Context(), u.FirmID, u.ID, x)
	if e != nil {
		h.fail(w, r, e)
		return
	}
	h.audit(r, "hearing.created", "hearing", &created.ID, map[string]any{"matterId": x.MatterID})
	writeJSON(w, 201, created)
}
func (h *Handler) CreateFinancialEntry(w http.ResponseWriter, r *http.Request) {
	matterID := chi.URLParam(r, "id")
	var in struct {
		Type        string  `json:"type"`
		AmountCents int64   `json:"amountCents"`
		Description string  `json:"description"`
		DueDate     *string `json:"dueDate"`
	}
	if !validID(matterID) || decode(w, r, &in) != nil || in.AmountCents < 0 || !map[string]bool{"fee": true, "expense": true, "court_cost": true, "reimbursement": true, "success_fee": true, "payment": true}[in.Type] {
		bad(w, r, "Invalid financial entry")
		return
	}
	description, ok := required(in.Description, 300)
	if !ok {
		bad(w, r, "Description is required")
		return
	}
	var due *time.Time
	if in.DueDate != nil {
		x, valid := date(*in.DueDate)
		if !valid {
			bad(w, r, "Invalid due date")
			return
		}
		due = &x
	}
	u := user(r)
	if !h.canMatter(w, r, u, matterID, "write") {
		return
	}
	id, e := h.Store.CreateFinancialEntry(r.Context(), u.FirmID, u.ID, matterID, in.Type, description, in.AmountCents, due)
	if e != nil {
		h.fail(w, r, e)
		return
	}
	h.audit(r, "finance.updated", "finance", &id, map[string]any{"matterId": matterID, "type": in.Type})
	writeJSON(w, 201, map[string]string{"id": id})
}
func (h *Handler) CreateWorkflow(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name        string                 `json:"name"`
		Description *string                `json:"description"`
		Stages      []domain.WorkflowStage `json:"stages"`
	}
	if decode(w, r, &in) != nil {
		bad(w, r, "Invalid workflow")
		return
	}
	name, ok := required(in.Name, 160)
	if !ok || len(in.Stages) > 50 {
		bad(w, r, "Workflow name is required and stages are limited to 50")
		return
	}
	for _, stage := range in.Stages {
		if _, valid := required(stage.Name, 120); !valid || !colorPattern.MatchString(stage.Color) {
			bad(w, r, "Every stage needs a name and valid color")
			return
		}
	}
	u := user(r)
	workflow, e := h.Store.CreateWorkflow(r.Context(), u.FirmID, name, in.Description)
	if e == nil {
		e = h.Store.ReplaceWorkflowStages(r.Context(), u.FirmID, workflow.ID, in.Stages)
	}
	if e != nil {
		h.fail(w, r, e)
		return
	}
	h.audit(r, "workflow.created", "workflow", &workflow.ID, map[string]any{"stageCount": len(in.Stages)})
	writeJSON(w, 201, workflow)
}
func (h *Handler) ReplaceWorkflowStages(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var in struct {
		Stages []domain.WorkflowStage `json:"stages"`
	}
	if !validID(id) || decode(w, r, &in) != nil || len(in.Stages) > 50 {
		bad(w, r, "Invalid stages")
		return
	}
	for _, stage := range in.Stages {
		if _, valid := required(stage.Name, 120); !valid || !colorPattern.MatchString(stage.Color) {
			bad(w, r, "Every stage needs a name and valid color")
			return
		}
	}
	u := user(r)
	if e := h.Store.ReplaceWorkflowStages(r.Context(), u.FirmID, id, in.Stages); e != nil {
		h.fail(w, r, e)
		return
	}
	h.audit(r, "workflow.updated", "workflow", &id, map[string]any{"stageCount": len(in.Stages)})
	w.WriteHeader(204)
}
func (h *Handler) CreateCustomField(w http.ResponseWriter, r *http.Request) {
	var in struct {
		EntityType string          `json:"entityType"`
		Key        string          `json:"key"`
		Label      string          `json:"label"`
		Type       string          `json:"type"`
		Required   bool            `json:"required"`
		Options    json.RawMessage `json:"options"`
	}
	if decode(w, r, &in) != nil || !map[string]bool{"matter": true, "client": true}[in.EntityType] || !map[string]bool{"text": true, "textarea": true, "number": true, "date": true, "boolean": true, "select": true}[in.Type] || !keyPattern.MatchString(in.Key) {
		bad(w, r, "Invalid custom field")
		return
	}
	label, ok := required(in.Label, 120)
	if !ok {
		bad(w, r, "Label is required")
		return
	}
	u := user(r)
	id, e := h.Store.CreateCustomField(r.Context(), u.FirmID, in.EntityType, in.Key, label, in.Type, in.Required, in.Options)
	if e != nil {
		h.fail(w, r, e)
		return
	}
	writeJSON(w, 201, map[string]string{"id": id})
}
func (h *Handler) CreateTag(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	}
	if decode(w, r, &in) != nil || !colorPattern.MatchString(in.Color) {
		bad(w, r, "Invalid tag")
		return
	}
	name, ok := required(in.Name, 60)
	if !ok {
		bad(w, r, "Name is required")
		return
	}
	u := user(r)
	id, e := h.Store.CreateTag(r.Context(), u.FirmID, name, in.Color)
	if e != nil {
		h.fail(w, r, e)
		return
	}
	writeJSON(w, 201, map[string]string{"id": id})
}
func (h *Handler) canMatter(w http.ResponseWriter, r *http.Request, u domain.User, id, level string) bool {
	ok, e := h.Store.CanAccessMatter(r.Context(), u.FirmID, u.ID, id, level)
	if e != nil || !ok {
		h.fail(w, r, repository.ErrForbidden)
		return false
	}
	return true
}

var colorPattern = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
var keyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{1,79}$`)
