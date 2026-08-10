package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const maxProviderResponseBytes = 4 << 20

type OpenAI struct {
	APIKey             string
	BaseURL            string
	Model              string
	EmbeddingModelName string
	Client             *http.Client
}

func (client *OpenAI) Generate(ctx context.Context, request GenerateRequest) (GenerateResult, error) {
	payload := struct {
		Model            string `json:"model"`
		Instructions     string `json:"instructions"`
		Input            string `json:"input"`
		Store            bool   `json:"store"`
		SafetyIdentifier string `json:"safety_identifier,omitempty"`
	}{Model: client.Model, Instructions: request.Instructions, Input: request.Input, Store: false, SafetyIdentifier: request.SafetyIdentifier}
	var response struct {
		Model  string `json:"model"`
		Output []struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
	}
	if err := client.post(ctx, "/responses", payload, &response); err != nil {
		return GenerateResult{}, err
	}
	var text strings.Builder
	for _, output := range response.Output {
		for _, content := range output.Content {
			if content.Type == "output_text" {
				text.WriteString(content.Text)
			}
		}
	}
	answer := strings.TrimSpace(text.String())
	if answer == "" {
		return GenerateResult{}, fmt.Errorf("%w: empty generation", ErrProvider)
	}
	model := response.Model
	if model == "" {
		model = client.Model
	}
	return GenerateResult{Text: answer, Model: model}, nil
}

func (client *OpenAI) Embed(ctx context.Context, input []string) ([][]float32, error) {
	if len(input) == 0 || len(input) > 100 {
		return nil, fmt.Errorf("%w: invalid embedding batch", ErrProvider)
	}
	payload := struct {
		Model string   `json:"model"`
		Input []string `json:"input"`
	}{Model: client.EmbeddingModelName, Input: input}
	var response struct {
		Data []struct {
			Index     int       `json:"index"`
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := client.post(ctx, "/embeddings", payload, &response); err != nil {
		return nil, err
	}
	if len(response.Data) != len(input) {
		return nil, fmt.Errorf("%w: incomplete embedding batch", ErrProvider)
	}
	result := make([][]float32, len(input))
	for _, item := range response.Data {
		if item.Index < 0 || item.Index >= len(result) || len(item.Embedding) == 0 || result[item.Index] != nil {
			return nil, fmt.Errorf("%w: invalid embedding response", ErrProvider)
		}
		result[item.Index] = item.Embedding
	}
	return result, nil
}

func (client *OpenAI) EmbeddingModel() string { return client.EmbeddingModelName }

func (client *OpenAI) post(ctx context.Context, path string, payload, destination any) error {
	if client.Client == nil || strings.TrimSpace(client.APIKey) == "" || strings.TrimSpace(client.BaseURL) == "" {
		return ErrDisabled
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(client.BaseURL, "/")+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+client.APIKey)
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Client.Do(request)
	if err != nil {
		return fmt.Errorf("%w: transport", ErrProvider)
	}
	defer response.Body.Close()
	limited := io.LimitReader(response.Body, maxProviderResponseBytes+1)
	responseBody, err := io.ReadAll(limited)
	if err != nil {
		return fmt.Errorf("%w: response read", ErrProvider)
	}
	if len(responseBody) > maxProviderResponseBytes {
		return fmt.Errorf("%w: response too large", ErrProvider)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("%w: HTTP %d", ErrProvider, response.StatusCode)
	}
	if err = json.Unmarshal(responseBody, destination); err != nil {
		return fmt.Errorf("%w: invalid response", ErrProvider)
	}
	return nil
}

func IsOperationalError(err error) bool {
	return errors.Is(err, ErrDisabled) || errors.Is(err, ErrProvider) || errors.Is(err, context.DeadlineExceeded)
}
