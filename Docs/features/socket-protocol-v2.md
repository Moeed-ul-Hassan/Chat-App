# Feature Specification: WebSocket Protocol v2

**Status**: ✅ Updated (Backend Implemented)
**Location**: `backend/features/chat/websocket.go`

## Overview

Echo's WebSocket protocol enables real-time messaging, status updates, and session management. v2 adds message IDs, persistence support, and server-side broadcast for edits and deletions.

## Message Types

### 1. `chat` (Client ↔ Server)

The primary message type for sending/receiving chat content.

```json
{
  "type": "chat",
  "id": "uuid",
  "room": "channel-pk",
  "username": "Moeed",
  "content": "Hello World!",
  "fileName": "optional",
  "reaction": "optional"
}
```

**Persistence**: All `chat` messages are automatically saved to PostgreSQL on the backend.

### 2. `msg_edited` (Server → Client)

Broadcast to all clients when a message is successfully updated via the REST API (`PUT /api/messages/{id}`).

```json
{
  "type": "msg_edited",
  "id": "uuid",
  "room": "channel-pk",
  "content": "New edited content",
  "editedAt": "ISO8601"
}
```

### 3. `msg_deleted` (Server → Client)

Broadcast to all clients when a message is soft-deleted via the REST API (`DELETE /api/messages/{id}`).

```json
{
  "type": "msg_deleted",
  "id": "uuid",
  "room": "channel-pk",
  "content": "🚫 This message was deleted"
}
```

### 4. `error` (Server → Client)

Used to notify users of failed operations or invalid connections.

```json
{
  "type": "error",
  "content": "Room and username required"
}
```

## Connection Parameters

Clients connect to `/ws` with the following URL query parameters:

- `room`: (Required) The Room ID or Channel ID to join.
- `username`: (Required) Display name of the user.
- `userId`: (Required for persistence) The UUID of the authenticated user.

## Client Behavior (Reconnection)

- **Automatic History**: When a client connects to a channel, they should first call `GET /api/channels/{id}/messages` to populate the chat history before establishing the WS connection.
- **Deduplication**: Use the message `id` to ensure same message isn't rendered twice if received via both history and live broadcast.
