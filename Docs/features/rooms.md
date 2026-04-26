# Rooms Feature

## Responsibilities
- Create private vault rooms with passkey protection.
- Lookup room metadata.
- Validate room join passkey and return encrypted key material.

## Key Paths
- `backend/features/room/handler.go`
- `backend/features/room/models.go`

## Edge Cases
- Invalid passkey handling.
- Unauthorized room creation.
- Brute-force attempts on join endpoint.

## Test Checklist
- Create requires auth + valid payload.
- Join rejects invalid passkey.
- Room metadata does not leak passkey hash.
