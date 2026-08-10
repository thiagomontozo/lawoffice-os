package repository

import (
	"strings"
	"testing"
)

func TestSplitExtractionPageIsBoundedAndOverlapping(t *testing.T) {
	content := strings.Repeat("cláusula de exemplo. ", 300)
	chunks := splitExtractionPage(content)
	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}
	for _, chunk := range chunks {
		if len([]rune(chunk)) > chunkSizeRunes {
			t.Fatalf("chunk exceeded limit: %d", len([]rune(chunk)))
		}
	}
	if !strings.Contains(chunks[1], "cláusula") {
		t.Fatal("overlap lost readable boundary context")
	}
}
