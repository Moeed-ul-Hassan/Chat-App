# Feature Specification: Room Search & Discovery

**Status**: 🏗️ Backend Ready / Frontend Pending
**Location**: `backend/features/chat/handler.go` (Implicit in world chat logic and future Room list logic)

## Overview

Echo provides a dedicated search flow for users in the World Chat to discover and join Private Rooms using a unique Room ID.

## Search Flow UX

1. **Trigger**: User clicks the "Search" button in the World Chat side panel.
2. **Input**: A modal/input field appears asking for a "Room ID" (e.g., `falcon-7482`).
3. **Lookup**:
   - Client calls `GET /api/rooms/search?room_id={id}`.
   - Server checks if the room exists and is not `invite_only`.
4. **Result**:
   - If found: Show room card (Name, Member count, Security level).
   - If passkey required: Show a passkey input field.
   - Max 3 wrong attempts → 15-minute lockout (enforced by backend).
5. **Entry**: On successful passkey verification, the user is issued a room session and navigates to the Room view.

## API Specification (Planned for M3)

### Search Room

`GET /api/rooms/search?room_id={id}`

- **Response (Success)**:

  ```json
  {
    "id": "uuid",
    "room_id": "falcon-7482",
    "name": "Secret Project",
    "require_passkey": true,
    "security_level": 7
  }
  ```

### Join Room

`POST /api/rooms/join`

- **Body**: `{ "room_id": "...", "passkey": "..." }`
- **Logic**: Verifies Argon2id hash. Max 3 attempts.

## Security Considerations

- **Brute Force**: IPs are rate-limited on the `/join` endpoint. 3 failures = 15m cooldown.
- **Discovery**: Rooms marked `invite_only` will *never* appear in search results, even if the ID is valid.
