# Database

Versioned SQL migrations run in lexical order and each applied filename is recorded in `schema_migrations`. PostgreSQL is the source of truth.

## Isolation and integrity

Business tables include `firm_id`; composite uniqueness on `(id, firm_id)` enables foreign keys that prove related records belong to the same firm. Unique constraints cover firm slug, user e-mail per firm, Matter internal number, document version number and other natural boundaries. Check constraints protect enum-like states and nonnegative integer money.

## Indexes

Indexes prioritize firm/status/time access paths: Matter status/priority/updated, legal case/internal number, documents by Matter, deadlines/tasks/hearings by due time, timeline ordering, notifications and audit chronology. The second migration enables the trusted `pg_trgm` extension and adds partial GIN indexes for active Matter, client, contact, document, party and tag names. Global search combines substring matching with trigram relevance ranking while retaining firm and Matter authorization predicates. The fourth migration adds a seven-day `realtime_events` log indexed by firm and monotonic ID; the coordinated scheduler removes expired rows in bounded batches.

The fifth migration adds encrypted outbound jobs and hashed portal password-reset tokens. Job payloads are opaque `bytea`; lease and retry columns support delivery without exposing recipient or message content. Reset rows are single-use and expire automatically. The sixth migration adds per-user notification preferences, an e-mail queued marker and a partial unique deduplication index for outbound jobs.

The seventh migration adds one `document_extractions` record per immutable version and page-level text in `document_extraction_pages`. A generated Portuguese `tsvector` and GIN index support full-text retrieval. Status, lease, attempt and sanitized error columns provide observable asynchronous processing without placing document contents in logs.

The eighth migration adds version/page-aware `document_chunks` with generated full-text vectors, content hashes and optional JSON embeddings. Embedding status, leases and attempts form a durable worker queue. `ai_responses` and `ai_feedback` store only operational response identifiers, model/retrieval metadata and user ratings; prompts and generated answers are deliberately absent.

## Records and retention

Matter, Client, User and Document metadata use soft-delete/disable semantics. Archive has its own closure record. Document bytes never enter a PostgreSQL BYTEA; version rows store object keys and checksums. Financial amounts are signed `bigint` cents with domain checks rather than floating point. Audit metadata is JSONB only for bounded, non-secret context—not as a substitute for core relational modeling.
