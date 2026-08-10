package jobs

import (
	"bytes"
	"io"
	"log/slog"
	"testing"
)

func TestJobPayloadEncryptionRoundTrip(t *testing.T) {
	queue, err := New(nil, "a-long-session-secret-for-job-encryption", nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	plaintext := []byte(`{"to":"client@example.test","text":"one-time-link"}`)
	encrypted, err := queue.seal(plaintext)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encrypted, []byte("client@example.test")) || bytes.Contains(encrypted, []byte("one-time-link")) {
		t.Fatal("encrypted job payload exposed sensitive content")
	}
	opened, err := queue.open(encrypted)
	if err != nil || !bytes.Equal(opened, plaintext) {
		t.Fatalf("payload round trip failed: %v", err)
	}
}
