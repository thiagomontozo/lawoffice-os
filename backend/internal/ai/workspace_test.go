package ai

import (
	"strings"
	"testing"

	"github.com/thiagomontozo/lawoffice-os/backend/internal/domain"
)

func TestHybridRankingAndCitationLineage(t *testing.T) {
	chunks := []domain.DocumentChunk{
		{ID: "lexical", DocumentID: "document-a", DocumentVersionID: "version-a", DocumentTitle: "Contrato", PageNumber: 2, CharacterCount: 20, Content: "prazo contratual", KeywordScore: 1, Embedding: []float32{0, 1}},
		{ID: "semantic", DocumentID: "document-b", DocumentVersionID: "version-b", DocumentTitle: "Aditivo", PageNumber: 4, CharacterCount: 20, Content: "vencimento ajustado", KeywordScore: 0, Embedding: []float32{1, 0}},
	}
	ranked := rank(chunks, []float32{1, 0})
	if ranked[0].chunk.ID != "semantic" {
		t.Fatalf("semantic candidate should rank first, got %s", ranked[0].chunk.ID)
	}
	prompt, citations := buildContext("qual prazo?", ranked)
	if len(citations) != 2 || citations[0].DocumentVersion != "version-b" || citations[0].PageNumber != 4 {
		t.Fatalf("citation lineage was lost: %+v", citations)
	}
	if !strings.Contains(prompt, "[S1]") || !strings.Contains(prompt, "immutable version: version-b") {
		t.Fatalf("prompt does not preserve source boundary: %s", prompt)
	}
}
