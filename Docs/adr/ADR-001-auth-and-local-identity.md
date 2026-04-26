# ADR-001: Clerk JWT with Local User Identity

## Status
Accepted

## Context
APIs need external authentication plus stable local user IDs for data ownership and authorization checks.

## Decision
Use Clerk JWT for session auth and map `claims.Subject` to local user records (`users.clerk_id`).

## Consequences
- Clear separation of external auth provider and internal authorization.
- Requires reliable subject->local ID lookup and caching.
