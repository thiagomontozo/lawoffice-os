package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thiagomontozo/lawoffice-os/backend/internal/scanner"
	"github.com/thiagomontozo/lawoffice-os/backend/internal/storage"
)

type threatScanner struct{}

func (threatScanner) Scan(context.Context, io.Reader) error { return scanner.ErrThreat }
func (threatScanner) Health(context.Context) error          { return nil }

func uploadHeader(t *testing.T, name string, content []byte) *multipart.FileHeader {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = part.Write(content); err != nil {
		t.Fatal(err)
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("POST", "/", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	if err = request.ParseMultipartForm(int64(body.Len())); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = request.MultipartForm.RemoveAll() })
	return request.MultipartForm.File["file"][0]
}

func TestStoreUploadStreamsAndCalculatesIntegrity(t *testing.T) {
	root := t.TempDir()
	objects, err := storage.NewLocal(root)
	if err != nil {
		t.Fatal(err)
	}
	service := New(nil, objects, nil, 1024*1024)
	content := []byte("legal document content")
	key, mimeType, name, size, checksum, err := service.storeUpload(context.Background(), "firm-id", uploadHeader(t, "memo.txt", content))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(key, "firm-id_") || filepath.Ext(key) != ".txt" {
		t.Fatalf("unexpected isolated storage key: %q", key)
	}
	if mimeType != "text/plain; charset=utf-8" || name != "memo.txt" || size != int64(len(content)) {
		t.Fatalf("unexpected metadata: mime=%q name=%q size=%d", mimeType, name, size)
	}
	want := sha256.Sum256(content)
	if checksum != hex.EncodeToString(want[:]) {
		t.Fatalf("checksum mismatch: got %s", checksum)
	}
}

func TestStoreUploadRejectsExtensionMismatch(t *testing.T) {
	objects, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	service := New(nil, objects, nil, 1024)
	_, _, _, _, _, err = service.storeUpload(context.Background(), "firm-id", uploadHeader(t, "malware.pdf", []byte("plain text")))
	if err == nil {
		t.Fatal("expected extension mismatch")
	}
}

func TestStoreUploadScansBeforeObjectCommit(t *testing.T) {
	root := t.TempDir()
	objects, err := storage.NewLocal(root)
	if err != nil {
		t.Fatal(err)
	}
	service := New(nil, objects, threatScanner{}, 1024)
	_, _, _, _, _, err = service.storeUpload(context.Background(), "firm-id", uploadHeader(t, "memo.txt", []byte("suspicious")))
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("threat should be rejected as validation error, got %v", err)
	}
	entries, readErr := os.ReadDir(root)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("scanner rejection committed %d storage objects", len(entries))
	}
}
