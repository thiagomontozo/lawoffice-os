package repository

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/domain"
	"strings"
	"time"
)

var defaultPermissions = []string{"firm.manage", "branding.manage", "users.read", "users.manage", "clients.read", "clients.create", "clients.update", "clients.archive", "matter.read", "matter.create", "matter.update", "matter.archive", "matter.reopen", "document.read", "document.upload", "document.update", "document.delete", "deadline.read", "deadline.manage", "task.read", "task.manage", "workflow.read", "workflow.manage", "finance.read", "finance.manage", "audit.read", "archive.read", "portal.manage"}

func (s *Store) Setup(ctx context.Context, firm domain.Firm, adminName, email, passwordHash string) (domain.User, error) {
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return domain.User{}, e
	}
	defer tx.Rollback(ctx)
	if e = tx.QueryRow(ctx, `INSERT INTO firms(legal_name,display_name,slug,email,phone,website,timezone,locale)VALUES($1,$2,$3,lower($4),$5,$6,$7,$8)RETURNING id`, firm.LegalName, firm.DisplayName, firm.Slug, firm.Email, firm.Phone, firm.Website, firm.Timezone, firm.Locale).Scan(&firm.ID); e != nil {
		return domain.User{}, mapError(e)
	}
	var user domain.User
	if e = tx.QueryRow(ctx, `INSERT INTO users(firm_id,name,email,password_hash)VALUES($1,$2,lower($3),$4)RETURNING id,firm_id,name,email,active,created_at`, firm.ID, adminName, email, passwordHash).Scan(&user.ID, &user.FirmID, &user.Name, &user.Email, &user.Active, &user.CreatedAt); e != nil {
		return user, mapError(e)
	}
	var roleID string
	if e = tx.QueryRow(ctx, `INSERT INTO roles(firm_id,name,description,system)VALUES($1,'Owner','Full firm ownership',true)RETURNING id`, firm.ID).Scan(&roleID); e != nil {
		return user, e
	}
	for _, p := range defaultPermissions {
		if _, e = tx.Exec(ctx, `INSERT INTO role_permissions(role_id,permission_key)VALUES($1,$2)`, roleID, p); e != nil {
			return user, e
		}
	}
	if _, e = tx.Exec(ctx, `INSERT INTO user_roles(user_id,role_id,firm_id)VALUES($1,$2,$3)`, user.ID, roleID, firm.ID); e != nil {
		return user, e
	}
	if _, e = tx.Exec(ctx, `INSERT INTO firm_branding(firm_id,firm_display_name,system_title,support_email)VALUES($1,$2,$3,$4)`, firm.ID, firm.DisplayName, "Legal Workspace", firm.Email); e != nil {
		return user, e
	}
	areas := []string{"Civil", "Labor", "Tax", "Administrative", "Corporate", "Consumer", "Family", "Criminal", "Real Estate", "Social Security", "Contract", "Other"}
	for _, a := range areas {
		if _, e = tx.Exec(ctx, `INSERT INTO legal_areas(firm_id,name)VALUES($1,$2)`, firm.ID, a); e != nil {
			return user, e
		}
	}
	types := map[string]string{"judicial_process": "Judicial Process", "administrative_process": "Administrative Process", "legal_consultation": "Legal Consultation", "contract": "Contract", "advisory": "Advisory", "arbitration": "Arbitration", "extrajudicial": "Extrajudicial", "internal_legal_project": "Internal Legal Project", "other": "Other"}
	for k, n := range types {
		if _, e = tx.Exec(ctx, `INSERT INTO matter_types(firm_id,key,name)VALUES($1,$2,$3)`, firm.ID, k, n); e != nil {
			return user, e
		}
	}
	var workflowID string
	if e = tx.QueryRow(ctx, `INSERT INTO workflow_definitions(firm_id,name,description)VALUES($1,'Default Legal Workflow','Initial firm workflow')RETURNING id`, firm.ID).Scan(&workflowID); e != nil {
		return user, e
	}
	for i, n := range []string{"Intake", "Initial analysis", "Legal work", "Review", "Filing / Delivery", "Follow-up", "Closing"} {
		if _, e = tx.Exec(ctx, `INSERT INTO workflow_stages(workflow_id,firm_id,name,sort_order)VALUES($1,$2,$3,$4)`, workflowID, firm.ID, n, i); e != nil {
			return user, e
		}
	}
	if _, e = tx.Exec(ctx, `UPDATE firms SET setup_completed=true WHERE id=$1`, firm.ID); e != nil {
		return user, e
	}
	if e = tx.Commit(ctx); e != nil {
		return user, e
	}
	user.Roles = []string{"Owner"}
	user.Permissions = defaultPermissions
	return user, nil
}
func (s *Store) Credentials(ctx context.Context, slug, email string) (domain.User, string, error) {
	var u domain.User
	var hash string
	e := s.Pool.QueryRow(ctx, `SELECT u.id,u.firm_id,u.name,u.email,u.password_hash,u.active,u.last_login_at,u.created_at FROM users u JOIN firms f ON f.id=u.firm_id WHERE f.slug=$1 AND lower(u.email)=lower($2) AND u.deleted_at IS NULL`, strings.TrimSpace(slug), strings.TrimSpace(email)).Scan(&u.ID, &u.FirmID, &u.Name, &u.Email, &hash, &u.Active, &u.LastLoginAt, &u.CreatedAt)
	if errors.Is(e, pgx.ErrNoRows) {
		return u, "", ErrNotFound
	}
	if e == nil {
		u.Roles, u.Permissions, _ = s.UserGrants(ctx, u.FirmID, u.ID)
	}
	return u, hash, e
}
func (s *Store) PasswordHash(ctx context.Context, firmID, userID string) (string, error) {
	var hash string
	err := s.Pool.QueryRow(ctx, `SELECT password_hash FROM users WHERE firm_id=$1 AND id=$2 AND active AND deleted_at IS NULL`, firmID, userID).Scan(&hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return hash, err
}
func (s *Store) UserGrants(ctx context.Context, firmID, userID string) ([]string, []string, error) {
	rows, e := s.Pool.Query(ctx, `SELECT DISTINCT r.name,rp.permission_key FROM user_roles ur JOIN roles r ON r.id=ur.role_id AND r.firm_id=ur.firm_id LEFT JOIN role_permissions rp ON rp.role_id=r.id WHERE ur.firm_id=$1 AND ur.user_id=$2`, firmID, userID)
	if e != nil {
		return nil, nil, e
	}
	defer rows.Close()
	rolesMap := map[string]bool{}
	permMap := map[string]bool{}
	for rows.Next() {
		var r string
		var p *string
		if e = rows.Scan(&r, &p); e != nil {
			return nil, nil, e
		}
		rolesMap[r] = true
		if p != nil {
			permMap[*p] = true
		}
	}
	roles := []string{}
	perms := []string{}
	for v := range rolesMap {
		roles = append(roles, v)
	}
	for v := range permMap {
		perms = append(perms, v)
	}
	return roles, perms, rows.Err()
}
func (s *Store) CreateSession(ctx context.Context, u domain.User, hash []byte, expires time.Time) error {
	_, e := s.Pool.Exec(ctx, `INSERT INTO sessions(firm_id,user_id,token_hash,expires_at)VALUES($1,$2,$3,$4)`, u.FirmID, u.ID, hash, expires)
	if e == nil {
		_, _ = s.Pool.Exec(ctx, `UPDATE users SET last_login_at=now() WHERE id=$1 AND firm_id=$2`, u.ID, u.FirmID)
	}
	return e
}
func (s *Store) UserBySession(ctx context.Context, hash []byte) (domain.User, error) {
	var u domain.User
	e := s.Pool.QueryRow(ctx, `SELECT u.id,u.firm_id,u.name,u.email,u.active,u.last_login_at,u.created_at FROM sessions s JOIN users u ON u.id=s.user_id AND u.firm_id=s.firm_id WHERE s.token_hash=$1 AND s.expires_at>now() AND u.active AND u.deleted_at IS NULL`, hash).Scan(&u.ID, &u.FirmID, &u.Name, &u.Email, &u.Active, &u.LastLoginAt, &u.CreatedAt)
	if errors.Is(e, pgx.ErrNoRows) {
		return u, ErrNotFound
	}
	if e == nil {
		u.Roles, u.Permissions, _ = s.UserGrants(ctx, u.FirmID, u.ID)
	}
	return u, e
}
func (s *Store) DeleteSession(ctx context.Context, hash []byte) error {
	_, e := s.Pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash=$1`, hash)
	return e
}
func (s *Store) RevokeUserSessions(ctx context.Context, firmID, userID string) error {
	_, e := s.Pool.Exec(ctx, `DELETE FROM sessions WHERE firm_id=$1 AND user_id=$2`, firmID, userID)
	return e
}
func (s *Store) Firm(ctx context.Context, firmID string) (domain.Firm, error) {
	var f domain.Firm
	e := s.Pool.QueryRow(ctx, `SELECT id,legal_name,display_name,slug,email,phone,website,timezone,locale,setup_completed FROM firms WHERE id=$1`, firmID).Scan(&f.ID, &f.LegalName, &f.DisplayName, &f.Slug, &f.Email, &f.Phone, &f.Website, &f.Timezone, &f.Locale, &f.SetupCompleted)
	if errors.Is(e, pgx.ErrNoRows) {
		return f, ErrNotFound
	}
	return f, e
}
func (s *Store) Branding(ctx context.Context, firmID string) (domain.Branding, error) {
	var b domain.Branding
	var light, dark, favicon, background *string
	e := s.Pool.QueryRow(ctx, `SELECT firm_id,system_title,firm_display_name,logo_light_storage_key,logo_dark_storage_key,favicon_storage_key,login_background_storage_key,primary_color,secondary_color,accent_color,sidebar_style,border_radius_style,support_email,support_phone,website,client_portal_title,client_portal_welcome_text,updated_at FROM firm_branding WHERE firm_id=$1`, firmID).Scan(&b.FirmID, &b.SystemTitle, &b.FirmDisplayName, &light, &dark, &favicon, &background, &b.PrimaryColor, &b.SecondaryColor, &b.AccentColor, &b.SidebarStyle, &b.BorderRadiusStyle, &b.SupportEmail, &b.SupportPhone, &b.Website, &b.ClientPortalTitle, &b.ClientPortalWelcomeText, &b.UpdatedAt)
	if light != nil {
		v := "/api/v1/branding/assets/logo-light"
		b.LogoLightURL = &v
	}
	if dark != nil {
		v := "/api/v1/branding/assets/logo-dark"
		b.LogoDarkURL = &v
	}
	if favicon != nil {
		v := "/api/v1/branding/assets/favicon"
		b.FaviconURL = &v
	}
	if background != nil {
		v := "/api/v1/branding/assets/login-background"
		b.LoginBackgroundURL = &v
	}
	return b, e
}
func (s *Store) BrandingBySlug(ctx context.Context, slug string) (domain.Branding, error) {
	var firmID string
	if err := s.Pool.QueryRow(ctx, `SELECT id FROM firms WHERE slug=$1`, slug).Scan(&firmID); err != nil {
		return domain.Branding{}, ErrNotFound
	}
	b, err := s.Branding(ctx, firmID)
	if err != nil {
		return b, err
	}
	prefix := "/api/v1/public/branding/" + slug + "/assets/"
	if b.LogoLightURL != nil {
		v := prefix + "logo-light"
		b.LogoLightURL = &v
	}
	if b.LogoDarkURL != nil {
		v := prefix + "logo-dark"
		b.LogoDarkURL = &v
	}
	if b.FaviconURL != nil {
		v := prefix + "favicon"
		b.FaviconURL = &v
	}
	if b.LoginBackgroundURL != nil {
		v := prefix + "login-background"
		b.LoginBackgroundURL = &v
	}
	return b, nil
}
func (s *Store) BrandAssetKeyBySlug(ctx context.Context, slug, kind string) (string, error) {
	var firmID string
	if err := s.Pool.QueryRow(ctx, `SELECT id FROM firms WHERE slug=$1`, slug).Scan(&firmID); err != nil {
		return "", ErrNotFound
	}
	return s.BrandAssetKey(ctx, firmID, kind)
}
func (s *Store) SetBrandAsset(ctx context.Context, firmID, userID, kind, key string) error {
	columns := map[string]string{"logo-light": "logo_light_storage_key", "logo-dark": "logo_dark_storage_key", "favicon": "favicon_storage_key", "login-background": "login_background_storage_key"}
	column, ok := columns[kind]
	if !ok {
		return ErrInvalid
	}
	_, err := s.Pool.Exec(ctx, `UPDATE firm_branding SET `+column+`=$3,updated_by=$2,updated_at=now() WHERE firm_id=$1`, firmID, userID, key)
	return err
}
func (s *Store) BrandAssetKey(ctx context.Context, firmID, kind string) (string, error) {
	columns := map[string]string{"logo-light": "logo_light_storage_key", "logo-dark": "logo_dark_storage_key", "favicon": "favicon_storage_key", "login-background": "login_background_storage_key"}
	column, ok := columns[kind]
	if !ok {
		return "", ErrInvalid
	}
	var key *string
	if err := s.Pool.QueryRow(ctx, `SELECT `+column+` FROM firm_branding WHERE firm_id=$1`, firmID).Scan(&key); err != nil {
		return "", err
	}
	if key == nil {
		return "", ErrNotFound
	}
	return *key, nil
}
func (s *Store) UpdateBranding(ctx context.Context, firmID, userID string, b domain.Branding) (domain.Branding, error) {
	e := s.Pool.QueryRow(ctx, `UPDATE firm_branding SET system_title=$3,firm_display_name=$4,primary_color=$5,secondary_color=$6,accent_color=$7,sidebar_style=$8,border_radius_style=$9,support_email=$10,support_phone=$11,website=$12,client_portal_title=$13,client_portal_welcome_text=$14,updated_at=now(),updated_by=$2 WHERE firm_id=$1 RETURNING updated_at`, firmID, userID, b.SystemTitle, b.FirmDisplayName, b.PrimaryColor, b.SecondaryColor, b.AccentColor, b.SidebarStyle, b.BorderRadiusStyle, b.SupportEmail, b.SupportPhone, b.Website, b.ClientPortalTitle, b.ClientPortalWelcomeText).Scan(&b.UpdatedAt)
	b.FirmID = firmID
	return b, e
}
