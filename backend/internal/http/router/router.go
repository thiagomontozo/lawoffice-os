package router

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/config"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/http/handlers"
	appmw "github.com/thiagomontozo/lawoffice-os/backend/internal/http/middleware"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/observability"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/repository"
)

func New(h *handlers.Handler, store *repository.Store, cfg config.Config, metrics *observability.Metrics) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RealIP, appmw.RequestID, metrics.Middleware, appmw.Recover(h.Logger, handlers.WriteError), appmw.Logging(h.Logger), appmw.CORS(cfg.WebOrigin, handlers.WriteError))
	r.Get("/healthz", h.Health)
	r.Get("/readyz", h.Ready)
	if cfg.MetricsToken != "" {
		r.With(appmw.BearerToken(cfg.MetricsToken, handlers.WriteError)).Get("/metrics", metrics.ServeHTTP)
	}
	r.With(appmw.RateLimit(5, time.Hour, handlers.WriteError)).Post("/api/v1/setup", h.Setup)
	r.With(appmw.RateLimit(10, time.Minute, handlers.WriteError)).Post("/api/v1/auth/login", h.Login)
	r.With(appmw.RateLimit(10, time.Minute, handlers.WriteError)).Post("/api/v1/portal/login", h.PortalLogin)
	r.With(appmw.RateLimit(10, time.Minute, handlers.WriteError)).Post("/api/v1/portal/invitations/accept", h.AcceptPortalInvitation)
	r.With(appmw.RateLimit(5, 15*time.Minute, handlers.WriteError)).Post("/api/v1/portal/password/forgot", h.ForgotPortalPassword)
	r.With(appmw.RateLimit(10, 15*time.Minute, handlers.WriteError)).Post("/api/v1/portal/password/reset", h.ResetPortalPassword)
	r.Post("/api/v1/portal/logout", h.PortalLogout)
	r.Get("/api/v1/portal/matters", h.PortalMatters)
	r.Get("/api/v1/portal/matters/{id}", h.PortalMatter)
	r.Get("/api/v1/portal/documents/{id}/download", h.PortalDownloadDocument)
	r.Get("/api/v1/public/branding/{slug}", h.PublicBranding)
	r.Get("/api/v1/public/branding/{slug}/assets/{kind}", h.PublicBrandAsset)
	r.Group(func(a chi.Router) {
		a.Use(appmw.Authenticate(store, handlers.SessionCookie, cfg.SessionSecret, handlers.WriteError))
		a.Post("/api/v1/auth/logout", h.Logout)
		a.Get("/api/v1/auth/me", h.Me)
		a.Post("/api/v1/auth/change-password", h.ChangePassword)
		a.Get("/api/v1/stream", h.Stream)
		a.Get("/api/v1/branding", h.Branding)
		a.Get("/api/v1/branding/assets/{kind}", h.BrandAsset)
		a.With(appmw.Permission("branding.manage", handlers.WriteError)).Put("/api/v1/branding", h.UpdateBranding)
		a.With(appmw.Permission("branding.manage", handlers.WriteError)).Post("/api/v1/branding/assets/{kind}", h.UploadBrandAsset)

		a.With(appmw.Permission("users.read", handlers.WriteError)).Get("/api/v1/users", h.Users)
		a.With(appmw.Permission("users.manage", handlers.WriteError)).Post("/api/v1/users", h.CreateUser)
		a.With(appmw.Permission("users.manage", handlers.WriteError)).Patch("/api/v1/users/{id}/active", h.SetUserActive)
		a.With(appmw.Permission("users.read", handlers.WriteError)).Get("/api/v1/roles", h.Roles)
		a.With(appmw.Permission("users.manage", handlers.WriteError)).Post("/api/v1/roles", h.CreateRole)
		a.With(appmw.Permission("users.manage", handlers.WriteError)).Put("/api/v1/roles/{id}", h.UpdateRole)
		a.With(appmw.Permission("users.manage", handlers.WriteError)).Put("/api/v1/users/{id}/roles", h.UpdateUserRoles)
		a.With(appmw.Permission("users.manage", handlers.WriteError)).Get("/api/v1/permissions", h.Permissions)

		a.With(appmw.Permission("clients.read", handlers.WriteError)).Get("/api/v1/clients", h.Clients)
		a.With(appmw.Permission("clients.read", handlers.WriteError)).Get("/api/v1/clients/{id}", h.Client)
		a.With(appmw.Permission("clients.create", handlers.WriteError)).Post("/api/v1/clients", h.CreateClient)
		a.With(appmw.Permission("clients.update", handlers.WriteError)).Put("/api/v1/clients/{id}", h.UpdateClient)
		a.With(appmw.Permission("clients.archive", handlers.WriteError)).Delete("/api/v1/clients/{id}", h.ArchiveClient)
		a.With(appmw.Permission("clients.read", handlers.WriteError)).Get("/api/v1/contacts", h.Contacts)
		a.With(appmw.Permission("clients.create", handlers.WriteError)).Post("/api/v1/contacts", h.CreateContact)
		a.With(appmw.Permission("clients.update", handlers.WriteError)).Put("/api/v1/contacts/{id}", h.UpdateContact)
		a.With(appmw.Permission("clients.archive", handlers.WriteError)).Delete("/api/v1/contacts/{id}", h.ArchiveContact)
		a.With(appmw.Permission("matter.read", handlers.WriteError)).Get("/api/v1/matters", h.Matters)
		a.With(appmw.Permission("matter.read", handlers.WriteError)).Get("/api/v1/matters/{id}", h.Matter)
		a.With(appmw.Permission("matter.create", handlers.WriteError)).Post("/api/v1/matters", h.CreateMatter)
		a.With(appmw.Permission("matter.update", handlers.WriteError)).Patch("/api/v1/matters/{id}/status", h.UpdateMatterStatus)
		a.With(appmw.Permission("matter.update", handlers.WriteError)).Post("/api/v1/matters/{id}/parties", h.AddParty)
		a.With(appmw.Permission("matter.update", handlers.WriteError)).Post("/api/v1/matters/{id}/notes", h.AddNote)
		a.With(appmw.Permission("matter.update", handlers.WriteError)).Post("/api/v1/matters/{id}/access", h.GrantMatterAccess)
		a.With(appmw.Permission("clients.read", handlers.WriteError)).Post("/api/v1/conflicts/check", h.ConflictCheck)

		a.With(appmw.Permission("document.read", handlers.WriteError)).Get("/api/v1/documents", h.Documents)
		a.With(appmw.Permission("document.delete", handlers.WriteError)).Get("/api/v1/documents/deleted", h.DeletedDocuments)
		a.With(appmw.Permission("document.read", handlers.WriteError)).Get("/api/v1/documents/{id}/download", h.DownloadDocument)
		a.With(appmw.Permission("document.read", handlers.WriteError)).Get("/api/v1/documents/{id}/versions", h.Versions)
		a.With(appmw.Permission("document.upload", handlers.WriteError)).Post("/api/v1/documents", h.UploadDocument)
		a.With(appmw.Permission("document.upload", handlers.WriteError)).Post("/api/v1/documents/{id}/versions", h.AddVersion)
		a.With(appmw.Permission("document.delete", handlers.WriteError)).Delete("/api/v1/documents/{id}", h.DeleteDocument)
		a.With(appmw.Permission("document.delete", handlers.WriteError)).Post("/api/v1/documents/{id}/restore", h.RestoreDocument)

		a.With(appmw.Permission("deadline.read", handlers.WriteError)).Get("/api/v1/deadlines", h.Deadlines)
		a.With(appmw.Permission("deadline.manage", handlers.WriteError)).Post("/api/v1/deadlines", h.CreateDeadline)
		a.With(appmw.Permission("deadline.manage", handlers.WriteError)).Patch("/api/v1/deadlines/{id}/status", h.UpdateDeadlineStatus)
		a.With(appmw.Permission("task.read", handlers.WriteError)).Get("/api/v1/tasks", h.Tasks)
		a.With(appmw.Permission("task.manage", handlers.WriteError)).Post("/api/v1/tasks", h.CreateTask)
		a.With(appmw.Permission("task.manage", handlers.WriteError)).Patch("/api/v1/tasks/{id}/status", h.UpdateTaskStatus)
		a.With(appmw.Permission("deadline.read", handlers.WriteError)).Get("/api/v1/calendar", h.Calendar)
		a.With(appmw.Permission("deadline.manage", handlers.WriteError)).Post("/api/v1/hearings", h.CreateHearing)
		a.With(appmw.Permission("workflow.read", handlers.WriteError)).Get("/api/v1/workflows", h.Workflows)
		a.With(appmw.Permission("workflow.manage", handlers.WriteError)).Post("/api/v1/workflows", h.CreateWorkflow)
		a.With(appmw.Permission("workflow.manage", handlers.WriteError)).Put("/api/v1/workflows/{id}/stages", h.ReplaceWorkflowStages)
		a.With(appmw.Permission("firm.manage", handlers.WriteError)).Post("/api/v1/custom-fields", h.CreateCustomField)
		a.With(appmw.Permission("firm.manage", handlers.WriteError)).Post("/api/v1/tags", h.CreateTag)
		a.With(appmw.Permission("finance.manage", handlers.WriteError)).Post("/api/v1/matters/{id}/finance", h.CreateFinancialEntry)
		a.With(appmw.Permission("matter.archive", handlers.WriteError)).Post("/api/v1/matters/{id}/archive", h.Archive)
		a.With(appmw.Permission("matter.reopen", handlers.WriteError)).Post("/api/v1/matters/{id}/reopen", h.Reopen)
		a.With(appmw.Permission("matter.read", handlers.WriteError)).Get("/api/v1/search", h.Search)
		a.Get("/api/v1/notifications", h.Notifications)
		a.Patch("/api/v1/notifications/{id}/read", h.ReadNotification)
		a.Get("/api/v1/notifications/preferences", h.NotificationPreferences)
		a.Put("/api/v1/notifications/preferences", h.UpdateNotificationPreferences)
		a.With(appmw.Permission("audit.read", handlers.WriteError)).Get("/api/v1/audit", h.AuditEvents)
		a.With(appmw.Permission("portal.manage", handlers.WriteError)).Get("/api/v1/portal/users", h.PortalUsers)
		a.With(appmw.Permission("portal.manage", handlers.WriteError)).Post("/api/v1/portal/invitations", h.CreatePortalInvitation)
		a.With(appmw.Permission("portal.manage", handlers.WriteError)).Patch("/api/v1/portal/users/{id}/active", h.SetPortalUserActive)
		a.Get("/api/v1/dashboard", h.CommandCenter)
	})
	return r
}
