package ocr

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
)

var ErrUnsupported = errors.New("document type is not supported by OCR provider")

type Page struct {
	Number     int      `json:"number"`
	Text       string   `json:"text"`
	Confidence *float64 `json:"confidence,omitempty"`
}

type Result struct {
	Provider string `json:"provider"`
	Language string `json:"language"`
	Pages    []Page `json:"pages"`
}

type Document struct {
	Reader   io.Reader
	MimeType string
	FileName string
}

type Provider interface {
	Extract(context.Context, Document) (Result, error)
	Name() string
}

type Builtin struct{ MaxBytes int64 }

func (b Builtin) Name() string { return "builtin-text" }

func (b Builtin) Extract(ctx context.Context, document Document) (Result, error) {
	limit := b.MaxBytes
	if limit < 1 {
		limit = 25 << 20
	}
	data, err := io.ReadAll(io.LimitReader(contextReader{ctx: ctx, reader: document.Reader}, limit+1))
	if err != nil {
		return Result{}, err
	}
	if int64(len(data)) > limit {
		return Result{}, errors.New("OCR input exceeds configured limit")
	}
	var pages []Page
	switch document.MimeType {
	case "text/plain", "text/plain; charset=utf-8":
		pages = []Page{{Number: 1, Text: strings.TrimSpace(string(data))}}
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		text, extractionErr := extractDOCX(data)
		if extractionErr != nil {
			return Result{}, extractionErr
		}
		pages = []Page{{Number: 1, Text: text}}
	case "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
		pages, err = extractXLSX(data)
		if err != nil {
			return Result{}, err
		}
	default:
		return Result{}, ErrUnsupported
	}
	return Result{Provider: b.Name(), Language: "und", Pages: compactPages(pages)}, nil
}

func extractDOCX(data []byte) (string, error) {
	files, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("open DOCX: %w", err)
	}
	for _, file := range files.File {
		if filepath.ToSlash(file.Name) != "word/document.xml" {
			continue
		}
		reader, openErr := file.Open()
		if openErr != nil {
			return "", openErr
		}
		text, parseErr := xmlText(reader, true)
		closeErr := reader.Close()
		if parseErr != nil {
			return "", parseErr
		}
		if closeErr != nil {
			return "", closeErr
		}
		return text, nil
	}
	return "", errors.New("DOCX document.xml is missing")
}

func extractXLSX(data []byte) ([]Page, error) {
	files, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("open XLSX: %w", err)
	}
	worksheets := make([]*zip.File, 0)
	for _, file := range files.File {
		name := filepath.ToSlash(file.Name)
		if strings.HasPrefix(name, "xl/worksheets/sheet") && strings.HasSuffix(name, ".xml") {
			worksheets = append(worksheets, file)
		}
	}
	sort.Slice(worksheets, func(i, j int) bool { return worksheets[i].Name < worksheets[j].Name })
	pages := make([]Page, 0, len(worksheets))
	for index, file := range worksheets {
		reader, openErr := file.Open()
		if openErr != nil {
			return nil, openErr
		}
		text, parseErr := xmlText(reader, false)
		closeErr := reader.Close()
		if parseErr != nil {
			return nil, parseErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		pages = append(pages, Page{Number: index + 1, Text: text})
	}
	if len(pages) == 0 {
		return nil, errors.New("XLSX has no worksheets")
	}
	return pages, nil
}

func xmlText(reader io.Reader, paragraphBreaks bool) (string, error) {
	decoder := xml.NewDecoder(reader)
	var builder strings.Builder
	insideText := false
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", err
		}
		switch value := token.(type) {
		case xml.StartElement:
			insideText = value.Name.Local == "t" || value.Name.Local == "v"
		case xml.CharData:
			if insideText {
				builder.Write(value)
				builder.WriteByte(' ')
			}
		case xml.EndElement:
			if value.Name.Local == "t" || value.Name.Local == "v" {
				insideText = false
			}
			if paragraphBreaks && value.Name.Local == "p" {
				builder.WriteByte('\n')
			}
		}
	}
	return strings.TrimSpace(builder.String()), nil
}

func compactPages(pages []Page) []Page {
	result := make([]Page, 0, len(pages))
	for _, page := range pages {
		page.Text = strings.TrimSpace(page.Text)
		if page.Text != "" {
			result = append(result, page)
		}
	}
	return result
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (reader contextReader) Read(buffer []byte) (int, error) {
	select {
	case <-reader.ctx.Done():
		return 0, reader.ctx.Err()
	default:
		return reader.reader.Read(buffer)
	}
}
