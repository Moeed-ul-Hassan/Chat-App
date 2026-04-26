# Admin Feature

## Responsibilities
- Provide admin-only analytics and operational views.

## Key Paths
- `backend/features/admin/handler.go`
- `backend/core/middleware/auth.go` (`RequireAdmin`)

## Security Notes
- All admin endpoints require both authentication and local `is_admin=true`.

## Test Checklist
- Authenticated non-admin receives 403.
- Admin user can access all admin routes.
