package ai

import (
	"context"
	"errors"
)

var (
	ErrDisabled  = errors.New("AI workspace is disabled")
	ErrNoSources = errors.New("no extracted sources are available for this Matter")
	ErrProvider  = errors.New("AI provider request failed")
)

type GenerateRequest struct {
	Instructions     string
	Input            string
	SafetyIdentifier string
}

type GenerateResult struct {
	Text  string
	Model string
}

type Generator interface {
	Generate(context.Context, GenerateRequest) (GenerateResult, error)
}

type Embedder interface {
	Embed(context.Context, []string) ([][]float32, error)
	EmbeddingModel() string
}
