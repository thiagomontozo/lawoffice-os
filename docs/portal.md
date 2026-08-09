# Client Portal

The portal is a separate layout and identity boundary. An authorized team member creates a 72-hour, single-use invitation for a client and selected Matters. The full invitation token is returned only at creation time. Its HMAC hash is stored in PostgreSQL, and the client defines a password on acceptance. `PortalUser` credentials are bcrypt hashes and sessions use a distinct HttpOnly cookie path.

`PortalAccess` links the identity to selected Matters and controls summary, timeline and appointment visibility. The portal never returns a complete internal Matter. Responses reduce metadata to the allowed summary and include only `clientVisible` events and documents. Downloads repeat portal identity, firm, Matter and visibility checks. Internal notes, confidential fields, Matter access rules, finance and audit are not portal data. A client cannot select another Matter or document merely by changing its ID.

Brand Studio supplies the office logo, colors, portal title and welcome text. The administration screen lists invited accounts, last login, Matter count and active state. Revocation immediately deletes existing portal sessions. Re-invitation is allowed only while an invitation has not been accepted; an active credential cannot be silently replaced.

V0.1 returns the invitation link to the authorized administrator for delivery through a trusted channel. Automated e-mail delivery, password recovery, document preview and per-field sharing controls are follow-up work.
