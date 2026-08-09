package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/domain"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/repository"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/storage"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
)

var ErrValidation = errors.New("validation error")

type Service struct {
	Store     *repository.Store
	Storage   storage.ObjectStorage
	MaxUpload int64
}

func New(r *repository.Store, s storage.ObjectStorage, max int64) *Service {
	return &Service{r, s, max}
}
func (s *Service) RequirePermission(ctx context.Context, firmID, userID, key string) error {
	ok, e := s.Store.HasPermission(ctx, firmID, userID, key)
	if e != nil {
		return e
	}
	if !ok {
		return repository.ErrForbidden
	}
	return nil
}
func (s *Service) MatterDetail(ctx context.Context, firmID, userID, id string) (domain.MatterDetail, error) {
	ok, e := s.Store.CanAccessMatter(ctx, firmID, userID, id, "read")
	if e != nil || !ok {
		return domain.MatterDetail{}, repository.ErrForbidden
	}
	m, e := s.Store.Matter(ctx, firmID, id)
	if e != nil {
		return domain.MatterDetail{}, e
	}
	timeline, e := s.Store.Timeline(ctx, firmID, id, userID)
	if e != nil {
		return domain.MatterDetail{}, e
	}
	docs, e := s.Store.Documents(ctx, firmID, userID, "", &id, 100, 0)
	if e != nil {
		return domain.MatterDetail{}, e
	}
	deadlines, e := s.Store.Deadlines(ctx, firmID, userID, &id, false)
	if e != nil {
		return domain.MatterDetail{}, e
	}
	tasks, e := s.Store.Tasks(ctx, firmID, userID, &id, false)
	if e != nil {
		return domain.MatterDetail{}, e
	}
	parties, e := s.Store.Parties(ctx, firmID, id)
	if e != nil {
		return domain.MatterDetail{}, e
	}
	notes, e := s.Store.Notes(ctx, firmID, id, userID)
	if e != nil {
		return domain.MatterDetail{}, e
	}
	financial, e := s.Store.Financial(ctx, firmID, id)
	return domain.MatterDetail{Matter: m, Timeline: timeline, Documents: docs, Deadlines: deadlines, Tasks: tasks, Parties: parties, Notes: notes, Financial: financial}, e
}

type Upload struct {
	FirmID, UserID, Title, Category string
	MatterID, ClientID              *string
	Description                     *string
	ClientVisible                   bool
	Header                          *multipart.FileHeader
	Notes                           *string
}

var fileTypes = map[string]string{"application/pdf": ".pdf", "application/vnd.openxmlformats-officedocument.wordprocessingml.document": ".docx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": ".xlsx", "image/png": ".png", "image/jpeg": ".jpg", "image/webp": ".webp", "text/plain; charset=utf-8": ".txt", "text/plain": ".txt"}

type countWriter struct{ n int64 }

func (w *countWriter) Write(p []byte) (int, error) {
	w.n += int64(len(p))
	return len(p), nil
}

func (s *Service) storeUpload(ctx context.Context, firmID string, h *multipart.FileHeader) (string, string, string, int64, string, error) {
	if h == nil || h.Size < 1 || h.Size > s.MaxUpload {
		return "", "", "", 0, "", fmt.Errorf("%w: invalid file size", ErrValidation)
	}
	f, e := h.Open()
	if e != nil {
		return "", "", "", 0, "", e
	}
	defer f.Close()
	limited := io.LimitReader(f, s.MaxUpload+1)
	prefix := make([]byte, 512)
	n, readErr := io.ReadFull(limited, prefix)
	if readErr != nil && !errors.Is(readErr, io.EOF) && !errors.Is(readErr, io.ErrUnexpectedEOF) {
		return "", "", "", 0, "", readErr
	}
	if n == 0 {
		return "", "", "", 0, "", fmt.Errorf("%w: empty upload", ErrValidation)
	}
	mime := http.DetectContentType(prefix[:n])
	ext, ok := fileTypes[mime]
	if !ok {
		return "", "", "", 0, "", fmt.Errorf("%w: unsupported MIME type", ErrValidation)
	}
	provided := strings.ToLower(filepath.Ext(h.Filename))
	if mime == "image/jpeg" {
		if provided != ".jpg" && provided != ".jpeg" {
			return "", "", "", 0, "", fmt.Errorf("%w: extension mismatch", ErrValidation)
		}
	} else if provided != ext {
		return "", "", "", 0, "", fmt.Errorf("%w: extension mismatch", ErrValidation)
	}
	hasher := sha256.New()
	counter := &countWriter{}
	source := io.MultiReader(bytes.NewReader(prefix[:n]), limited)
	key := firmID + "_" + uuid.NewString() + ext
	if e = s.Storage.Put(ctx, key, io.TeeReader(source, io.MultiWriter(hasher, counter))); e != nil {
		return "", "", "", 0, "", e
	}
	if counter.n < 1 || counter.n > s.MaxUpload {
		_ = s.Storage.Delete(ctx, key)
		return "", "", "", 0, "", fmt.Errorf("%w: upload too large", ErrValidation)
	}
	return key, mime, safeName(h.Filename), counter.n, hex.EncodeToString(hasher.Sum(nil)), nil
}
func (s *Service) UploadDocument(ctx context.Context, u Upload) (domain.Document, error) {
	u.Title = strings.TrimSpace(u.Title)
	if u.Title == "" || len([]rune(u.Title)) > 220 {
		return domain.Document{}, ErrValidation
	}
	categories := map[string]bool{"petition": true, "contract": true, "evidence": true, "power_of_attorney": true, "decision": true, "judgment": true, "correspondence": true, "invoice": true, "receipt": true, "report": true, "internal": true, "other": true}
	if !categories[u.Category] {
		return domain.Document{}, fmt.Errorf("%w: unsupported document category", ErrValidation)
	}
	if u.MatterID != nil {
		ok, e := s.Store.CanAccessMatter(ctx, u.FirmID, u.UserID, *u.MatterID, "write")
		if e != nil || !ok {
			return domain.Document{}, repository.ErrForbidden
		}
	}
	key, mime, originalName, size, checksum, e := s.storeUpload(ctx, u.FirmID, u.Header)
	if e != nil {
		return domain.Document{}, e
	}
	d := domain.Document{MatterID: u.MatterID, ClientID: u.ClientID, Title: u.Title, Description: u.Description, Category: u.Category, OriginalFileName: originalName, MimeType: mime, SizeBytes: size, Checksum: checksum, ClientVisible: u.ClientVisible}
	created, e := s.Store.CreateDocument(ctx, u.FirmID, u.UserID, d, key)
	if e != nil {
		_ = s.Storage.Delete(ctx, key)
		return d, e
	}
	return created, nil
}
func (s *Service) AddVersion(ctx context.Context, firmID, userID, documentID string, h *multipart.FileHeader, notes *string) (domain.DocumentVersion, error) {
	current, _, e := s.Store.DocumentVersion(ctx, firmID, userID, documentID, nil)
	if e != nil {
		return domain.DocumentVersion{}, e
	}
	if current.MatterID != nil {
		ok, e := s.Store.CanAccessMatter(ctx, firmID, userID, *current.MatterID, "write")
		if e != nil || !ok {
			return domain.DocumentVersion{}, repository.ErrForbidden
		}
	}
	key, mime, originalName, size, checksum, e := s.storeUpload(ctx, firmID, h)
	if e != nil {
		return domain.DocumentVersion{}, e
	}
	v, e := s.Store.AddDocumentVersion(ctx, firmID, userID, documentID, originalName, key, mime, size, checksum, notes)
	if e != nil {
		_ = s.Storage.Delete(ctx, key)
	}
	return v, e
}
func (s *Service) OpenDocument(ctx context.Context, firmID, userID, id string, version *int) (domain.Document, io.ReadCloser, error) {
	d, key, e := s.Store.DocumentVersion(ctx, firmID, userID, id, version)
	if e != nil {
		return d, nil, e
	}
	f, e := s.Storage.Open(ctx, key)
	return d, f, e
}

func (s *Service) DeleteDocument(ctx context.Context, firmID, userID, id string) error {
	matterID, err := s.Store.DocumentMatter(ctx, firmID, id, false)
	if err != nil {
		return err
	}
	if matterID != nil {
		allowed, accessErr := s.Store.CanAccessMatter(ctx, firmID, userID, *matterID, "manage")
		if accessErr != nil || !allowed {
			return repository.ErrForbidden
		}
	}
	return s.Store.SoftDeleteDocument(ctx, firmID, userID, id)
}

func (s *Service) RestoreDocument(ctx context.Context, firmID, userID, id string) error {
	matterID, err := s.Store.DocumentMatter(ctx, firmID, id, true)
	if err != nil {
		return err
	}
	if matterID != nil {
		allowed, accessErr := s.Store.CanAccessMatter(ctx, firmID, userID, *matterID, "manage")
		if accessErr != nil || !allowed {
			return repository.ErrForbidden
		}
	}
	return s.Store.RestoreDocument(ctx, firmID, id)
}
func (s *Service) UpdateBranding(ctx context.Context, firmID, userID string, b domain.Branding) (domain.Branding, error) {
	hexColor := regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
	if strings.TrimSpace(b.SystemTitle) == "" || !hexColor.MatchString(b.PrimaryColor) || !hexColor.MatchString(b.SecondaryColor) || !hexColor.MatchString(b.AccentColor) {
		return b, ErrValidation
	}
	return s.Store.UpdateBranding(ctx, firmID, userID, b)
}
func (s *Service) UploadBrandAsset(ctx context.Context, firmID, userID, kind string, header *multipart.FileHeader) error {
	key, mimeType, _, _, _, err := s.storeUpload(ctx, firmID, header)
	if err != nil {
		return err
	}
	if mimeType != "image/png" && mimeType != "image/jpeg" && mimeType != "image/webp" {
		_ = s.Storage.Delete(ctx, key)
		return fmt.Errorf("%w: branding assets must be PNG, JPEG, or WEBP", ErrValidation)
	}
	if err := s.Store.SetBrandAsset(ctx, firmID, userID, kind, key); err != nil {
		_ = s.Storage.Delete(ctx, key)
		return err
	}
	return nil
}
func (s *Service) Archive(ctx context.Context, firmID, userID, matterID, reason, outcome, summary string, force bool) (map[string]any, error) {
	ok, e := s.Store.CanAccessMatter(ctx, firmID, userID, matterID, "manage")
	if e != nil || !ok {
		return nil, repository.ErrForbidden
	}
	allowed := map[string]bool{"completed": true, "settlement": true, "dismissed": true, "transferred": true, "withdrawn": true, "cancelled": true, "other": true}
	if !allowed[reason] || strings.TrimSpace(summary) == "" {
		return nil, ErrValidation
	}
	return s.Store.ArchiveMatter(ctx, firmID, userID, matterID, reason, outcome, summary, force)
}
func safeName(v string) string {
	v = filepath.Base(strings.ReplaceAll(v, "\\", "/"))
	v = strings.TrimSpace(v)
	if v == "" {
		return "document"
	}
	r := []rune(v)
	if len(r) > 255 {
		return string(r[:255])
	}
	return v
}
