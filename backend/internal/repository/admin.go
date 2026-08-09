package repository

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/domain"
	"time"
)

func (s *Store) Users(ctx context.Context, firmID string) ([]domain.User, error) {
	rows, e := s.Pool.Query(ctx, `SELECT id,firm_id,name,email,active,last_login_at,created_at FROM users WHERE firm_id=$1 AND deleted_at IS NULL ORDER BY name`, firmID)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	items := []domain.User{}
	for rows.Next() {
		var u domain.User
		if e = rows.Scan(&u.ID, &u.FirmID, &u.Name, &u.Email, &u.Active, &u.LastLoginAt, &u.CreatedAt); e != nil {
			return nil, e
		}
		u.Roles, u.Permissions, _ = s.UserGrants(ctx, firmID, u.ID)
		items = append(items, u)
	}
	return items, rows.Err()
}
func (s *Store) CreateUser(ctx context.Context, firmID, name, email, hash string, roleIDs []string) (domain.User, error) {
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return domain.User{}, e
	}
	defer tx.Rollback(ctx)
	var u domain.User
	e = tx.QueryRow(ctx, `INSERT INTO users(firm_id,name,email,password_hash)VALUES($1,$2,lower($3),$4)RETURNING id,firm_id,name,email,active,created_at`, firmID, name, email, hash).Scan(&u.ID, &u.FirmID, &u.Name, &u.Email, &u.Active, &u.CreatedAt)
	if e != nil {
		return u, mapError(e)
	}
	for _, r := range roleIDs {
		if _, e = tx.Exec(ctx, `INSERT INTO user_roles(user_id,role_id,firm_id)SELECT $1,id,$2 FROM roles WHERE id=$3 AND firm_id=$2`, u.ID, firmID, r); e != nil {
			return u, mapError(e)
		}
	}
	if e = tx.Commit(ctx); e != nil {
		return u, e
	}
	return u, nil
}
func (s *Store) SetUserActive(ctx context.Context, firmID, id string, active bool) error {
	if !active {
		var isOwner bool
		var otherOwners int
		if e := s.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM user_roles ur JOIN roles r ON r.id=ur.role_id AND r.firm_id=ur.firm_id WHERE ur.firm_id=$1 AND ur.user_id=$2 AND r.name='Owner'),(SELECT count(DISTINCT u.id) FROM users u JOIN user_roles ur ON ur.user_id=u.id AND ur.firm_id=u.firm_id JOIN roles r ON r.id=ur.role_id AND r.firm_id=ur.firm_id WHERE u.firm_id=$1 AND u.id<>$2 AND u.active AND u.deleted_at IS NULL AND r.name='Owner')`, firmID, id).Scan(&isOwner, &otherOwners); e != nil {
			return e
		}
		if isOwner && otherOwners == 0 {
			return ErrInvalid
		}
	}
	r, e := s.Pool.Exec(ctx, `UPDATE users SET active=$3,disabled_at=CASE WHEN $3 THEN NULL ELSE now() END WHERE firm_id=$1 AND id=$2 AND deleted_at IS NULL`, firmID, id, active)
	if e != nil {
		return e
	}
	if r.RowsAffected() == 0 {
		return ErrNotFound
	}
	if !active {
		return s.RevokeUserSessions(ctx, firmID, id)
	}
	return nil
}
func (s *Store) ChangePassword(ctx context.Context, firmID, id, hash string) error {
	_, e := s.Pool.Exec(ctx, `UPDATE users SET password_hash=$3,updated_at=now() WHERE firm_id=$1 AND id=$2 AND active`, firmID, id, hash)
	if e == nil {
		e = s.RevokeUserSessions(ctx, firmID, id)
	}
	return e
}
func (s *Store) Roles(ctx context.Context, firmID string) ([]domain.Role, error) {
	rows, e := s.Pool.Query(ctx, `SELECT id,name,description,system FROM roles WHERE firm_id=$1 ORDER BY system DESC,name`, firmID)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	items := []domain.Role{}
	for rows.Next() {
		var r domain.Role
		if e = rows.Scan(&r.ID, &r.Name, &r.Description, &r.System); e != nil {
			return nil, e
		}
		p, e := s.Pool.Query(ctx, `SELECT permission_key FROM role_permissions WHERE role_id=$1 ORDER BY permission_key`, r.ID)
		if e != nil {
			return nil, e
		}
		for p.Next() {
			var k string
			if e = p.Scan(&k); e != nil {
				p.Close()
				return nil, e
			}
			r.Permissions = append(r.Permissions, k)
		}
		p.Close()
		items = append(items, r)
	}
	return items, rows.Err()
}
func (s *Store) CreateRole(ctx context.Context, firmID, name string, description *string, permissions []string) (domain.Role, error) {
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return domain.Role{}, e
	}
	defer tx.Rollback(ctx)
	var r domain.Role
	e = tx.QueryRow(ctx, `INSERT INTO roles(firm_id,name,description)VALUES($1,$2,$3)RETURNING id,name,description,system`, firmID, name, description).Scan(&r.ID, &r.Name, &r.Description, &r.System)
	if e != nil {
		return r, mapError(e)
	}
	for _, p := range permissions {
		if _, e = tx.Exec(ctx, `INSERT INTO role_permissions(role_id,permission_key)VALUES($1,$2)`, r.ID, p); e != nil {
			return r, mapError(e)
		}
	}
	if e = tx.Commit(ctx); e != nil {
		return r, e
	}
	r.Permissions = permissions
	return r, nil
}

func (s *Store) Permissions(ctx context.Context) ([]string, error) {
	rows, err := s.Pool.Query(ctx, `SELECT key FROM permissions ORDER BY key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []string{}
	for rows.Next() {
		var key string
		if err = rows.Scan(&key); err != nil {
			return nil, err
		}
		items = append(items, key)
	}
	return items, rows.Err()
}

func (s *Store) UpdateRole(ctx context.Context, firmID, id, name string, description *string, permissions []string) (domain.Role, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return domain.Role{}, err
	}
	defer tx.Rollback(ctx)
	var role domain.Role
	err = tx.QueryRow(ctx, `UPDATE roles SET name=$3,description=$4 WHERE firm_id=$1 AND id=$2 AND NOT system RETURNING id,name,description,system`, firmID, id, name, description).Scan(&role.ID, &role.Name, &role.Description, &role.System)
	if errors.Is(err, pgx.ErrNoRows) {
		return role, ErrNotFound
	}
	if err != nil {
		return role, mapError(err)
	}
	if _, err = tx.Exec(ctx, `DELETE FROM role_permissions WHERE role_id=$1`, id); err != nil {
		return role, err
	}
	for _, permission := range permissions {
		if _, err = tx.Exec(ctx, `INSERT INTO role_permissions(role_id,permission_key)VALUES($1,$2)`, id, permission); err != nil {
			return role, mapError(err)
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return role, err
	}
	role.Permissions = permissions
	return role, nil
}

func (s *Store) UpdateUserRoles(ctx context.Context, firmID, userID string, roleIDs []string) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var targetIsOwner bool
	var otherOwners int
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM user_roles ur JOIN roles r ON r.id=ur.role_id AND r.firm_id=ur.firm_id WHERE ur.firm_id=$1 AND ur.user_id=$2 AND r.name='Owner'),(SELECT count(DISTINCT u.id) FROM users u JOIN user_roles ur ON ur.user_id=u.id AND ur.firm_id=u.firm_id JOIN roles r ON r.id=ur.role_id AND r.firm_id=ur.firm_id WHERE u.firm_id=$1 AND u.id<>$2 AND u.active AND u.deleted_at IS NULL AND r.name='Owner')`, firmID, userID).Scan(&targetIsOwner, &otherOwners); err != nil {
		return err
	}
	proposedOwner := false
	for _, roleID := range roleIDs {
		var isOwner bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM roles WHERE firm_id=$1 AND id=$2 AND name='Owner')`, firmID, roleID).Scan(&isOwner); err != nil {
			return err
		}
		proposedOwner = proposedOwner || isOwner
	}
	if targetIsOwner && !proposedOwner && otherOwners == 0 {
		return ErrInvalid
	}
	if _, err = tx.Exec(ctx, `DELETE FROM user_roles WHERE firm_id=$1 AND user_id=$2`, firmID, userID); err != nil {
		return err
	}
	for _, roleID := range roleIDs {
		result, execErr := tx.Exec(ctx, `INSERT INTO user_roles(user_id,role_id,firm_id) SELECT $1,id,$2 FROM roles WHERE firm_id=$2 AND id=$3`, userID, firmID, roleID)
		if execErr != nil {
			return mapError(execErr)
		}
		if result.RowsAffected() == 0 {
			return ErrInvalid
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) UpdateTaskStatus(ctx context.Context, firmID, userID, id, status string) error {
	var matterID *string
	err := s.Pool.QueryRow(ctx, `UPDATE tasks SET status=$3::varchar,completed_at=CASE WHEN $3::varchar='done' THEN now() ELSE NULL END WHERE firm_id=$1 AND id=$2 RETURNING matter_id`, firmID, id, status).Scan(&matterID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err == nil && matterID != nil {
		_, _ = s.Pool.Exec(ctx, `INSERT INTO matter_events(firm_id,matter_id,type,summary,resource_type,resource_id,actor_id)VALUES($1,$2,'task.updated',$3,'task',$4,$5)`, firmID, *matterID, "Task status changed to "+status, id, userID)
	}
	return mapError(err)
}

func (s *Store) UpdateDeadlineStatus(ctx context.Context, firmID, userID, id, status string) error {
	var matterID string
	err := s.Pool.QueryRow(ctx, `UPDATE deadlines SET status=$3::varchar,completed_at=CASE WHEN $3::varchar='completed' THEN now() ELSE NULL END WHERE firm_id=$1 AND id=$2 RETURNING matter_id`, firmID, id, status).Scan(&matterID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err == nil {
		_, _ = s.Pool.Exec(ctx, `INSERT INTO matter_events(firm_id,matter_id,type,summary,resource_type,resource_id,actor_id)VALUES($1,$2,'deadline.updated',$3,'deadline',$4,$5)`, firmID, matterID, "Deadline status changed to "+status, id, userID)
	}
	return mapError(err)
}

func (s *Store) TaskMatter(ctx context.Context, firmID, id string) (*string, error) {
	var matterID *string
	err := s.Pool.QueryRow(ctx, `SELECT matter_id FROM tasks WHERE firm_id=$1 AND id=$2`, firmID, id).Scan(&matterID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return matterID, err
}

func (s *Store) DeadlineMatter(ctx context.Context, firmID, id string) (string, error) {
	var matterID string
	err := s.Pool.QueryRow(ctx, `SELECT matter_id FROM deadlines WHERE firm_id=$1 AND id=$2`, firmID, id).Scan(&matterID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return matterID, err
}
func (s *Store) CreateDeadline(ctx context.Context, firmID, userID string, x domain.Deadline) (domain.Deadline, error) {
	e := s.Pool.QueryRow(ctx, `INSERT INTO deadlines(firm_id,matter_id,title,description,due_at,priority,assigned_to,created_by)VALUES($1,$2,$3,$4,$5,$6,$7,$8)RETURNING id,status`, firmID, x.MatterID, x.Title, x.Description, x.DueAt, x.Priority, x.AssignedTo, userID).Scan(&x.ID, &x.Status)
	if e == nil {
		_, _ = s.Pool.Exec(ctx, `INSERT INTO matter_events(firm_id,matter_id,type,summary,resource_type,resource_id,actor_id)VALUES($1,$2,'deadline.created',$3,'deadline',$4,$5)`, firmID, x.MatterID, "Deadline created: "+x.Title, x.ID, userID)
	}
	return x, mapError(e)
}
func (s *Store) CreateTask(ctx context.Context, firmID, userID string, x domain.Task) (domain.Task, error) {
	e := s.Pool.QueryRow(ctx, `INSERT INTO tasks(firm_id,matter_id,title,description,assigned_to,status,priority,due_at,created_by)VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)RETURNING id`, firmID, x.MatterID, x.Title, x.Description, x.AssignedTo, x.Status, x.Priority, x.DueAt, userID).Scan(&x.ID)
	if e == nil && x.MatterID != nil {
		_, _ = s.Pool.Exec(ctx, `INSERT INTO matter_events(firm_id,matter_id,type,summary,resource_type,resource_id,actor_id)VALUES($1,$2,'task.created',$3,'task',$4,$5)`, firmID, *x.MatterID, "Task created: "+x.Title, x.ID, userID)
	}
	return x, mapError(e)
}
func (s *Store) ArchiveMatter(ctx context.Context, firmID, userID, matterID, reason, outcome, summary string, force bool) (map[string]any, error) {
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback(ctx)
	var tasks, deadlines, hearings int
	var pendingCents int64
	e = tx.QueryRow(ctx, `SELECT (SELECT count(*) FROM tasks WHERE firm_id=$1 AND matter_id=$2 AND status NOT IN('done','cancelled')),(SELECT count(*) FROM deadlines WHERE firm_id=$1 AND matter_id=$2 AND status='open'),(SELECT count(*) FROM hearings WHERE firm_id=$1 AND matter_id=$2 AND status='scheduled' AND scheduled_at>now()),COALESCE((SELECT sum(amount_cents) FROM matter_financial_entries WHERE firm_id=$1 AND matter_id=$2 AND type IN('fee','court_cost','expense') AND paid_at IS NULL),0)`, firmID, matterID).Scan(&tasks, &deadlines, &hearings, &pendingCents)
	if e != nil {
		return nil, e
	}
	warnings := map[string]any{"openTasks": tasks, "openDeadlines": deadlines, "futureHearings": hearings, "financialPending": pendingCents > 0}
	if !force && (tasks+deadlines+hearings > 0 || pendingCents > 0) {
		warnings["requiresConfirmation"] = true
		return warnings, nil
	}
	_, e = tx.Exec(ctx, `INSERT INTO matter_closures(firm_id,matter_id,closure_reason,outcome,closed_at,summary,pending_tasks_count,pending_deadlines_count,financial_pending,archived_by)VALUES($1,$2,$3,NULLIF($4,''),CURRENT_DATE,$5,$6,$7,$8,$9)`, firmID, matterID, reason, outcome, summary, tasks, deadlines, pendingCents > 0, userID)
	if e != nil {
		return nil, mapError(e)
	}
	_, e = tx.Exec(ctx, `UPDATE matters SET status='archived',closed_at=CURRENT_DATE,archived_at=now(),updated_at=now() WHERE firm_id=$1 AND id=$2`, firmID, matterID)
	if e != nil {
		return nil, e
	}
	_, e = tx.Exec(ctx, `INSERT INTO matter_events(firm_id,matter_id,type,summary,actor_id)VALUES($1,$2,'matter.archived',$3,$4)`, firmID, matterID, "Matter archived: "+reason, userID)
	if e != nil {
		return nil, e
	}
	if e = tx.Commit(ctx); e != nil {
		return nil, e
	}
	return warnings, nil
}
func (s *Store) ReopenMatter(ctx context.Context, firmID, userID, matterID, reason string) error {
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	r, e := tx.Exec(ctx, `UPDATE matters SET status='active',closed_at=NULL,archived_at=NULL,updated_at=now() WHERE firm_id=$1 AND id=$2 AND archived_at IS NOT NULL`, firmID, matterID)
	if e != nil {
		return e
	}
	if r.RowsAffected() == 0 {
		return ErrNotFound
	}
	_, e = tx.Exec(ctx, `INSERT INTO matter_events(firm_id,matter_id,type,summary,actor_id)VALUES($1,$2,'matter.reopened',$3,$4)`, firmID, matterID, "Matter reopened: "+reason, userID)
	if e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func (s *Store) Search(ctx context.Context, firmID, userID, q, kind string, limit int) ([]domain.SearchGroup, error) {
	clientsAllowed, _ := s.HasPermission(ctx, firmID, userID, "clients.read")
	rows, e := s.Pool.Query(ctx, searchSQL, firmID, q, userID, clientsAllowed, kind, limit)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	items := []domain.SearchGroup{}
	for rows.Next() {
		var x domain.SearchGroup
		if e = rows.Scan(&x.Type, &x.ID, &x.Title, &x.Subtitle, &x.MatchedBy, &x.Score); e != nil {
			return nil, e
		}
		items = append(items, x)
	}
	return items, rows.Err()
}

const searchSQL = `WITH candidates AS (
SELECT 'matter'::text type,m.id,m.title,m.internal_number::text subtitle,'matter'::text matched_by,GREATEST(similarity(m.title,$2::text),similarity(m.internal_number,$2::text),COALESCE(similarity(lp.case_number,$2::text),0))::float8 score
FROM matters m LEFT JOIN matter_legal_process lp ON lp.matter_id=m.id AND lp.firm_id=m.firm_id
WHERE m.firm_id=$1 AND m.deleted_at IS NULL AND ($5 IN('','matter')) AND (m.title ILIKE '%'||$2||'%' OR m.internal_number ILIKE '%'||$2||'%' OR lp.case_number ILIKE '%'||$2||'%') AND (m.confidentiality='normal' OR m.responsible_user_id=$3 OR m.created_by=$3 OR (m.confidentiality='partners_only' AND EXISTS(SELECT 1 FROM user_roles ur JOIN roles r ON r.id=ur.role_id AND r.firm_id=ur.firm_id WHERE ur.firm_id=$1 AND ur.user_id=$3 AND r.name IN('Owner','Partner','Administrator'))) OR EXISTS(SELECT 1 FROM matter_access ma WHERE ma.matter_id=m.id AND ma.firm_id=m.firm_id AND (ma.user_id=$3 OR ma.role_id IN(SELECT role_id FROM user_roles WHERE firm_id=$1 AND user_id=$3))))
UNION ALL SELECT 'client',c.id,c.name,COALESCE(c.document,''),'client',GREATEST(similarity(c.name,$2::text),COALESCE(similarity(c.document,$2::text),0))::float8 FROM clients c WHERE $4 AND c.firm_id=$1 AND c.deleted_at IS NULL AND ($5 IN('','client')) AND (c.name ILIKE '%'||$2||'%' OR c.document ILIKE '%'||$2||'%')
UNION ALL SELECT 'contact',c.id,c.name,c.type,'contact',similarity(c.name,$2::text)::float8 FROM contacts c WHERE $4 AND c.firm_id=$1 AND c.deleted_at IS NULL AND ($5 IN('','contact')) AND c.name ILIKE '%'||$2||'%'
UNION ALL SELECT 'document',d.id,d.title,d.category,'document',similarity(d.title,$2::text)::float8 FROM documents d LEFT JOIN matters m ON m.id=d.matter_id AND m.firm_id=d.firm_id WHERE d.firm_id=$1 AND d.deleted_at IS NULL AND ($5 IN('','document')) AND d.title ILIKE '%'||$2||'%' AND (d.matter_id IS NULL OR m.confidentiality='normal' OR m.responsible_user_id=$3 OR m.created_by=$3 OR (m.confidentiality='partners_only' AND EXISTS(SELECT 1 FROM user_roles ur JOIN roles r ON r.id=ur.role_id AND r.firm_id=ur.firm_id WHERE ur.firm_id=$1 AND ur.user_id=$3 AND r.name IN('Owner','Partner','Administrator'))) OR EXISTS(SELECT 1 FROM matter_access ma WHERE ma.matter_id=m.id AND ma.firm_id=m.firm_id AND (ma.user_id=$3 OR ma.role_id IN(SELECT role_id FROM user_roles WHERE firm_id=$1 AND user_id=$3))))
UNION ALL SELECT 'matter',m.id,m.title,'Party: '||p.name,'party',similarity(p.name,$2::text)::float8 FROM matter_parties p JOIN matters m ON m.id=p.matter_id AND m.firm_id=p.firm_id WHERE p.firm_id=$1 AND m.deleted_at IS NULL AND ($5 IN('','matter')) AND p.name ILIKE '%'||$2||'%' AND (m.confidentiality='normal' OR m.responsible_user_id=$3 OR m.created_by=$3 OR EXISTS(SELECT 1 FROM matter_access ma WHERE ma.matter_id=m.id AND ma.firm_id=m.firm_id AND (ma.user_id=$3 OR ma.role_id IN(SELECT role_id FROM user_roles WHERE firm_id=$1 AND user_id=$3))))
UNION ALL SELECT 'matter',m.id,m.title,'Tag: '||t.name,'tag',similarity(t.name,$2::text)::float8 FROM entity_tags et JOIN tags t ON t.id=et.tag_id AND t.firm_id=et.firm_id JOIN matters m ON m.id=et.entity_id AND m.firm_id=et.firm_id WHERE et.firm_id=$1 AND et.entity_type='matter' AND m.deleted_at IS NULL AND ($5 IN('','matter')) AND t.name ILIKE '%'||$2||'%' AND (m.confidentiality='normal' OR m.responsible_user_id=$3 OR m.created_by=$3 OR EXISTS(SELECT 1 FROM matter_access ma WHERE ma.matter_id=m.id AND ma.firm_id=m.firm_id AND (ma.user_id=$3 OR ma.role_id IN(SELECT role_id FROM user_roles WHERE firm_id=$1 AND user_id=$3))))
), ranked AS (SELECT DISTINCT ON(type,id) type,id,title,subtitle,matched_by,score FROM candidates ORDER BY type,id,score DESC)
SELECT type,id,title,subtitle,matched_by,score FROM ranked ORDER BY score DESC,title LIMIT $6`

func (s *Store) Notifications(ctx context.Context, firmID, userID string) ([]domain.Notification, error) {
	rows, e := s.Pool.Query(ctx, `SELECT id,type,title,message,resource_type,resource_id,read_at,created_at FROM notifications WHERE firm_id=$1 AND user_id=$2 ORDER BY created_at DESC LIMIT 100`, firmID, userID)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	items := []domain.Notification{}
	for rows.Next() {
		var x domain.Notification
		if e = rows.Scan(&x.ID, &x.Type, &x.Title, &x.Message, &x.ResourceType, &x.ResourceID, &x.ReadAt, &x.CreatedAt); e != nil {
			return nil, e
		}
		items = append(items, x)
	}
	return items, rows.Err()
}
func (s *Store) ReadNotification(ctx context.Context, firmID, userID, id string) error {
	r, e := s.Pool.Exec(ctx, `UPDATE notifications SET read_at=COALESCE(read_at,now()) WHERE firm_id=$1 AND user_id=$2 AND id=$3`, firmID, userID, id)
	if e != nil {
		return e
	}
	if r.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
func (s *Store) AuditEvents(ctx context.Context, firmID string) ([]domain.AuditEvent, error) {
	rows, e := s.Pool.Query(ctx, `SELECT a.id,u.name,a.action,a.resource_type,a.resource_id,a.metadata,a.created_at FROM audit_events a LEFT JOIN users u ON u.id=a.user_id WHERE a.firm_id=$1 ORDER BY a.created_at DESC LIMIT 200`, firmID)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	items := []domain.AuditEvent{}
	for rows.Next() {
		var x domain.AuditEvent
		if e = rows.Scan(&x.ID, &x.UserName, &x.Action, &x.ResourceType, &x.ResourceID, &x.Metadata, &x.CreatedAt); e != nil {
			return nil, e
		}
		items = append(items, x)
	}
	return items, rows.Err()
}
func (s *Store) CommandCenter(ctx context.Context, firmID, userID string) (domain.CommandCenter, error) {
	var x domain.CommandCenter
	x.CriticalDeadlines, _ = s.Deadlines(ctx, firmID, userID, nil, false)
	if len(x.CriticalDeadlines) > 8 {
		x.CriticalDeadlines = x.CriticalDeadlines[:8]
	}
	all, _ := s.Deadlines(ctx, firmID, userID, nil, false)
	for _, d := range all {
		if d.Status == "open" && d.DueAt.Format("2006-01-02") == time.Now().Format("2006-01-02") {
			x.TodayDeadlines = append(x.TodayDeadlines, d)
		}
	}
	x.Hearings, _ = s.Hearings(ctx, firmID, userID, time.Now(), time.Now().Add(7*24*time.Hour))
	x.MyTasks, _ = s.Tasks(ctx, firmID, userID, nil, true)
	x.RecentDocuments, _ = s.Documents(ctx, firmID, userID, "", nil, 6, 0)
	x.PriorityMatters, _ = s.Matters(ctx, firmID, userID, "", "", "critical", "", false, 6, 0)
	x.RecentActivity = []domain.MatterEvent{}
	_ = s.Pool.QueryRow(ctx, `SELECT count(*) FROM matters m WHERE m.firm_id=$1 AND m.status='closing' AND NOT EXISTS(SELECT 1 FROM tasks t WHERE t.matter_id=m.id AND t.status NOT IN('done','cancelled')) AND NOT EXISTS(SELECT 1 FROM deadlines d WHERE d.matter_id=m.id AND d.status='open')`, firmID).Scan(&x.ArchiveReady)
	return x, nil
}
func (s *Store) PublishDueNotifications(ctx context.Context, horizon time.Time) (int, error) {
	r, e := s.Pool.Exec(ctx, dueNotificationsSQL, horizon)
	if e != nil {
		return 0, e
	}
	return int(r.RowsAffected()), nil
}

const dueNotificationsSQL = `INSERT INTO notifications(firm_id,user_id,type,title,message,resource_type,resource_id) SELECT d.firm_id,d.assigned_to,'deadline.approaching',d.title,'Deadline approaching','deadline',d.id FROM deadlines d WHERE d.assigned_to IS NOT NULL AND d.status='open' AND d.due_at<=$1 AND NOT EXISTS(SELECT 1 FROM notifications n WHERE n.firm_id=d.firm_id AND n.user_id=d.assigned_to AND n.resource_type='deadline' AND n.resource_id=d.id) UNION ALL SELECT t.firm_id,t.assigned_to,'task.overdue',t.title,'Task is overdue','task',t.id FROM tasks t WHERE t.assigned_to IS NOT NULL AND t.status NOT IN('done','cancelled') AND t.due_at<now() AND NOT EXISTS(SELECT 1 FROM notifications n WHERE n.firm_id=t.firm_id AND n.user_id=t.assigned_to AND n.resource_type='task' AND n.resource_id=t.id)`

func (s *Store) PublishDueNotificationsLocked(ctx context.Context, horizon time.Time) (int, bool, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return 0, false, err
	}
	defer tx.Rollback(ctx)
	var locked bool
	if err = tx.QueryRow(ctx, `SELECT pg_try_advisory_xact_lock($1)`, int64(734627091)).Scan(&locked); err != nil {
		return 0, false, err
	}
	if !locked {
		return 0, false, nil
	}
	result, err := tx.Exec(ctx, dueNotificationsSQL, horizon)
	if err != nil {
		return 0, true, err
	}
	count := int(result.RowsAffected())
	if err = tx.Commit(ctx); err != nil {
		return 0, true, err
	}
	return count, true, nil
}
func (s *Store) PortalMatter(ctx context.Context, firmID, portalUserID, matterID string) (domain.MatterDetail, error) {
	var allowed bool
	e := s.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM portal_access WHERE firm_id=$1 AND portal_user_id=$2 AND matter_id=$3)`, firmID, portalUserID, matterID).Scan(&allowed)
	if e != nil || !allowed {
		return domain.MatterDetail{}, ErrForbidden
	}
	m, e := s.Matter(ctx, firmID, matterID)
	if e != nil {
		return domain.MatterDetail{}, e
	}
	events := []domain.MatterEvent{}
	rows, e := s.Pool.Query(ctx, `SELECT e.id,e.type,e.summary,e.resource_type,e.resource_id,'Equipe',e.client_visible,e.created_at FROM matter_events e WHERE e.firm_id=$1 AND e.matter_id=$2 AND e.client_visible ORDER BY e.created_at DESC`, firmID, matterID)
	if e == nil {
		defer rows.Close()
		for rows.Next() {
			var x domain.MatterEvent
			rows.Scan(&x.ID, &x.Type, &x.Summary, &x.ResourceType, &x.ResourceID, &x.ActorName, &x.ClientVisible, &x.CreatedAt)
			events = append(events, x)
		}
	}
	documents := []domain.Document{}
	docRows, docErr := s.Pool.Query(ctx, `SELECT d.id,d.matter_id,d.client_id,d.title,d.description,d.category,v.version_number,v.original_file_name,v.mime_type,v.size_bytes,v.checksum,d.client_visible,d.created_at FROM documents d JOIN document_versions v ON v.id=d.current_version_id AND v.firm_id=d.firm_id WHERE d.firm_id=$1 AND d.matter_id=$2 AND d.client_visible AND d.deleted_at IS NULL ORDER BY d.updated_at DESC`, firmID, matterID)
	if docErr == nil {
		defer docRows.Close()
		for docRows.Next() {
			var d domain.Document
			if scanErr := docRows.Scan(&d.ID, &d.MatterID, &d.ClientID, &d.Title, &d.Description, &d.Category, &d.VersionNumber, &d.OriginalFileName, &d.MimeType, &d.SizeBytes, &d.Checksum, &d.ClientVisible, &d.CreatedAt); scanErr != nil {
				return domain.MatterDetail{}, scanErr
			}
			documents = append(documents, d)
		}
	}
	publicMatter := domain.Matter{ID: m.ID, Type: m.Type, Title: m.Title, Status: m.Status, OpenedAt: m.OpenedAt}
	return domain.MatterDetail{Matter: publicMatter, Timeline: events, Documents: documents}, nil
}

func (s *Store) PortalCredentials(ctx context.Context, slug, email string) (string, string, string, error) {
	var firmID, userID, passwordHash string
	err := s.Pool.QueryRow(ctx, `SELECT p.firm_id,p.id,p.password_hash FROM portal_users p JOIN firms f ON f.id=p.firm_id WHERE f.slug=$1 AND lower(p.email)=lower($2) AND p.active`, slug, email).Scan(&firmID, &userID, &passwordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", "", ErrNotFound
	}
	return firmID, userID, passwordHash, err
}
func (s *Store) CreatePortalSession(ctx context.Context, firmID, userID string, tokenHash []byte, expiresAt time.Time) error {
	_, err := s.Pool.Exec(ctx, `INSERT INTO portal_sessions(token_hash,firm_id,portal_user_id,expires_at)VALUES($1,$2,$3,$4)`, tokenHash, firmID, userID, expiresAt)
	return err
}
func (s *Store) PortalBySession(ctx context.Context, tokenHash []byte) (string, string, error) {
	var firmID, userID string
	err := s.Pool.QueryRow(ctx, `SELECT s.firm_id,s.portal_user_id FROM portal_sessions s JOIN portal_users p ON p.id=s.portal_user_id AND p.firm_id=s.firm_id WHERE s.token_hash=$1 AND s.expires_at>now() AND p.active`, tokenHash).Scan(&firmID, &userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", ErrNotFound
	}
	return firmID, userID, err
}
func (s *Store) DeletePortalSession(ctx context.Context, tokenHash []byte) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM portal_sessions WHERE token_hash=$1`, tokenHash)
	return err
}
func (s *Store) PortalMatters(ctx context.Context, firmID, portalUserID string) ([]domain.Matter, error) {
	rows, err := s.Pool.Query(ctx, matterSelect+` JOIN portal_access pa ON pa.matter_id=m.id AND pa.firm_id=m.firm_id WHERE pa.firm_id=$1 AND pa.portal_user_id=$2 AND m.deleted_at IS NULL ORDER BY m.updated_at DESC`, firmID, portalUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.Matter{}
	for rows.Next() {
		m, scanErr := scanMatter(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, domain.Matter{ID: m.ID, Type: m.Type, Title: m.Title, Status: m.Status, OpenedAt: m.OpenedAt})
	}
	return items, rows.Err()
}

func (s *Store) CreatePortalUser(ctx context.Context, firmID, clientID, email, passwordHash string, matterIDs []string) (string, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	var id string
	if err = tx.QueryRow(ctx, `INSERT INTO portal_users(firm_id,client_id,email,password_hash)VALUES($1,$2,lower($3),$4)RETURNING id`, firmID, clientID, email, passwordHash).Scan(&id); err != nil {
		return "", mapError(err)
	}
	for _, matterID := range matterIDs {
		if _, err = tx.Exec(ctx, `INSERT INTO portal_access(firm_id,portal_user_id,matter_id,summary_visible,timeline_visible,appointments_visible)VALUES($1,$2,$3,true,true,true)`, firmID, id, matterID); err != nil {
			return "", mapError(err)
		}
	}
	return id, tx.Commit(ctx)
}
