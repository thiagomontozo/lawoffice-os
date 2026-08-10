# Security Model

## Layers

1. **Authentication:** random HttpOnly cookie token; only its HMAC-SHA-256 hash is stored. Passwords use bcrypt.
2. **Firm isolation:** authenticated `firm_id` scopes queries and relationships.
3. **RBAC:** data-driven roles grant action permissions.
4. **Matter access:** confidentiality and explicit user/role grants scope individual Matters.
5. **Document authorization:** firm, document, Matter and requested version are checked before storage opens.
6. **Portal authorization:** separate sessions plus explicit `PortalAccess`; each event/document also needs client visibility.

Disabling a user revokes their active sessions. Logout removes the server-side session. Cookies use `HttpOnly` and `SameSite=Strict`, add `Secure` in production and store only an HMAC hash server-side. Browser mutations with an `Origin` header must match the configured `WEB_ORIGIN`; this supplements SameSite cookies against cross-site request forgery. Authentication, setup and invitation-acceptance endpoints have bounded per-instance attempt limits.

API responses set `nosniff`, deny framing, restrict referrers and browser capabilities, and use a restrictive API content security policy. GitHub security automation runs CodeQL extended queries, Go vulnerability analysis and an npm runtime dependency audit. Dependabot tracks Go, npm and GitHub Actions updates.

## Uploads

Requests are bounded before reading. Allowed V0.1 types are PDF, DOCX, XLSX, PNG, JPEG, WEBP and plain text; branding accepts only PNG/JPEG/WEBP. Content sniffing and extension agreement reduce spoofing. UUID-based storage keys avoid trusting original names. Local and S3 adapters reject unsafe keys. Required ClamAV mode scans the request-temporary file before object commit and fails closed when the scanner is unavailable.

Signature updates, content-disarm, storage-side encryption and retention automation remain deployment responsibilities. Uploaded files are never executed.

The built-in rate limiter is defense in depth for a single API instance. A production deployment should also enforce distributed rate limits and abuse controls at a trusted reverse proxy or gateway.

## Audit, deletion and logging

Audit metadata excludes passwords, cookies, authorization values and content. Legal records use soft deletion where appropriate; document objects are retained until an explicit retention process. Structured logs include request IDs but avoid client documents and arbitrary field values.

Outbound portal e-mail jobs keep recipient and one-time URLs only in AES-GCM encrypted payloads. Completed and terminally failed jobs discard ciphertext. Invitation and reset tables store token hashes, not raw tokens. Production SMTP requires STARTTLS; password recovery does not reveal account existence and successful resets revoke portal sessions.

## Privacy

The architecture includes privacy-oriented controls, but legal compliance depends on deployment, configuration and organizational processes. Backups, retention, access reviews, incident response and data-subject procedures remain organizational responsibilities.
