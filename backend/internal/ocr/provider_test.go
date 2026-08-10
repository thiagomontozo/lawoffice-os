package ocr

import (
	"archive/zip"
	"bytes"
	"context"
	"testing"
)

func TestBuiltinExtractsPlainText(t *testing.T) {
	provider := Builtin{MaxBytes: 1024}
	result, err := provider.Extract(context.Background(), Document{Reader: bytes.NewBufferString("  legal evidence  "), MimeType: "text/plain", FileName: "note.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Pages) != 1 || result.Pages[0].Text != "legal evidence" {
		t.Fatalf("unexpected extraction: %+v", result)
	}
}

func TestBuiltinExtractsDOCXText(t *testing.T) {
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	file, err := writer.Create("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = file.Write([]byte(`<w:document xmlns:w="urn:test"><w:body><w:p><w:r><w:t>First clause</w:t></w:r></w:p><w:p><w:r><w:t>Second clause</w:t></w:r></w:p></w:body></w:document>`)); err != nil {
		t.Fatal(err)
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}
	result, err := (Builtin{MaxBytes: 4096}).Extract(context.Background(), Document{Reader: bytes.NewReader(archive.Bytes()), MimeType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document", FileName: "contract.docx"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Pages) != 1 || result.Pages[0].Text != "First clause \nSecond clause" {
		t.Fatalf("unexpected DOCX extraction: %q", result.Pages[0].Text)
	}
}

func TestBuiltinRejectsUnsupportedImage(t *testing.T) {
	_, err := (Builtin{MaxBytes: 1024}).Extract(context.Background(), Document{Reader: bytes.NewReader([]byte("image")), MimeType: "image/png", FileName: "scan.png"})
	if err != ErrUnsupported {
		t.Fatalf("expected ErrUnsupported, got %v", err)
	}
}
