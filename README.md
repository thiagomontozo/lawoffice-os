# LawOffice OS

> White-label legal practice management platform built with Go, React and TypeScript.

**Current status: Experimental.** LawOffice OS is a portfolio-grade systems and product engineering project. The repository now has initial automated tests and CI, but it is not production-ready and still requires deployment-specific security and operational validation.

## Overview

LawOffice OS is a customizable legal operations workspace for modern law firms. Instead of treating a lawsuit as an isolated record, it makes the broader **Matter** the operational center for clients, parties, files, deadlines, tasks, hearings, workflow, finance, history, access control and audit.

## Why LawOffice OS?

Legal work rarely fits into a single “process” table. A practice handles litigation, administrative proceedings, consultations, contracts, arbitration, advisory work and internal projects. Each has different confidentiality, workflow and closure requirements. LawOffice OS models those differences while retaining a coherent workspace.

## Product Vision

LawOffice OS is a **Legal Practice Operating System**: one modular product that a firm can shape around its visual identity, areas, Matter types, roles, custom fields, workflows, templates and client-facing experience.

## Key Differentiators

- **Brand Studio:** title, firm name, colors, logos, favicon, login background and client portal language.
- **White-label client portal:** controlled Matter, event and document sharing under the firm brand.
- **Matter-centric architecture:** judicial and non-judicial legal work share one adaptable model.
- **Legal Timeline:** chronological, attributable and optionally client-visible events.
- **Workflow Builder:** ordered stages, checklists, default responsibility and declarative entry tasks.
- **Conflict Check:** broad possible-match search with explicit human-decision boundaries.
- **Matter-level security:** normal, team-only, partners-only and restricted visibility with explicit grants.
- **Document versioning:** immutable file versions, checksums and historical download.
- **Structured archive workflow:** open-work warnings, closure metadata, search and controlled reopen.
- **Searchable Legal Archive:** archived work remains authorized, auditable and discoverable.
- **Command Center and Meu Dia:** action-oriented operational awareness.
- **Deep audit trail:** security and domain changes recorded without sensitive content.

## Architecture

The V0.1 is a modular monolith: a Go API owns business rules, PostgreSQL is the source of truth, filesystem object storage keeps binary files out of the database, and a React client presents internal and portal experiences.

```mermaid
flowchart LR
    React["React + TypeScript"] --> API["Go API"]
    Portal["White-label Client Portal"] --> API
    API --> PG[(PostgreSQL)]
    API --> Storage[(Object Storage)]
    API --> Scheduler["Managed Scheduler"]
    API --> SSE["SSE Hub + PostgreSQL NOTIFY"]
```

```mermaid
flowchart TD
    Matter --> Timeline
    Matter --> Documents
    Matter --> Tasks
    Matter --> Deadlines
    Matter --> Parties
    Matter --> Workflow
    Matter --> Finance
    Matter --> Audit
```

See [architecture](docs/architecture.md) and [domain model](docs/domain-model.md).

## Technology Stack

- Backend: Go, Chi, pgx, `slog`, `context.Context`
- Frontend: React, TypeScript strict mode, Vite, React Router, Tailwind CSS
- Database: PostgreSQL with versioned SQL migrations
- Storage: local filesystem behind an `ObjectStorage` interface
- Realtime: Server-Sent Events
- Deployment assets: multi-stage Dockerfiles and Docker Compose

## White-Label Brand Studio

Brand Studio persists the system title, displayed firm name, primary/secondary/accent colors, sidebar style, radius style, support details, portal title and welcome message. PNG, JPEG and WEBP assets can be uploaded for light/dark logos, favicon and login background. The frontend applies colors through `--brand-primary`, `--brand-secondary` and `--brand-accent` CSS variables. SVG is deliberately excluded in V0.1.

## Firm Isolation

Every relevant record carries `firm_id`. Repository queries scope by the authenticated firm and cross-entity foreign keys frequently use `(id, firm_id)` to prevent cross-firm references. IDs provided by the browser are never sufficient authorization by themselves.

## User Management

The initial setup creates an Owner. Administrators can add users, attach roles, disable/reactivate accounts and trigger session revocation. Passwords are bcrypt hashes and authenticated sessions use HMAC-hashed random tokens stored server-side.

## Roles and Permissions

Roles are data, not hardcoded application branches. A role can receive granular permissions for firm, branding, user, client, Matter, document, deadline, task, workflow, finance, audit, archive and portal operations.

## Matter-Level Security

Role permission answers “may this user perform this class of action?” Matter access answers “may this user see or modify this specific Matter?” Restricted Matters require an explicit user or role grant. Partner-only records also verify qualifying role membership on the backend.

## Matter Management

Supported Matter types include judicial and administrative processes, consultations, contracts, advisory work, arbitration, extrajudicial work and internal legal projects. Judicial fields live in an optional extension rather than burdening every Matter. Legal areas, types, tags, templates and moderate custom fields are firm-owned.

## Legal Timeline

Matter events record actor, timestamp, type, summary and related resource. Public portal visibility is explicit per event; internal events never become public merely because a client has Matter access.

## Document Management

Metadata belongs to PostgreSQL; content belongs to object storage. Uploads are streamed through size enforcement and SHA-256 calculation, storage keys are firm-prefixed and generated internally, filenames are never paths, and downloads repeat firm/Matter authorization. Each version records checksum, creator and notes; downloads are audited.

## Document Versioning

Uploading a new version never silently overwrites the prior object. The current pointer advances transactionally while older versions remain downloadable according to the same authorization boundary. Deletion is a reversible metadata action; physical purge is a separate retention decision that must account for legal hold.

## Deadlines and Tasks

Deadlines support open/completed/cancelled states and operational urgency windows. Tasks may belong to a Matter or the firm, support assignee, priority and due date, and can later be created from workflow entry rules. V0.1 does **not** calculate procedural deadlines automatically.

## Calendar and Hearings

The calendar consolidates hearing, deadline, task and custom-event concepts with day, week, month and agenda views. External calendar providers are roadmap work.

## Workflow Builder

Workflows define ordered stages, colors, descriptions, checklists, default roles and declarative tasks created upon stage entry. This intentionally avoids a general scripting engine or full BPMN complexity.

## Matter Templates

Templates can preselect Matter type, legal area, workflow, folders, tasks, tags and custom-field defaults, allowing repeatable firm-specific intake.

## Conflict Check

Conflict Check searches active and archived clients, contacts, opposing or related parties and Matter titles. Its result says **Possible conflict**, never a legal conclusion. False positives and name collisions require professional review.

## Legal Archive

Closing computes warnings for open tasks, deadlines, future hearings and financial items. Authorized users may proceed while preserving warning counts and closure metadata. Archived Matters stay searchable and can be reopened with a reason and audit event.

## Financial Tracking

Matter finance tracks fees, expenses, court costs, reimbursements, success fees and payments in integer cents. Summaries are operational and must not be treated as official accounting.

## Command Center

The dashboard prioritizes critical/today deadlines, hearings, assigned tasks, recent documents, Matter activity, high-priority Matters and archive readiness. “Meu Dia” narrows work to the authenticated user.

## Global Search

The persistent search and Ctrl/Cmd+K command palette rank and group authorized results for Matter titles, internal/case numbers, clients, contacts, documents, parties and tags. PostgreSQL trigram indexes and relevance scoring remain behind a replaceable search boundary without weakening firm or Matter permissions.

## Client Portal

Portal users have separate credentials and sessions. An administrator creates a 72-hour, single-use invitation and the client defines their own password. A `PortalAccess` grant links that identity to selected Matters and defines shareable summary, timeline and appointments. Documents and events need an explicit `clientVisible` flag, downloads repeat authorization checks, revocation invalidates existing sessions, and the portal uses firm branding.

## Audit Trail

Important authentication, administration, Matter, document, branding, archive and finance events record firm, actor, action, resource, request context and sanitized metadata. Passwords, tokens, authorization headers and file contents are excluded.

## Security Model

Security is layered: authentication → firm isolation → RBAC → Matter authorization → document authorization → portal authorization. Strict HttpOnly cookies, origin checks, sensitive-endpoint rate limits and browser security headers harden the HTTP boundary. Uploads are size/MIME limited and stored under generated keys. Sensitive records use soft deletion. CodeQL, Go vulnerability analysis, npm audit and Dependabot provide continuous repository checks. See [security model](docs/security-model.md).

## Privacy-Oriented Design

The architecture includes privacy-oriented controls, but legal compliance depends on deployment, configuration and organizational processes. Access control, audit, data isolation, session revocation, minimal logging and export foundations help support responsible operation; they do not automatically establish LGPD compliance.

## Getting Started

### Requirements

- Go 1.24+
- Node.js 24+
- PostgreSQL 17+
- Docker Compose is optional

### First access and office registration

1. Copy `.env.example` to `.env` and replace `SESSION_SECRET`. In production it must contain at least 32 characters and must not be `change-me`.
2. Create the PostgreSQL database and start the backend from `backend/`. Migrations run on API startup.
3. Start the frontend from `frontend/`.
4. Open `http://localhost:5173/setup` (or `http://localhost:3000/setup` with Compose).
5. Complete the setup wizard: office details → visual identity → administrator → initial legal structure → finish.
6. The first administrator is assigned the **Owner** role. The wizard also creates default legal areas, Matter types and a starter workflow.
7. Future access uses `/login` with three values: **firm slug**, administrator e-mail and password.

After login, a practical first-use path is:

1. Open **Brand Studio** and upload the firm logo/favicon.
2. Review **Roles** and create the internal users.
3. Run **Conflict Check** for the prospective client.
4. Register the client and create a Matter.
5. Add parties, deadlines, tasks and document versions from Matter Detail.
6. Configure a workflow and, when appropriate, open **Portal do Cliente**, select a client and the Matters to share, then send the one-time invitation link through a trusted channel.

No real legal data should be used until deployment security, backups and validation are reviewed.

## Configuration

| Variable | Purpose | Default/example |
|---|---|---|
| `APP_ENV` | `development` or `production` | `development` |
| `API_PORT` | API listener | `8080` |
| `DATABASE_URL` | PostgreSQL connection | see `.env.example` |
| `WEB_ORIGIN` | exact allowed browser origin | `http://localhost:5173` |
| `SESSION_SECRET` | session HMAC key | required |
| `STORAGE_PATH` | local object root | `./data/storage` |
| `MAX_UPLOAD_MB` | upload limit, 1–100 | `25` |
| `LOG_LEVEL` | structured log level | `info` |
| `DEFAULT_LOCALE` | firm setup fallback | `pt-BR` |
| `DEFAULT_TIMEZONE` | firm setup fallback | `America/Sao_Paulo` |

## API

Health endpoints live at `/healthz` and `/readyz`. Business endpoints are versioned below `/api/v1`; cookie-based calls require credentials. See [API reference](docs/api.md).

## Database

The initial migration creates the multi-firm relational model, constraints, indexes and permission catalog. Money is integer cents; files are metadata only; soft-delete fields protect legal records. See [database notes](docs/database.md).

## Object Storage

`ObjectStorage` exposes `Put`, `Open`, `Delete` and `Health`. The local adapter enforces root confinement and atomic placement. S3/MinIO support belongs to V0.2.

## Docker

The repository includes multi-stage backend/frontend Dockerfiles and a Compose topology for PostgreSQL, API, web and persistent document storage. CI builds both images. A local Compose validation requires Docker Desktop or another compatible daemon.

## Testing and CI

The first reliability milestone adds:

- Go unit tests for password/session handling, configuration and local storage safety;
- PostgreSQL integration coverage for firm isolation, restricted Matter access and document authorization;
- Vitest coverage for the typed API client and standard error envelope;
- frontend type checking and production build;
- `go vet`, Go build and race-enabled tests;
- remote PostgreSQL and container builds in GitHub Actions.
- CodeQL extended analysis, Go vulnerability scanning, npm runtime dependency audit and Dependabot updates.

Run the local checks with:

```bash
cd backend
go vet ./...
go test ./...
go build -o ../.tools/lawoffice-api ./cmd/api

cd ../frontend
npm ci --ignore-scripts
npm run format:check
npm test
npm run build
```

Repository integration tests use `TEST_DATABASE_URL`. Without it they skip safely; CI always supplies an isolated PostgreSQL service.

## Project Structure

```text
backend/             Go modular monolith, migration runner and SQL migrations
frontend/            React/TypeScript workspace and white-label portal
docs/                Architecture, domain, security and decisions
Dockerfile.backend   Multi-stage API image
Dockerfile.frontend  Multi-stage static web image
compose.yml          PostgreSQL + API + web + persistent storage
```

## Design Decisions

Twelve ADRs in [`docs/decisions`](docs/decisions) cover the stack, modular monolith, Matter domain, firm isolation, layered authorization, object storage, versioning, white label, SSE and structured archive.

## Limitations

- The automated suite is intentionally focused on critical foundations and does not yet cover every handler or React interaction.
- PostgreSQL distributes live SSE events across replicas, but the bounded replay window remains instance-local rather than a durable event log.
- Local object storage assumes a single shared filesystem and has no virus scanner or preview service.
- Search uses ranked PostgreSQL matching and indexed trigrams, but does not yet provide semantic search.
- Portal invitations must be delivered by the firm; password recovery and per-field sharing controls are not yet available.
- The built-in abuse limiter is instance-local; production needs a distributed edge rate limiter and monitoring.
- Scheduler notification rules are basic and do not calculate legal procedural deadlines.
- No e-mail/SMS, calendar provider, court connector, OCR, AI, SSO or SaaS billing.

## Roadmap

### v0.2

- broader handler, browser and accessibility test coverage
- e-mail notifications
- richer custom fields
- PDF report generation
- improved global search
- advanced workflow automation
- document preview
- S3/MinIO adapter

### v0.3

- Google Calendar / Microsoft Calendar integration
- legal publication ingestion connectors
- tribunal integrations
- advanced deadline automation
- approval flows and document templates

### v0.4

- private legal AI workspace
- Matter-scoped RAG and permission-aware retrieval
- document summarization and chronology generation
- semantic legal search

### v0.5

- multi-firm SaaS mode
- billing foundations
- advanced analytics
- enterprise SSO

AI is deliberately not implemented. A future Matter AI Workspace must never retrieve a document the requesting user cannot access.

## Contributing

Open an issue before large changes. Keep the modular monolith, firm-scoped queries, backend authorization and documentation aligned. Tests and CI are the first recommended contribution milestone.

## License

MIT © 2026 Thiago Montozo. See [LICENSE](LICENSE).
