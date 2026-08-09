# Build a modular monolith

**Status:** Accepted

## Context
Legal operations cross many domains but V0.1 does not justify distributed transactions or independent deployments.

## Decision
Keep one Go API and PostgreSQL database with explicit module and service/repository boundaries.

## Consequences
Transactions and deployment stay simple. Future extraction requires evidence from scale or ownership, not speculative microservices.
