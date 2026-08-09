# Use PostgreSQL as source of truth

**Status:** Accepted

## Context
The domain is relational, audit-heavy and benefits from constraints and transactions.

## Decision
Persist application state in PostgreSQL with versioned SQL migrations. Keep binary documents in object storage.

## Consequences
Foreign keys enforce firm-scoped relationships and transactional changes. Deployment requires managed backups and migration discipline.
