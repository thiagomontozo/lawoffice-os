package storage

import (
	"context"
	"io"
	"strings"
	"testing"
)

func TestLocalStorageLifecycle(t *testing.T) {
	store, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err = store.Put(ctx, "document.bin", strings.NewReader("legal-content")); err != nil {
		t.Fatal(err)
	}
	reader, err := store.Open(ctx, "document.bin")
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(reader)
	reader.Close()
	if err != nil || string(body) != "legal-content" {
		t.Fatalf("unexpected body %q: %v", body, err)
	}
	if err = store.Delete(ctx, "document.bin"); err != nil {
		t.Fatal(err)
	}
	if _, err = store.Open(ctx, "document.bin"); err == nil {
		t.Fatal("deleted object should not open")
	}
}
func TestLocalStorageRejectsTraversal(t *testing.T) {
	store, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"../secret", "folder/file", "folder\\file", "C:\\secret"} {
		if err = store.Put(context.Background(), key, strings.NewReader("x")); err == nil {
			t.Fatalf("unsafe key accepted: %s", key)
		}
	}
}
