package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenAIResponsesAndEmbeddings(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer test-provider-key" {
			t.Fatal("missing provider authorization")
		}
		switch request.URL.Path {
		case "/responses":
			var body struct {
				Store bool `json:"store"`
			}
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body.Store {
				t.Fatal("response storage must remain disabled")
			}
			_, _ = response.Write([]byte("{\"model\":\"test-model\",\"output\":[{\"content\":[{\"type\":\"output_text\",\"text\":\"Answer [S1]\"}]}]}"))
		case "/embeddings":
			_, _ = response.Write([]byte("{\"data\":[{\"index\":1,\"embedding\":[0,1]},{\"index\":0,\"embedding\":[1,0]}]}"))
		default:
			http.NotFound(response, request)
		}
	}))
	defer server.Close()

	provider := &OpenAI{
		APIKey: "test-provider-key", BaseURL: server.URL, Model: "test-model",
		EmbeddingModelName: "test-embedding", Client: server.Client(),
	}
	generated, err := provider.Generate(context.Background(), GenerateRequest{Instructions: "grounded", Input: "question"})
	if err != nil || generated.Text != "Answer [S1]" || generated.Model != "test-model" {
		t.Fatalf("unexpected generation: result=%+v err=%v", generated, err)
	}
	embeddings, err := provider.Embed(context.Background(), []string{"first", "second"})
	if err != nil || len(embeddings) != 2 || embeddings[0][0] != 1 || embeddings[1][1] != 1 {
		t.Fatalf("unexpected embeddings: result=%+v err=%v", embeddings, err)
	}
}
