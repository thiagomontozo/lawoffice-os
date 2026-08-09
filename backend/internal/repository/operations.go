package repository

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/domain"
	"time"
)

func (s *Store) CreateDocument(ctx context.Context, firmID, userID string, d domain.Document, storageKey string) (domain.Document, error) {
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return d, e
	}
	defer tx.Rollback(ctx)
	e = tx.QueryRow(ctx, `INSERT INTO documents(firm_id,matter_id,client_id,title,description,category,client_visible,created_by)VALUES($1,$2,$3,$4,$5,$6,$7,$8)RETURNING id,created_at`, firmID, d.MatterID, d.ClientID, d.Title, d.Description, d.Category, d.ClientVisible, userID).Scan(&d.ID, &d.CreatedAt)
	if e != nil {
		return d, mapError(e)
	}
	var versionID string
	e = tx.QueryRow(ctx, `INSERT INTO document_versions(document_id,firm_id,version_number,original_file_name,storage_key,mime_type,size_bytes,checksum,created_by)VALUES($1,$2,1,$3,$4,$5,$6,$7,$8)RETURNING id`, d.ID, firmID, d.OriginalFileName, storageKey, d.MimeType, d.SizeBytes, d.Checksum, userID).Scan(&versionID)
	if e != nil {
		return d, mapError(e)
	}
	if _, e = tx.Exec(ctx, `UPDATE documents SET current_version_id=$1 WHERE id=$2 AND firm_id=$3`, versionID, d.ID, firmID); e != nil {
		return d, e
	}
	if d.MatterID != nil {
		_, e = tx.Exec(ctx, `INSERT INTO matter_events(firm_id,matter_id,type,summary,resource_type,resource_id,actor_id,client_visible)VALUES($1,$2,'document.created',$3,'document',$4,$5,$6)`, firmID, *d.MatterID, "Document uploaded: "+d.Title, d.ID, userID, d.ClientVisible)
		if e != nil {
			return d, e
		}
	}
	if e = tx.Commit(ctx); e != nil {
		return d, e
	}
	d.VersionNumber = 1
	return d, nil
}
func (s *Store) AddDocumentVersion(ctx context.Context, firmID, userID, documentID, fileName, key, mime string, size int64, checksum string, notes *string) (domain.DocumentVersion, error) {
	tx, e := s.Pool.Begin(ctx)
	if e != nil {
		return domain.DocumentVersion{}, e
	}
	defer tx.Rollback(ctx)
	var v domain.DocumentVersion
	e = tx.QueryRow(ctx, `INSERT INTO document_versions(document_id,firm_id,version_number,original_file_name,storage_key,mime_type,size_bytes,checksum,created_by,notes)SELECT $1,$2,COALESCE(max(version_number),0)+1,$3,$4,$5,$6,$7,$8,$9 FROM document_versions WHERE document_id=$1 AND firm_id=$2 RETURNING id,version_number,created_at`, documentID, firmID, fileName, key, mime, size, checksum, userID, notes).Scan(&v.ID, &v.VersionNumber, &v.CreatedAt)
	if e != nil {
		return v, mapError(e)
	}
	if _, e = tx.Exec(ctx, `UPDATE documents SET current_version_id=$1,updated_at=now() WHERE id=$2 AND firm_id=$3 AND deleted_at IS NULL`, v.ID, documentID, firmID); e != nil {
		return v, e
	}
	var matterID *string
	var title string
	if e = tx.QueryRow(ctx, `SELECT matter_id,title FROM documents WHERE id=$1 AND firm_id=$2`, documentID, firmID).Scan(&matterID, &title); e != nil {
		return v, e
	}
	if matterID != nil {
		_, e = tx.Exec(ctx, `INSERT INTO matter_events(firm_id,matter_id,type,summary,resource_type,resource_id,actor_id)VALUES($1,$2,'document.version_added',$3,'document',$4,$5)`, firmID, *matterID, "Version added: "+title, documentID, userID)
		if e != nil {
			return v, e
		}
	}
	if e = tx.Commit(ctx); e != nil {
		return v, e
	}
	v.OriginalFileName = fileName
	v.MimeType = mime
	v.SizeBytes = size
	v.Checksum = checksum
	v.Notes = notes
	return v, nil
}
func (s *Store) Documents(ctx context.Context, firmID, userID, q string, matterID *string, limit, offset int) ([]domain.Document, error) {
	rows, e := s.Pool.Query(ctx, `SELECT d.id,d.matter_id,d.client_id,d.title,d.description,d.category,v.version_number,v.original_file_name,v.mime_type,v.size_bytes,v.checksum,d.client_visible,d.created_at FROM documents d JOIN document_versions v ON v.id=d.current_version_id AND v.firm_id=d.firm_id LEFT JOIN matters m ON m.id=d.matter_id AND m.firm_id=d.firm_id WHERE d.firm_id=$1 AND d.deleted_at IS NULL AND ($2='' OR d.title ILIKE '%'||$2||'%') AND ($3::uuid IS NULL OR d.matter_id=$3) AND (d.matter_id IS NULL OR m.confidentiality='normal' OR m.responsible_user_id=$4 OR m.created_by=$4 OR (m.confidentiality='partners_only' AND EXISTS(SELECT 1 FROM user_roles ur JOIN roles r ON r.id=ur.role_id AND r.firm_id=ur.firm_id WHERE ur.firm_id=$1 AND ur.user_id=$4 AND r.name IN('Owner','Partner','Administrator'))) OR EXISTS(SELECT 1 FROM matter_access ma WHERE ma.matter_id=m.id AND ma.firm_id=m.firm_id AND (ma.user_id=$4 OR ma.role_id IN(SELECT role_id FROM user_roles WHERE firm_id=$1 AND user_id=$4)))) ORDER BY d.updated_at DESC LIMIT $5 OFFSET $6`, firmID, q, matterID, userID, limit, offset)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	items := []domain.Document{}
	for rows.Next() {
		var d domain.Document
		if e = rows.Scan(&d.ID, &d.MatterID, &d.ClientID, &d.Title, &d.Description, &d.Category, &d.VersionNumber, &d.OriginalFileName, &d.MimeType, &d.SizeBytes, &d.Checksum, &d.ClientVisible, &d.CreatedAt); e != nil {
			return nil, e
		}
		items = append(items, d)
	}
	return items, rows.Err()
}

func (s *Store) DeletedDocuments(ctx context.Context, firmID, userID string) ([]domain.Document, error) {
	rows, err := s.Pool.Query(ctx, `SELECT d.id,d.matter_id,d.client_id,d.title,d.description,d.category,v.version_number,v.original_file_name,v.mime_type,v.size_bytes,v.checksum,d.client_visible,d.created_at,d.deleted_at FROM documents d JOIN document_versions v ON v.id=d.current_version_id AND v.firm_id=d.firm_id LEFT JOIN matters m ON m.id=d.matter_id AND m.firm_id=d.firm_id WHERE d.firm_id=$1 AND d.deleted_at IS NOT NULL AND (d.matter_id IS NULL OR m.confidentiality='normal' OR m.responsible_user_id=$2 OR m.created_by=$2 OR (m.confidentiality='partners_only' AND EXISTS(SELECT 1 FROM user_roles ur JOIN roles r ON r.id=ur.role_id AND r.firm_id=ur.firm_id WHERE ur.firm_id=$1 AND ur.user_id=$2 AND r.name IN('Owner','Partner','Administrator'))) OR EXISTS(SELECT 1 FROM matter_access ma WHERE ma.matter_id=m.id AND ma.firm_id=m.firm_id AND (ma.user_id=$2 OR ma.role_id IN(SELECT role_id FROM user_roles WHERE firm_id=$1 AND user_id=$2)))) ORDER BY d.deleted_at DESC LIMIT 200`, firmID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.Document{}
	for rows.Next() {
		var document domain.Document
		if err = rows.Scan(&document.ID, &document.MatterID, &document.ClientID, &document.Title, &document.Description, &document.Category, &document.VersionNumber, &document.OriginalFileName, &document.MimeType, &document.SizeBytes, &document.Checksum, &document.ClientVisible, &document.CreatedAt, &document.DeletedAt); err != nil {
			return nil, err
		}
		items = append(items, document)
	}
	return items, rows.Err()
}
func (s *Store) DocumentVersion(ctx context.Context, firmID, userID, documentID string, version *int) (domain.Document, string, error) {
	query := `SELECT d.id,d.matter_id,d.client_id,d.title,d.description,d.category,v.version_number,v.original_file_name,v.storage_key,v.mime_type,v.size_bytes,v.checksum,d.client_visible,d.created_at FROM documents d JOIN document_versions v ON v.document_id=d.id AND v.firm_id=d.firm_id LEFT JOIN matters m ON m.id=d.matter_id AND m.firm_id=d.firm_id WHERE d.firm_id=$1 AND d.id=$2 AND d.deleted_at IS NULL AND ($3::int IS NULL OR v.version_number=$3) AND ($3::int IS NOT NULL OR v.id=d.current_version_id) AND (d.matter_id IS NULL OR m.confidentiality='normal' OR m.responsible_user_id=$4 OR m.created_by=$4 OR (m.confidentiality='partners_only' AND EXISTS(SELECT 1 FROM user_roles ur JOIN roles r ON r.id=ur.role_id AND r.firm_id=ur.firm_id WHERE ur.firm_id=$1 AND ur.user_id=$4 AND r.name IN('Owner','Partner','Administrator'))) OR EXISTS(SELECT 1 FROM matter_access ma WHERE ma.matter_id=m.id AND ma.firm_id=m.firm_id AND (ma.user_id=$4 OR ma.role_id IN(SELECT role_id FROM user_roles WHERE firm_id=$1 AND user_id=$4))))`
	var d domain.Document
	var key string
	e := s.Pool.QueryRow(ctx, query, firmID, documentID, version, userID).Scan(&d.ID, &d.MatterID, &d.ClientID, &d.Title, &d.Description, &d.Category, &d.VersionNumber, &d.OriginalFileName, &key, &d.MimeType, &d.SizeBytes, &d.Checksum, &d.ClientVisible, &d.CreatedAt)
	if errors.Is(e, pgx.ErrNoRows) {
		return d, "", ErrNotFound
	}
	return d, key, e
}

func (s *Store) DocumentMatter(ctx context.Context, firmID, documentID string, includeDeleted bool) (*string, error) {
	var matterID *string
	err := s.Pool.QueryRow(ctx, `SELECT matter_id FROM documents WHERE firm_id=$1 AND id=$2 AND ($3 OR deleted_at IS NULL)`, firmID, documentID, includeDeleted).Scan(&matterID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return matterID, err
}

func (s *Store) SoftDeleteDocument(ctx context.Context, firmID, userID, documentID string) error {
	result, err := s.Pool.Exec(ctx, `UPDATE documents SET deleted_at=now(),deleted_by=$3,updated_at=now() WHERE firm_id=$1 AND id=$2 AND deleted_at IS NULL`, firmID, documentID, userID)
	if err != nil {
		return mapError(err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) RestoreDocument(ctx context.Context, firmID, documentID string) error {
	result, err := s.Pool.Exec(ctx, `UPDATE documents SET deleted_at=NULL,deleted_by=NULL,updated_at=now() WHERE firm_id=$1 AND id=$2 AND deleted_at IS NOT NULL`, firmID, documentID)
	if err != nil {
		return mapError(err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
func (s *Store) Versions(ctx context.Context, firmID, documentID string) ([]domain.DocumentVersion, error) {
	rows, e := s.Pool.Query(ctx, `SELECT v.id,v.version_number,v.original_file_name,v.mime_type,v.size_bytes,v.checksum,u.name,v.created_at,v.notes FROM document_versions v JOIN users u ON u.id=v.created_by AND u.firm_id=v.firm_id JOIN documents d ON d.id=v.document_id AND d.firm_id=v.firm_id WHERE v.firm_id=$1 AND v.document_id=$2 AND d.deleted_at IS NULL ORDER BY v.version_number DESC`, firmID, documentID)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	items := []domain.DocumentVersion{}
	for rows.Next() {
		var v domain.DocumentVersion
		if e = rows.Scan(&v.ID, &v.VersionNumber, &v.OriginalFileName, &v.MimeType, &v.SizeBytes, &v.Checksum, &v.CreatedByName, &v.CreatedAt, &v.Notes); e != nil {
			return nil, e
		}
		items = append(items, v)
	}
	return items, rows.Err()
}
func (s *Store) Deadlines(ctx context.Context, firmID, userID string, matterID *string, assignedOnly bool) ([]domain.Deadline, error) {
	rows, e := s.Pool.Query(ctx, `SELECT d.id,d.matter_id,m.title,d.title,d.description,d.due_at,d.status,d.priority,d.assigned_to,u.name,d.completed_at FROM deadlines d JOIN matters m ON m.id=d.matter_id AND m.firm_id=d.firm_id LEFT JOIN users u ON u.id=d.assigned_to WHERE d.firm_id=$1 AND ($2::uuid IS NULL OR d.matter_id=$2) AND (NOT $3 OR d.assigned_to=$4) AND (m.confidentiality='normal' OR m.responsible_user_id=$4 OR m.created_by=$4 OR (m.confidentiality='partners_only' AND EXISTS(SELECT 1 FROM user_roles ur JOIN roles r ON r.id=ur.role_id AND r.firm_id=ur.firm_id WHERE ur.firm_id=$1 AND ur.user_id=$4 AND r.name IN('Owner','Partner','Administrator'))) OR EXISTS(SELECT 1 FROM matter_access ma WHERE ma.matter_id=m.id AND ma.firm_id=m.firm_id AND (ma.user_id=$4 OR ma.role_id IN(SELECT role_id FROM user_roles WHERE firm_id=$1 AND user_id=$4)))) ORDER BY d.status='open' DESC,d.due_at`, firmID, matterID, assignedOnly, userID)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	items := []domain.Deadline{}
	for rows.Next() {
		var x domain.Deadline
		if e = rows.Scan(&x.ID, &x.MatterID, &x.MatterTitle, &x.Title, &x.Description, &x.DueAt, &x.Status, &x.Priority, &x.AssignedTo, &x.AssigneeName, &x.CompletedAt); e != nil {
			return nil, e
		}
		items = append(items, x)
	}
	return items, rows.Err()
}
func (s *Store) Tasks(ctx context.Context, firmID, userID string, matterID *string, assignedOnly bool) ([]domain.Task, error) {
	rows, e := s.Pool.Query(ctx, `SELECT t.id,t.matter_id,m.title,t.title,t.description,t.assigned_to,u.name,t.status,t.priority,t.due_at FROM tasks t LEFT JOIN matters m ON m.id=t.matter_id AND m.firm_id=t.firm_id LEFT JOIN users u ON u.id=t.assigned_to WHERE t.firm_id=$1 AND ($2::uuid IS NULL OR t.matter_id=$2) AND (NOT $3 OR t.assigned_to=$4) AND (t.matter_id IS NULL OR m.confidentiality='normal' OR m.responsible_user_id=$4 OR m.created_by=$4 OR (m.confidentiality='partners_only' AND EXISTS(SELECT 1 FROM user_roles ur JOIN roles r ON r.id=ur.role_id AND r.firm_id=ur.firm_id WHERE ur.firm_id=$1 AND ur.user_id=$4 AND r.name IN('Owner','Partner','Administrator'))) OR EXISTS(SELECT 1 FROM matter_access ma WHERE ma.matter_id=m.id AND ma.firm_id=m.firm_id AND (ma.user_id=$4 OR ma.role_id IN(SELECT role_id FROM user_roles WHERE firm_id=$1 AND user_id=$4)))) ORDER BY t.status IN('todo','in_progress','blocked') DESC,t.due_at NULLS LAST`, firmID, matterID, assignedOnly, userID)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	items := []domain.Task{}
	for rows.Next() {
		var x domain.Task
		if e = rows.Scan(&x.ID, &x.MatterID, &x.MatterTitle, &x.Title, &x.Description, &x.AssignedTo, &x.AssigneeName, &x.Status, &x.Priority, &x.DueAt); e != nil {
			return nil, e
		}
		items = append(items, x)
	}
	return items, rows.Err()
}
func (s *Store) Hearings(ctx context.Context, firmID, userID string, from, to time.Time) ([]domain.Hearing, error) {
	rows, e := s.Pool.Query(ctx, `SELECT h.id,h.matter_id,m.title,h.title,h.type,h.scheduled_at,h.location,h.status FROM hearings h JOIN matters m ON m.id=h.matter_id AND m.firm_id=h.firm_id WHERE h.firm_id=$1 AND h.scheduled_at BETWEEN $2 AND $3 AND (m.confidentiality='normal' OR m.responsible_user_id=$4 OR m.created_by=$4 OR (m.confidentiality='partners_only' AND EXISTS(SELECT 1 FROM user_roles ur JOIN roles r ON r.id=ur.role_id AND r.firm_id=ur.firm_id WHERE ur.firm_id=$1 AND ur.user_id=$4 AND r.name IN('Owner','Partner','Administrator'))) OR EXISTS(SELECT 1 FROM matter_access ma WHERE ma.matter_id=m.id AND ma.firm_id=m.firm_id AND (ma.user_id=$4 OR ma.role_id IN(SELECT role_id FROM user_roles WHERE firm_id=$1 AND user_id=$4)))) ORDER BY h.scheduled_at`, firmID, from, to, userID)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	items := []domain.Hearing{}
	for rows.Next() {
		var x domain.Hearing
		if e = rows.Scan(&x.ID, &x.MatterID, &x.MatterTitle, &x.Title, &x.Type, &x.ScheduledAt, &x.Location, &x.Status); e != nil {
			return nil, e
		}
		items = append(items, x)
	}
	return items, rows.Err()
}
func (s *Store) Financial(ctx context.Context, firmID, matterID string) (domain.FinancialSummary, error) {
	var x domain.FinancialSummary
	e := s.Pool.QueryRow(ctx, `SELECT COALESCE(sum(amount_cents)FILTER(WHERE type IN('fee','success_fee')),0),COALESCE(sum(amount_cents)FILTER(WHERE type='payment'),0),COALESCE(sum(amount_cents)FILTER(WHERE type='expense'),0),COALESCE(sum(amount_cents)FILTER(WHERE type='court_cost'),0),COALESCE(sum(amount_cents)FILTER(WHERE type='reimbursement'),0) FROM matter_financial_entries WHERE firm_id=$1 AND matter_id=$2`, firmID, matterID).Scan(&x.FeesCents, &x.PaymentsCents, &x.ExpensesCents, &x.CourtCostsCents, &x.ReimbursementsCents)
	x.PendingCents = x.FeesCents - x.PaymentsCents
	x.NetCents = x.PaymentsCents + x.ReimbursementsCents - x.ExpensesCents - x.CourtCostsCents
	return x, e
}
func (s *Store) Workflows(ctx context.Context, firmID string) ([]domain.Workflow, error) {
	rows, e := s.Pool.Query(ctx, `SELECT id,name,description,active FROM workflow_definitions WHERE firm_id=$1 ORDER BY name`, firmID)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	items := []domain.Workflow{}
	for rows.Next() {
		var w domain.Workflow
		if e = rows.Scan(&w.ID, &w.Name, &w.Description, &w.Active); e != nil {
			return nil, e
		}
		w.Stages, _ = s.WorkflowStages(ctx, firmID, w.ID)
		items = append(items, w)
	}
	return items, rows.Err()
}
func (s *Store) WorkflowStages(ctx context.Context, firmID, workflowID string) ([]domain.WorkflowStage, error) {
	rows, e := s.Pool.Query(ctx, `SELECT id,name,description,color,sort_order,checklist,on_enter_tasks FROM workflow_stages WHERE firm_id=$1 AND workflow_id=$2 ORDER BY sort_order`, firmID, workflowID)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	items := []domain.WorkflowStage{}
	for rows.Next() {
		var x domain.WorkflowStage
		if e = rows.Scan(&x.ID, &x.Name, &x.Description, &x.Color, &x.SortOrder, &x.Checklist, &x.OnEnterTasks); e != nil {
			return nil, e
		}
		items = append(items, x)
	}
	return items, rows.Err()
}
