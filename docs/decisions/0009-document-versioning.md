# Preserve document versions

**Status:** Accepted

## Context
Silently replacing a legal file destroys authorship and review history.

## Decision
Store immutable versions with sequential number, checksum, creator and notes; maintain a current pointer.

## Consequences
History is recoverable and auditable. Storage grows monotonically until a documented retention process exists.
