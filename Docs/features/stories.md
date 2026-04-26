# Stories Feature

## Responsibilities
- Create ephemeral media stories.
- List active stories.
- Track viewers and restrict viewer list visibility.

## Key Paths
- `backend/features/story/handler.go`
- `backend/features/story/models.go`

## Data Rules
- `expires_at` TTL cleanup in Mongo.
- Global feed cache with invalidation on create/view.

## Test Checklist
- Owner-only viewer list access.
- Media type validation for uploads.
- Mark viewed is idempotent from client perspective.
