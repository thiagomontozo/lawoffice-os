# Abstract object storage

**Status:** Accepted

## Context
Legal files are large, immutable by version and unsuitable for PostgreSQL BYTEA.

## Decision
Define `ObjectStorage` and initially provide a root-confined local filesystem adapter.

## Consequences
Development remains simple and S3/MinIO can follow. Multi-instance use requires shared storage and operational retention controls.
