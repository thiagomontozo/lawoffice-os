# Use Go for the backend

**Status:** Accepted

## Context
The API needs explicit concurrency/lifecycle control, low operational overhead and clear dependency boundaries.

## Decision
Use idiomatic Go with Chi, pgx, `context.Context` and `slog`.

## Consequences
The service ships as a small static binary and remains easy to profile. The team owns more application structure than with a full-stack framework.
