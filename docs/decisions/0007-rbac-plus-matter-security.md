# Combine RBAC with Matter security

**Status:** Accepted

## Context
A lawyer may generally read Matters yet be excluded from a sensitive engagement.

## Decision
Use data-driven action permissions plus Matter confidentiality and explicit user/role grants.

## Consequences
Authorization matches real firm structures. List, search, documents and nested resources must repeat the Matter boundary on the backend.
