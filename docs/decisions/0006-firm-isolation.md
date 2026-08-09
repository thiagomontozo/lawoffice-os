# Enforce firm isolation everywhere

**Status:** Accepted

## Context
Even single-firm deployments handle highly confidential records and future multi-tenancy is anticipated.

## Decision
Carry authenticated `firm_id` through services, scope every query and use composite cross-firm constraints.

## Consequences
Guessing an ID cannot cross the tenancy boundary. Repository review must treat missing firm scope as a security defect.
