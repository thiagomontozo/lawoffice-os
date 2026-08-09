# Structured Matter Archive

Archive is a legal-work lifecycle, not a boolean toggle.

1. Request closure and compute open tasks, open deadlines, future hearings and financial pending state.
2. Present warnings. Authorized users may continue explicitly; warnings are persisted.
3. Require reason, outcome where relevant, closing date and summary.
4. Store `MatterClosure`, change Matter status and append Matter/Audit events.
5. Keep the Matter searchable with year, area, client, responsible, court, outcome, tag and closure filters.

No Matter or document history is deleted during archive. Reopen requires `matter.reopen`, a reason, actor and timestamp. The closure record remains part of history. Permanent deletion is a separate, rare administrative/retention concern.
