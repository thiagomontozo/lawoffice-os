# Architecture

Rendered views of the main application surfaces are available in [Frontend interface](interface.md).

## Modular monolith

LawOffice OS is one deployable Go API organized by domain boundaries rather than a network of microservices. Identity, branding, clients, Matters, documents, deadlines, workflow, archive, search, finance, portal and audit share one transaction boundary and PostgreSQL source of truth. This keeps V0.1 operationally understandable while preserving module-level seams.

```mermaid
flowchart LR
  Web["React workspace"] --> HTTP["Chi HTTP boundary"]
  Portal["Client portal"] --> HTTP
  HTTP --> Services["Domain services"]
  Services --> Repositories["Firm-scoped repositories"]
  Repositories --> PG[(PostgreSQL)]
  Services --> Objects["ObjectStorage"]
  Scheduler --> Repositories
  Services --> SSE["SSE Hub"]
```

## Backend

Handlers decode bounded input and translate errors. Services enforce cross-resource rules such as Matter authorization and upload policy. Repositories own SQL and always receive `firmID`. Authentication middleware resolves a server-side session into a user plus roles/permissions. Domain objects remain independent of HTTP presentation.

The scheduler is one cancellation-aware goroutine for the entire process, not one goroutine per deadline. Each cycle takes a PostgreSQL transaction-scoped advisory lock, so only one API replica scans near-term deadlines/tasks while idempotent inserts provide a second safety layer. SIGINT/SIGTERM cancels the scheduler, closes SSE, drains HTTP and then closes PostgreSQL.

## Frontend

React Router separates public login/setup, protected `/app` routes and `/portal`. An authentication context loads the user, firm and branding. CSS variables apply the firm palette. The app layout owns desktop navigation, mobile navigation, global search, quick create and the Ctrl/Cmd+K command palette. Feature pages handle loading, empty, error and populated states.

## Database and storage

PostgreSQL holds relational state and authorization relationships. Composite `(id, firm_id)` constraints prevent cross-firm relationships. Binary documents are stored through `ObjectStorage`; only immutable version metadata and a current-version pointer live in PostgreSQL.

## SSE and failure modes

SSE delivers notification, timeline and task events with event IDs, heartbeats and a bounded in-memory replay window. PostgreSQL `LISTEN/NOTIFY` fans live events across API replicas without adding a broker; when the listener cannot start, the API logs degradation and falls back to local delivery. Slow consumers may miss events, and replay remains instance-local, so PostgreSQL is authoritative and clients refetch after gaps or reconnects routed to another replica. A database outage fails readiness and data operations. A storage outage fails readiness and document operations but does not redefine PostgreSQL as file storage.

## Future scale

The next scale boundaries are S3/MinIO for shared objects, a durable event log if cross-instance replay becomes necessary, a dedicated search implementation behind `SearchService`, and a separate job worker if scheduler volume grows. These can evolve without splitting every domain into a service.
