# Chat Feature

## Responsibilities
- Public channel messaging and DM conversation APIs.
- WebSocket realtime transport with auth and routing.
- DM persistence and authorization checks.

## Key Paths
- `backend/features/chat/handler.go`
- `backend/features/chat/websocket.go`
- `backend/features/chat/models.go`

## Edge Cases
- Non-participant fetching DM messages.
- Initiator trying to accept own conversation request.
- Slow websocket clients during fanout.

## Failure Modes
- Message save succeeds but activity update fails.
- Connection write fails and client cleanup required.

## Test Checklist
- Conversation access control checks.
- DM history access denies non-participants.
- WebSocket handshake denies invalid token/origin.
