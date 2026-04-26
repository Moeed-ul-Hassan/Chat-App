# Feature Specification: Chat Data Schema

**Status**: ✅ Implemented
**Location**: `backend/features/chat/models.go` & `backend/core/db/postgres.go`

## Overview

This defines the persistence layer for Public Channels and their Messages. It handles dynamic Channel creation (based on IP Geolocation) and guarantees message history is saved in PostgreSQL.

## Table Structures

### `channels`

Represents a World Chat space, identified by ISO 3166-1 alpha-2 country codes.

```sql
CREATE TABLE channels (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    country_code CHAR(2) NOT NULL,
    name         TEXT NOT NULL,
    description  TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### `messages`

Stores actual chat content. Used by both Channels (Public) and Rooms (Private, in Milestone 3).

```sql
CREATE TABLE messages (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id    UUID REFERENCES rooms(id) ON DELETE CASCADE,
    channel_id UUID REFERENCES channels(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users(id),
    content    TEXT,
    type       TEXT NOT NULL DEFAULT 'text',
    file_name  TEXT,
    edited     BOOLEAN NOT NULL DEFAULT FALSE,
    deleted    BOOLEAN NOT NULL DEFAULT FALSE,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    edited_at  TIMESTAMPTZ
);
```

**Indexes:**

- `idx_messages_room_id` on `messages(room_id)`
- `idx_messages_channel_id` on `messages(channel_id)`
(Used to quickly fetch room or channel history without table scans)

## Go Structs

`Channel` and `Message` structs map to these tables in `features/chat/models.go`.

### `Channel`

- Maps 1:1 with DB schema. Used to route users to the correct socket hub based on their geographical location.

### `Message`

- Standard message structure containing `Edited` and `Deleted` states.
- Hydrated Fields: When querying the DB for messages, the user's `Username` and `DisplayName` are automatically `JOIN`ed directly into the struct response.

## Database Query Behaviors

- **Dynamic Channels:** If a user connects from a Country code that does not exist in the `channels` table yet, the DB creates it on the fly (e.g., "PK General").
- **Message Edit:** The database explicitly sets the `edited = TRUE` and updates the `edited_at` timestamp.
- **Message Deletion:** Handled via logical soft-delete. The DB query `SET deleted = TRUE, content = '🚫 This message was deleted'` wipes the underlying message from the database while keeping the record so the UI renders the deletion marker.
