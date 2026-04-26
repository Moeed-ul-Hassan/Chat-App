# Swagger Redesign Guide

## Objectives
- Consistent tags by bounded context.
- Reusable success/error schemas.
- Route-level security and role requirements documented.
- Request and response examples for every endpoint.

## Conventions
- Error envelope: `code`, `message`, `request_id`, `details`.
- Use explicit `401`, `403`, `404`, `409`, `500` annotations where relevant.
- Group tags: `auth`, `users`, `chat`, `rooms`, `stories`, `admin`, `health`.

## Workflow
1. Update handler annotations.
2. Run `swag init -g cmd/server/main.go -o docs`.
3. Commit regenerated `docs.go`, `swagger.json`, and `swagger.yaml`.
4. CI verifies no Swagger drift.
