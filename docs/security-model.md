# Security Model

## Layers

1. **Authentication:** random HttpOnly cookie token; only its HMAC-SHA-256 hash is stored. Passwords use bcrypt.
2. **Firm isolation:** authenticated `firm_id` scopes queries and relationships.
3. **RBAC:** data-driven roles grant action permissions.
4. **Matter access:** confidentiality and explicit user/role grants scope individual Matters.
5. **Document authorization:** firm, document, Matter and requested version are checked before storage opens.
6. **Portal authorization:** separate sessions plus explicit `PortalAccess`; each event/document also needs client visibility.

Disabling a user revokes their active sessions. Logout removes the server-side session. Production cookies use `Secure`, `HttpOnly` and `SameSite=Lax`; deployments should terminate TLS and rotate secrets.

## Uploads

Requests are bounded before reading. Allowed V0.1 types are PDF, DOCX, XLSX, PNG, JPEG, WEBP and plain text; branding accepts only PNG/JPEG/WEBP. Content sniffing and extension agreement reduce spoofing. UUID-based storage keys avoid trusting original names. The local adapter cleans paths, rejects absolute/traversal input and verifies root confinement.

Virus scanning, content-disarm, encrypted object storage and retention automation are deployment or future-version responsibilities. Uploaded files are never executed.

## Audit, deletion and logging

Audit metadata excludes passwords, cookies, authorization values and content. Legal records use soft deletion where appropriate; document objects are retained until an explicit retention process. Structured logs include request IDs but avoid client documents and arbitrary field values.

## Privacy

The architecture includes privacy-oriented controls, but legal compliance depends on deployment, configuration and organizational processes. Backups, retention, access reviews, incident response and data-subject procedures remain organizational responsibilities.
