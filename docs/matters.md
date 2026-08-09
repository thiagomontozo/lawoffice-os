# Matters

## Matter, not Process

A Process implies litigation. A Matter covers litigation, administrative proceedings, advice, contracts, arbitration, extrajudicial work and internal legal projects. Process-only fields are stored in `MatterLegalProcess` and are optional.

## Lifecycle

`draft → active → on_hold → closing → archived`. Archiving is a structured operation: the service counts open tasks/deadlines, future hearings and pending financial work, displays warnings, records closure reason/outcome/summary and appends audit/timeline events. Reopen requires permission and a reason.

## Visibility

- `normal`: users with Matter read permission may view.
- `team_only`: responsible/creator or explicit access.
- `partners_only`: Owner, Administrator, Partner or explicit access.
- `restricted`: explicit user/role access, responsible or creator only.

The backend applies visibility to detail, lists, documents, deadlines, tasks, calendar and search. The browser never grants access.

## Customization

Firm-owned legal areas, Matter types, tags and text/textarea/number/date/boolean/select custom fields allow a moderate schema extension. Templates combine defaults for area, type, workflow, folders, tasks, tags and custom fields.

## Timeline

Every material lifecycle action produces an attributable event. Events can link to a document, task or deadline. Portal visibility is explicit and independent from internal access.
