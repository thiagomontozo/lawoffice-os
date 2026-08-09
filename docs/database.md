# Database

Versioned SQL migrations run in lexical order and each applied filename is recorded in `schema_migrations`. PostgreSQL is the source of truth.

## Isolation and integrity

Business tables include `firm_id`; composite uniqueness on `(id, firm_id)` enables foreign keys that prove related records belong to the same firm. Unique constraints cover firm slug, user e-mail per firm, Matter internal number, document version number and other natural boundaries. Check constraints protect enum-like states and nonnegative integer money.

## Indexes

Indexes prioritize firm/status/time access paths: Matter status/priority/updated, legal case/internal number, client names, documents by Matter, deadlines/tasks/hearings by due time, timeline ordering, notifications and audit chronology. More indexes should be added only after measured query analysis.

## Records and retention

Matter, Client, User and Document metadata use soft-delete/disable semantics. Archive has its own closure record. Document bytes never enter a PostgreSQL BYTEA; version rows store object keys and checksums. Financial amounts are signed `bigint` cents with domain checks rather than floating point. Audit metadata is JSONB only for bounded, non-secret context—not as a substitute for core relational modeling.
