# Auth Feature

## Responsibilities
- Verify Clerk JWT for protected endpoints.
- Sync Clerk users into local user records.
- Enforce role-based access for admin endpoints.

## Key Paths
- `backend/core/middleware/auth.go`
- `backend/features/auth/handler.go`

## Edge Cases
- Missing/invalid bearer token.
- Clerk user exists remotely but not locally.
- Non-admin user hitting admin route.

## Test Checklist
- Unauthorized requests return 401.
- Invalid token returns 401.
- Non-admin on admin route returns 403.
