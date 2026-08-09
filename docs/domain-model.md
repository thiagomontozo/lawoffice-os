# Domain Model

The firm is the tenancy boundary and Matter is the legal-work aggregate. A judicial process is optional Matter detail, not the root entity.

```mermaid
erDiagram
  FIRM ||--|| FIRM_BRANDING : owns
  FIRM ||--o{ USER : employs
  USER }o--o{ ROLE : receives
  ROLE }o--o{ PERMISSION : grants
  FIRM ||--o{ CLIENT : serves
  CLIENT ||--o{ CONTACT : has
  FIRM ||--o{ MATTER : operates
  MATTER ||--o| MATTER_LEGAL_PROCESS : extends
  MATTER ||--o{ MATTER_PARTY : involves
  MATTER ||--o{ MATTER_ACCESS : restricts
  MATTER ||--o{ DOCUMENT : contains
  DOCUMENT ||--o{ DOCUMENT_VERSION : versions
  MATTER ||--o{ DEADLINE : tracks
  MATTER ||--o{ TASK : tracks
  MATTER ||--o{ HEARING : schedules
  MATTER ||--o{ MATTER_EVENT : narrates
  MATTER ||--o| MATTER_CLOSURE : closes
  MATTER }o--o{ PORTAL_USER : shares
```

- **Firm / FirmBranding:** identity, locale and white-label presentation.
- **User / Role / Permission:** internal identity and extensible RBAC.
- **Client / Contact / MatterParty:** represented entities and related people with explicit side/role.
- **Matter / MatterLegalProcess:** universal legal work plus litigation-specific detail.
- **MatterAccess:** user/role grants at read, write or manage level.
- **CustomFieldDefinition / Value and Tag:** moderate firm customization for clients and Matters.
- **Document / DocumentVersion / Folder:** stable metadata identity, immutable binaries and Matter organization.
- **Deadline / Task / Hearing / CalendarEvent:** operational commitments.
- **WorkflowDefinition / Stage / Transition / Instance:** configurable progression and declarative entry work.
- **MatterTemplate:** repeatable default legal structure.
- **MatterEvent / Note:** attributed timeline and private/team knowledge.
- **MatterClosure:** structured outcome and warnings retained at archive time.
- **MatterFinancialEntry:** operational money in integer cents.
- **PortalUser / PortalAccess:** separate external identity and explicit disclosure.
- **AuditEvent / Notification:** accountability and user attention.
