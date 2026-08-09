package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/domain"
	"strings"
)

func (s *Store) HasPermission(ctx context.Context, firmID, userID, key string) (bool, error) {
	var ok bool
	e := s.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM user_roles ur JOIN role_permissions rp ON rp.role_id=ur.role_id WHERE ur.firm_id=$1 AND ur.user_id=$2 AND rp.permission_key=$3)`, firmID, userID, key).Scan(&ok)
	return ok, e
}
func (s *Store) CanAccessMatter(ctx context.Context, firmID, userID, matterID, level string) (bool, error) {
	var ok bool
	e := s.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM matters m WHERE m.id=$1 AND m.firm_id=$2 AND m.deleted_at IS NULL AND (m.confidentiality='normal' OR m.responsible_user_id=$3 OR m.created_by=$3 OR (m.confidentiality='partners_only' AND EXISTS(SELECT 1 FROM user_roles ur JOIN roles r ON r.id=ur.role_id WHERE ur.firm_id=$2 AND ur.user_id=$3 AND r.name IN('Owner','Partner','Administrator'))) OR EXISTS(SELECT 1 FROM matter_access ma WHERE ma.matter_id=m.id AND ma.firm_id=m.firm_id AND (ma.user_id=$3 OR ma.role_id IN(SELECT role_id FROM user_roles WHERE firm_id=$2 AND user_id=$3)) AND CASE $4 WHEN 'read' THEN ma.access_level IN('read','write','manage') WHEN 'write' THEN ma.access_level IN('write','manage') ELSE ma.access_level='manage' END)))`, matterID, firmID, userID, level).Scan(&ok)
	return ok, e
}
func (s *Store) Audit(ctx context.Context, firmID, userID, action, resourceType string, resourceID *string, metadata any, ip, userAgent string) error {
	body, _ := json.Marshal(metadata)
	_, e := s.Pool.Exec(ctx, `INSERT INTO audit_events(firm_id,user_id,action,resource_type,resource_id,metadata,ip_address,user_agent)VALUES($1,$2,$3,$4,$5,$6,NULLIF($7,'')::inet,$8)`, firmID, userID, action, resourceType, resourceID, body, ip, userAgent)
	return e
}
func (s *Store) Clients(ctx context.Context, firmID, q string, limit, offset int) ([]domain.Client, error) {
	rows, e := s.Pool.Query(ctx, `SELECT id,type,name,legal_name,trade_name,document,email,phone,notes,active,created_at FROM clients WHERE firm_id=$1 AND deleted_at IS NULL AND ($2='' OR name ILIKE '%'||$2||'%' OR document ILIKE '%'||$2||'%') ORDER BY name LIMIT $3 OFFSET $4`, firmID, q, limit, offset)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	items := []domain.Client{}
	for rows.Next() {
		var x domain.Client
		if e = rows.Scan(&x.ID, &x.Type, &x.Name, &x.LegalName, &x.TradeName, &x.Document, &x.Email, &x.Phone, &x.Notes, &x.Active, &x.CreatedAt); e != nil {
			return nil, e
		}
		items = append(items, x)
	}
	return items, rows.Err()
}
func (s *Store) CreateClient(ctx context.Context, firmID string, x domain.Client) (domain.Client, error) {
	e := s.Pool.QueryRow(ctx, `INSERT INTO clients(firm_id,type,name,legal_name,trade_name,document,email,phone,notes)VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)RETURNING id,active,created_at`, firmID, x.Type, x.Name, x.LegalName, x.TradeName, x.Document, x.Email, x.Phone, x.Notes).Scan(&x.ID, &x.Active, &x.CreatedAt)
	return x, mapError(e)
}

func (s *Store) UpdateClient(ctx context.Context, firmID, id string, x domain.Client) (domain.Client, error) {
	e := s.Pool.QueryRow(ctx, `UPDATE clients SET type=$3,name=$4,legal_name=$5,trade_name=$6,document=$7,email=$8,phone=$9,notes=$10,updated_at=now() WHERE firm_id=$1 AND id=$2 AND deleted_at IS NULL RETURNING id,type,name,legal_name,trade_name,document,email,phone,notes,active,created_at`, firmID, id, x.Type, x.Name, x.LegalName, x.TradeName, x.Document, x.Email, x.Phone, x.Notes).Scan(&x.ID, &x.Type, &x.Name, &x.LegalName, &x.TradeName, &x.Document, &x.Email, &x.Phone, &x.Notes, &x.Active, &x.CreatedAt)
	if errors.Is(e, pgx.ErrNoRows) {
		return x, ErrNotFound
	}
	return x, mapError(e)
}

func (s *Store) ArchiveClient(ctx context.Context, firmID, userID, id string) error {
	r, e := s.Pool.Exec(ctx, `UPDATE clients SET active=false,deleted_at=now(),deleted_by=$3,updated_at=now() WHERE firm_id=$1 AND id=$2 AND deleted_at IS NULL`, firmID, id, userID)
	if e != nil {
		return mapError(e)
	}
	if r.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
func (s *Store) Client(ctx context.Context, firmID, id string) (domain.Client, error) {
	var x domain.Client
	e := s.Pool.QueryRow(ctx, `SELECT id,type,name,legal_name,trade_name,document,email,phone,notes,active,created_at FROM clients WHERE firm_id=$1 AND id=$2 AND deleted_at IS NULL`, firmID, id).Scan(&x.ID, &x.Type, &x.Name, &x.LegalName, &x.TradeName, &x.Document, &x.Email, &x.Phone, &x.Notes, &x.Active, &x.CreatedAt)
	if errors.Is(e, pgx.ErrNoRows) {
		return x, ErrNotFound
	}
	return x, e
}

const matterSelect = `SELECT m.id,m.type,m.internal_number,m.title,m.description,m.legal_area_id,la.name,m.status,m.priority,m.responsible_user_id,u.name,m.confidentiality,m.opened_at,m.archived_at,m.created_at,lp.case_number,lp.court FROM matters m LEFT JOIN legal_areas la ON la.id=m.legal_area_id LEFT JOIN users u ON u.id=m.responsible_user_id LEFT JOIN matter_legal_process lp ON lp.matter_id=m.id AND lp.firm_id=m.firm_id`

func scanMatter(row pgx.Row) (domain.Matter, error) {
	var x domain.Matter
	e := row.Scan(&x.ID, &x.Type, &x.InternalNumber, &x.Title, &x.Description, &x.LegalAreaID, &x.LegalAreaName, &x.Status, &x.Priority, &x.ResponsibleUserID, &x.ResponsibleName, &x.Confidentiality, &x.OpenedAt, &x.ArchivedAt, &x.CreatedAt, &x.CaseNumber, &x.Court)
	return x, e
}
func (s *Store) Matters(ctx context.Context, firmID, userID, q, status, priority, kind string, archived bool, limit, offset int) ([]domain.Matter, error) {
	where := `m.firm_id=$1 AND m.deleted_at IS NULL AND ($2='' OR m.title ILIKE '%'||$2||'%' OR m.internal_number ILIKE '%'||$2||'%' OR lp.case_number ILIKE '%'||$2||'%')`
	args := []any{firmID, q, userID}
	where += ` AND (m.confidentiality='normal' OR m.responsible_user_id=$3 OR m.created_by=$3 OR (m.confidentiality='partners_only' AND EXISTS(SELECT 1 FROM user_roles ur JOIN roles r ON r.id=ur.role_id AND r.firm_id=ur.firm_id WHERE ur.firm_id=$1 AND ur.user_id=$3 AND r.name IN('Owner','Partner','Administrator'))) OR EXISTS(SELECT 1 FROM matter_access ma WHERE ma.matter_id=m.id AND ma.firm_id=m.firm_id AND (ma.user_id=$3 OR ma.role_id IN(SELECT role_id FROM user_roles WHERE firm_id=$1 AND user_id=$3))))`
	add := func(field, value string) {
		if value != "" {
			args = append(args, value)
			where += fmt.Sprintf(" AND "+field+"=$%d", len(args))
		}
	}
	add("m.status", status)
	add("m.priority", priority)
	add("m.type", kind)
	if archived {
		where += ` AND m.archived_at IS NOT NULL`
	} else {
		where += ` AND m.archived_at IS NULL`
	}
	args = append(args, limit, offset)
	rows, e := s.Pool.Query(ctx, matterSelect+` WHERE `+where+fmt.Sprintf(` ORDER BY CASE m.priority WHEN 'critical' THEN 0 WHEN 'high' THEN 1 ELSE 2 END,m.updated_at DESC LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	items := []domain.Matter{}
	for rows.Next() {
		x, e := scanMatter(rows)
		if e != nil {
			return nil, e
		}
		items = append(items, x)
	}
	return items, rows.Err()
}
func (s *Store) CreateMatter(ctx context.Context, firmID, userID string, x domain.Matter) (domain.Matter, error) {
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return x, e
	}
	defer tx.Rollback(ctx)
	e = tx.QueryRow(ctx, `INSERT INTO matters(firm_id,type,internal_number,title,description,legal_area_id,status,priority,responsible_user_id,confidentiality,opened_at,created_by)VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)RETURNING id,created_at`, firmID, x.Type, x.InternalNumber, x.Title, x.Description, x.LegalAreaID, x.Status, x.Priority, x.ResponsibleUserID, x.Confidentiality, x.OpenedAt, userID).Scan(&x.ID, &x.CreatedAt)
	if e != nil {
		return x, mapError(e)
	}
	if x.CaseNumber != nil || x.Court != nil {
		if _, e = tx.Exec(ctx, `INSERT INTO matter_legal_process(matter_id,firm_id,case_number,court)VALUES($1,$2,$3,$4)`, x.ID, firmID, x.CaseNumber, x.Court); e != nil {
			return x, mapError(e)
		}
	}
	if _, e = tx.Exec(ctx, `INSERT INTO matter_events(firm_id,matter_id,type,summary,actor_id)VALUES($1,$2,'matter.created',$3,$4)`, firmID, x.ID, "Matter created: "+x.Title, userID); e != nil {
		return x, e
	}
	if e = tx.Commit(ctx); e != nil {
		return x, e
	}
	return x, nil
}
func (s *Store) Matter(ctx context.Context, firmID, id string) (domain.Matter, error) {
	x, e := scanMatter(s.Pool.QueryRow(ctx, matterSelect+` WHERE m.firm_id=$1 AND m.id=$2 AND m.deleted_at IS NULL`, firmID, id))
	if errors.Is(e, pgx.ErrNoRows) {
		return x, ErrNotFound
	}
	return x, e
}
func (s *Store) Timeline(ctx context.Context, firmID, matterID, userID string) ([]domain.MatterEvent, error) {
	rows, e := s.Pool.Query(ctx, `SELECT e.id,e.type,e.summary,e.resource_type,e.resource_id,u.name,e.client_visible,e.created_at FROM matter_events e JOIN users u ON u.id=e.actor_id WHERE e.firm_id=$1 AND e.matter_id=$2 ORDER BY e.created_at DESC LIMIT 200`, firmID, matterID)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	items := []domain.MatterEvent{}
	for rows.Next() {
		var x domain.MatterEvent
		if e = rows.Scan(&x.ID, &x.Type, &x.Summary, &x.ResourceType, &x.ResourceID, &x.ActorName, &x.ClientVisible, &x.CreatedAt); e != nil {
			return nil, e
		}
		items = append(items, x)
	}
	return items, rows.Err()
}
func (s *Store) Parties(ctx context.Context, firmID, matterID string) ([]domain.Party, error) {
	rows, e := s.Pool.Query(ctx, `SELECT id,name,role,side,document FROM matter_parties WHERE firm_id=$1 AND matter_id=$2 ORDER BY side,name`, firmID, matterID)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	items := []domain.Party{}
	for rows.Next() {
		var x domain.Party
		if e = rows.Scan(&x.ID, &x.Name, &x.Role, &x.Side, &x.Document); e != nil {
			return nil, e
		}
		items = append(items, x)
	}
	return items, rows.Err()
}
func (s *Store) Notes(ctx context.Context, firmID, matterID, userID string) ([]domain.Note, error) {
	rows, e := s.Pool.Query(ctx, `SELECT n.id,n.content,n.visibility,u.name,n.created_at FROM matter_notes n JOIN users u ON u.id=n.created_by WHERE n.firm_id=$1 AND n.matter_id=$2 AND (n.visibility='team' OR n.created_by=$3) ORDER BY n.created_at DESC`, firmID, matterID, userID)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	items := []domain.Note{}
	for rows.Next() {
		var x domain.Note
		if e = rows.Scan(&x.ID, &x.Content, &x.Visibility, &x.AuthorName, &x.CreatedAt); e != nil {
			return nil, e
		}
		items = append(items, x)
	}
	return items, rows.Err()
}
func (s *Store) ConflictCheck(ctx context.Context, firmID, term, document string) (domain.ConflictResult, error) {
	pattern := "%" + strings.TrimSpace(term) + "%"
	rows, e := s.Pool.Query(ctx, `SELECT entity_type,name,relationship,matter_id,matter_title,status,responsible FROM(SELECT 'client' entity_type,c.name,'client' relationship,NULL::uuid matter_id,NULL::text matter_title,CASE WHEN c.active THEN 'active' ELSE 'archived' END status,NULL::text responsible FROM clients c WHERE c.firm_id=$1 AND (c.name ILIKE $2 OR ($3<>'' AND c.document=$3)) UNION ALL SELECT 'contact',c.name,c.type,NULL,NULL,NULL,NULL FROM contacts c WHERE c.firm_id=$1 AND c.deleted_at IS NULL AND (c.name ILIKE $2 OR ($3<>'' AND c.document=$3)) UNION ALL SELECT 'party',p.name,p.side,m.id,m.title,m.status,u.name FROM matter_parties p JOIN matters m ON m.id=p.matter_id AND m.firm_id=p.firm_id LEFT JOIN users u ON u.id=m.responsible_user_id WHERE p.firm_id=$1 AND (p.name ILIKE $2 OR ($3<>'' AND p.document=$3)) UNION ALL SELECT 'matter',m.title,'matter title',m.id,m.title,m.status,u.name FROM matters m LEFT JOIN users u ON u.id=m.responsible_user_id WHERE m.firm_id=$1 AND m.title ILIKE $2)q LIMIT 100`, firmID, pattern, document)
	if e != nil {
		return domain.ConflictResult{}, e
	}
	defer rows.Close()
	matches := []domain.ConflictMatch{}
	for rows.Next() {
		var x domain.ConflictMatch
		if e = rows.Scan(&x.EntityType, &x.Name, &x.Relationship, &x.MatterID, &x.MatterTitle, &x.Status, &x.Responsible); e != nil {
			return domain.ConflictResult{}, e
		}
		matches = append(matches, x)
	}
	return domain.ConflictResult{Possible: len(matches) > 0, Matches: matches, Disclaimer: "Possible matches require professional review. The system does not make a legal conflict determination."}, rows.Err()
}
