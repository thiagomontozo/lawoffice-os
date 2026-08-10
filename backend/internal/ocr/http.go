package ocr

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
)

type HTTPProvider struct {
	Endpoint, Token, Language string
	Client                    *http.Client
	MaxBytes                  int64
	MaxPages, MaxCharacters   int
}

func (provider *HTTPProvider) Name() string { return "remote-http" }

func (provider *HTTPProvider) Extract(ctx context.Context, document Document) (Result, error) {
	if provider.Client == nil {
		return Result{}, errors.New("OCR HTTP client is required")
	}
	if _, err := url.ParseRequestURI(provider.Endpoint); err != nil {
		return Result{}, errors.New("invalid OCR endpoint")
	}
	data, err := io.ReadAll(io.LimitReader(contextReader{ctx: ctx, reader: document.Reader}, provider.MaxBytes+1))
	if err != nil {
		return Result{}, err
	}
	if int64(len(data)) > provider.MaxBytes {
		return Result{}, errors.New("OCR input exceeds configured limit")
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	file, err := writer.CreateFormFile("file", document.FileName)
	if err != nil {
		return Result{}, err
	}
	if _, err = file.Write(data); err != nil {
		return Result{}, err
	}
	_ = writer.WriteField("mimeType", document.MimeType)
	_ = writer.WriteField("language", provider.Language)
	if err = writer.Close(); err != nil {
		return Result{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, provider.Endpoint, &body)
	if err != nil {
		return Result{}, err
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("Accept", "application/json")
	if provider.Token != "" {
		request.Header.Set("Authorization", "Bearer "+provider.Token)
	}
	response, err := provider.Client.Do(request)
	if err != nil {
		return Result{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return Result{}, fmt.Errorf("OCR provider returned HTTP %d", response.StatusCode)
	}
	var result Result
	decoder := json.NewDecoder(io.LimitReader(response.Body, 16<<20))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&result); err != nil {
		return Result{}, fmt.Errorf("decode OCR response: %w", err)
	}
	result.Provider = strings.TrimSpace(result.Provider)
	if result.Provider == "" {
		result.Provider = provider.Name()
	}
	if result.Language == "" {
		result.Language = provider.Language
	}
	if err = validateResult(result, provider.MaxPages, provider.MaxCharacters); err != nil {
		return Result{}, err
	}
	result.Pages = compactPages(result.Pages)
	return result, nil
}

func validateResult(result Result, maxPages, maxCharacters int) error {
	if len(result.Pages) > maxPages {
		return errors.New("OCR response exceeds page limit")
	}
	seen := make(map[int]struct{}, len(result.Pages))
	characters := 0
	for _, page := range result.Pages {
		if page.Number < 1 {
			return errors.New("OCR response contains invalid page number")
		}
		if _, exists := seen[page.Number]; exists {
			return errors.New("OCR response contains duplicate page number")
		}
		seen[page.Number] = struct{}{}
		characters += len([]rune(page.Text))
		if characters > maxCharacters {
			return errors.New("OCR response exceeds character limit")
		}
		if page.Confidence != nil && (*page.Confidence < 0 || *page.Confidence > 1) {
			return errors.New("OCR response contains invalid confidence")
		}
	}
	return nil
}
