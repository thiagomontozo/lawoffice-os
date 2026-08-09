package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/domain"
)

func (s *Store) Contacts(ctx context.Context, firmID string) ([]domain.Contact, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id,client_id,name,type,email,phone,document FROM contacts WHERE firm_id=$1 AND deleted_at IS NULL ORDER BY name`, firmID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.Contact{}
	for rows.Next() {
		var x domain.Contact
		if err = rows.Scan(&x.ID, &x.ClientID, &x.Name, &x.Type, &x.Email, &x.Phone, &x.Document); err != nil {
			return nil, err
		}
		items = append(items, x)
	}
	return items, rows.Err()
}
func (s *Store) CreateContact(ctx context.Context, firmID string, x domain.Contact) (domain.Contact, error) {
	err := s.Pool.QueryRow(ctx, `INSERT INTO contacts(firm_id,client_id,name,type,email,phone,document)VALUES($1,$2,$3,$4,$5,$6,$7)RETURNING id`, firmID, x.ClientID, x.Name, x.Type, x.Email, x.Phone, x.Document).Scan(&x.ID)
	return x, mapError(err)
}

func (s *Store) UpdateContact(ctx context.Context, firmID, id string, x domain.Contact) (domain.Contact, error) {
	err := s.Pool.QueryRow(ctx, `UPDATE contacts SET client_id=$3,name=$4,type=$5,email=$6,phone=$7,document=$8,notes=$9 WHERE firm_id=$1 AND id=$2 AND deleted_at IS NULL RETURNING id,client_id,name,type,email,phone,document`, firmID, id, x.ClientID, x.Name, x.Type, x.Email, x.Phone, x.Document, nil).Scan(&x.ID, &x.ClientID, &x.Name, &x.Type, &x.Email, &x.Phone, &x.Document)
	if errors.Is(err, pgx.ErrNoRows) {
		return x, ErrNotFound
	}
	return x, mapError(err)
}

func (s *Store) ArchiveContact(ctx context.Context, firmID, id string) error {
	r, err := s.Pool.Exec(ctx, `UPDATE contacts SET deleted_at=now() WHERE firm_id=$1 AND id=$2 AND deleted_at IS NULL`, firmID, id)
	if err != nil {
		return mapError(err)
	}
	if r.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
func (s *Store) AddParty(ctx context.Context, firmID, userID, matterID string, x domain.Party) (domain.Party, error) {
	err := s.Pool.QueryRow(ctx, `INSERT INTO matter_parties(firm_id,matter_id,name,document,role,side)VALUES($1,$2,$3,$4,$5,$6)RETURNING id`, firmID, matterID, x.Name, x.Document, x.Role, x.Side).Scan(&x.ID)
	if err == nil {
		_, _ = s.Pool.Exec(ctx, `INSERT INTO matter_events(firm_id,matter_id,type,summary,resource_type,resource_id,actor_id)VALUES($1,$2,'party.added',$3,'party',$4,$5)`, firmID, matterID, "Party added: "+x.Name, x.ID, userID)
	}
	return x, mapError(err)
}
func (s *Store) AddNote(ctx context.Context, firmID, userID, matterID string, x domain.Note) (domain.Note, error) {
	err := s.Pool.QueryRow(ctx, `INSERT INTO matter_notes(firm_id,matter_id,content,visibility,created_by)VALUES($1,$2,$3,$4,$5)RETURNING id,created_at`, firmID, matterID, x.Content, x.Visibility, userID).Scan(&x.ID, &x.CreatedAt)
	if err == nil {
		x.AuthorName = "You"
	}
	return x, mapError(err)
}
func (s *Store) GrantMatterAccess(ctx context.Context, firmID, matterID string, userID, roleID *string, level string) error {
	_, err := s.Pool.Exec(ctx, `INSERT INTO matter_access(firm_id,matter_id,user_id,role_id,access_level)VALUES($1,$2,$3,$4,$5) ON CONFLICT DO NOTHING`, firmID, matterID, userID, roleID, level)
	return mapError(err)
}
func (s *Store) CreateHearing(ctx context.Context, firmID, userID string, x domain.Hearing) (domain.Hearing, error) {
	err := s.Pool.QueryRow(ctx, `INSERT INTO hearings(firm_id,matter_id,title,type,scheduled_at,location,status,responsible_user_id)VALUES($1,$2,$3,$4,$5,$6,$7,$8)RETURNING id`, firmID, x.MatterID, x.Title, x.Type, x.ScheduledAt, x.Location, x.Status, userID).Scan(&x.ID)
	if err == nil {
		_, _ = s.Pool.Exec(ctx, `INSERT INTO matter_events(firm_id,matter_id,type,summary,resource_type,resource_id,actor_id)VALUES($1,$2,'hearing.created',$3,'hearing',$4,$5)`, firmID, x.MatterID, "Hearing scheduled: "+x.Title, x.ID, userID)
	}
	return x, mapError(err)
}
func (s *Store) CreateFinancialEntry(ctx context.Context, firmID, userID, matterID, kind, description string, amount int64, dueDate *time.Time) (string, error) {
	var id string
	err := s.Pool.QueryRow(ctx, `INSERT INTO matter_financial_entries(firm_id,matter_id,type,amount_cents,description,due_date,created_by)VALUES($1,$2,$3,$4,$5,$6,$7)RETURNING id`, firmID, matterID, kind, amount, description, dueDate, userID).Scan(&id)
	if err == nil {
		_, _ = s.Pool.Exec(ctx, `INSERT INTO matter_events(firm_id,matter_id,type,summary,resource_type,resource_id,actor_id)VALUES($1,$2,'finance.entry_added',$3,'finance',$4,$5)`, firmID, matterID, "Financial entry added: "+description, id, userID)
	}
	return id, mapError(err)
}
func (s *Store) CreateWorkflow(ctx context.Context, firmID, name string, description *string) (domain.Workflow, error) {
	var x domain.Workflow
	err := s.Pool.QueryRow(ctx, `INSERT INTO workflow_definitions(firm_id,name,description)VALUES($1,$2,$3)RETURNING id,active`, firmID, name, description).Scan(&x.ID, &x.Active)
	x.Name = name
	x.Description = description
	return x, mapError(err)
}
func (s *Store) ReplaceWorkflowStages(ctx context.Context, firmID, workflowID string, stages []domain.WorkflowStage) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `DELETE FROM workflow_stages WHERE firm_id=$1 AND workflow_id=$2`, firmID, workflowID); err != nil {
		return err
	}
	for index, x := range stages {
		checklist := x.Checklist
		if len(checklist) == 0 {
			checklist = json.RawMessage(`[]`)
		}
		tasks := x.OnEnterTasks
		if len(tasks) == 0 {
			tasks = json.RawMessage(`[]`)
		}
		if _, err = tx.Exec(ctx, `INSERT INTO workflow_stages(firm_id,workflow_id,name,description,color,sort_order,checklist,on_enter_tasks)VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, firmID, workflowID, x.Name, x.Description, x.Color, index, checklist, tasks); err != nil {
			return mapError(err)
		}
	}
	return tx.Commit(ctx)
}
func (s *Store) CreateCustomField(ctx context.Context, firmID, entityType, key, label, kind string, required bool, options json.RawMessage) (string, error) {
	if len(options) == 0 {
		options = json.RawMessage(`[]`)
	}
	var id string
	err := s.Pool.QueryRow(ctx, `INSERT INTO custom_field_definitions(firm_id,entity_type,key,label,type,required,options)VALUES($1,$2,$3,$4,$5,$6,$7)RETURNING id`, firmID, entityType, key, label, kind, required, options).Scan(&id)
	return id, mapError(err)
}
func (s *Store) CreateTag(ctx context.Context, firmID, name, color string) (string, error) {
	var id string
	err := s.Pool.QueryRow(ctx, `INSERT INTO tags(firm_id,name,color)VALUES($1,$2,$3)RETURNING id`, firmID, name, color).Scan(&id)
	return id, mapError(err)
}
func (s *Store) UpdateMatterStatus(ctx context.Context, firmID, userID, matterID, status string) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	result, err := tx.Exec(ctx, `UPDATE matters SET status=$3,updated_at=now() WHERE firm_id=$1 AND id=$2 AND deleted_at IS NULL`, firmID, matterID, status)
	if err != nil {
		return mapError(err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	_, err = tx.Exec(ctx, `INSERT INTO matter_events(firm_id,matter_id,type,summary,actor_id)VALUES($1,$2,'matter.status_changed',$3,$4)`, firmID, matterID, "Matter status changed to "+status, userID)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}
