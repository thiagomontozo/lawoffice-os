# OCR and document text extraction

Document extraction is an auxiliary, asynchronous capability. The immutable original and its checksum remain authoritative; extracted text is searchable support material and may contain recognition errors.

## Lifecycle

Every `document_versions` insert automatically creates one `document_extractions` row. Workers claim pending rows with `FOR UPDATE SKIP LOCKED`, use a ten-minute lease, retry transient failures and write pages in one transaction. A new document version receives a separate extraction. Reprocessing resets only the current version's extraction.

Statuses are `pending`, `processing`, `succeeded`, `failed` and `unsupported`. Errors exposed through the API are stable codes, never provider response bodies or file contents.

## Providers

`OCR_MODE=off` stores pending work without running a worker. Enabling OCR later processes the backlog.

`OCR_MODE=builtin` extracts UTF-8 text, DOCX XML and XLSX worksheet text without an external process. It deliberately reports scanned PDF and image formats as unsupported because pretending that text parsing is OCR would be misleading.

`OCR_MODE=http` sends a bounded multipart request to `OCR_ENDPOINT` with fields `file`, `mimeType` and `language`. The service must return:

```json
{
  "provider": "provider-name",
  "language": "pt-BR",
  "pages": [
    { "number": 1, "text": "Extracted text", "confidence": 0.98 }
  ]
}
```

Page numbers must be positive and unique, confidence must be between zero and one, and configured page/character limits are enforced before persistence. Production endpoints require HTTPS and a sufficiently strong bearer token.

## Security

The worker receives only internal storage keys from PostgreSQL. Users cannot provide a path. The API checks the existing document and Matter authorization before returning text or accepting reprocessing. Provider tokens are environment variables and are never persisted or logged. Deployments must contractually review provider retention, training, residency and incident-response terms before sending legal documents outside their infrastructure.

Extracted text is untrusted input. The RAG/AI layer treats it as evidence, not instructions, and repeats firm, Matter and document authorization before retrieval. Successful extraction rebuilds version/page-aware chunks in the same transaction. See [Matter AI Workspace](ai-workspace.md).
