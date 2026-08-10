# Client Portal

The portal is a separate layout and identity boundary. An authorized team member creates a 72-hour, single-use invitation for a client and selected Matters. The full invitation token is returned only at creation time. Its HMAC hash is stored in PostgreSQL, and the client defines a password on acceptance. `PortalUser` credentials are bcrypt hashes and sessions use a distinct HttpOnly cookie path.

`PortalAccess` links the identity to selected Matters and controls summary, timeline and appointment visibility. The portal never returns a complete internal Matter. Responses reduce metadata to the allowed summary and include only `clientVisible` events and documents. Downloads repeat portal identity, firm, Matter and visibility checks. Internal notes, confidential fields, Matter access rules, finance and audit are not portal data. A client cannot select another Matter or document merely by changing its ID.

Brand Studio supplies the office logo, colors, portal title and welcome text. The administration screen lists invited accounts, last login, Matter count and active state. Revocation immediately deletes existing portal sessions. Re-invitation is allowed only while an invitation has not been accepted; an active credential cannot be silently replaced.

When SMTP is enabled, invitation delivery runs through a PostgreSQL job queue. Recipient, subject and one-time URL are encrypted with AES-GCM before insertion. Multiple replicas claim work with `SKIP LOCKED`, retry with bounded backoff and remove ciphertext after completion or terminal failure. The authorized administrator still receives the manual link as a controlled fallback.

Delivery is at least once: a process failure after SMTP accepts a message but before the completion update can cause a duplicate. One-time tokens make duplicate links equivalent. Drain or expire pending jobs before rotating `JOB_ENCRYPTION_SECRET`; old ciphertext cannot be decrypted with a new key.

Password recovery accepts firm slug and e-mail but always returns the same accepted response for known and unknown accounts. A known active account receives a one-hour, single-use link through the same durable queue. PostgreSQL stores only the token hash. A successful reset revokes every portal session and records a portal audit event.

MFA, document preview, bounce/webhook processing, per-field sharing controls and user-managed notification preferences remain follow-up work. When SMTP is disabled, recovery cannot deliver a link and the firm must manage access through a fresh invitation.
