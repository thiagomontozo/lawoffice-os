# API

All errors use `{ "error": { "code", "message", "requestId" } }`. Internal endpoints use an HttpOnly session cookie and permission middleware. JSON bodies are bounded and reject unknown fields.

## Public and authentication

- `GET /healthz`, `GET /readyz`
- `POST /api/v1/setup`
- `POST /api/v1/auth/login`, `POST /logout`, `GET /me`, `POST /change-password`
- `GET /api/v1/public/branding/:slug`

## Administration

- `/api/v1/branding` and `/branding/assets/:kind`
- `/api/v1/users`, `/users/:id/active`
- `/api/v1/roles`
- `/api/v1/audit`

## Legal operations

- `/api/v1/clients`
- `/api/v1/matters`, `/matters/:id`
- `/api/v1/matters/:id/archive`, `/reopen`
- `/api/v1/documents`, `/documents/:id/versions`, `/download`
- `/api/v1/deadlines`, `/tasks`, `/calendar`, `/workflows`
- `POST /api/v1/conflicts/check`
- `GET /api/v1/search`, `/dashboard`, `/notifications`, `/stream`

Lists use `page` and `pageSize` (maximum 100) and feature-specific filters such as `q`, `status`, `priority`, `type`, `matterId`, `archived` and `mine`. Uploads use multipart form data.

## Portal

- `POST /api/v1/portal/login`, `/logout`
- `GET /api/v1/portal/matters`, `/portal/matters/:id`

Portal calls use their own cookie and never accept an internal user session as authorization.
