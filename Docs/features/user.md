# User Feature

## Responsibilities
- Maintain user profile and identity mapping.
- Enforce unique username (case-insensitive).
- Resolve Clerk subject to local user record.

## Key Paths
- `backend/features/user/model.go`
- `backend/features/user/handler.go`

## Data Model Notes
- `username_lower` exists for unique normalized lookup.
- `clerk_id` maps external identity to local ID.

## Test Checklist
- Username normalization behavior.
- Unique username generation collision handling.
- Batched lookup by IDs returns expected mapping.
