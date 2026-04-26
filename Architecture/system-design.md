# Echo Backend Architecture

This document visualizes the core backend architecture of Echo, focusing on a high-performance, real-time messaging engine built with Go.

## 1. High-Level System Overview

The Echo Backend is a dedicated engine that handles authentication, real-time message routing, and persistent storage. External clients (Mobile, Web apps, or CLI tools) communicate via a dual-protocol interface.
<!--  -->
```mermaid
graph TD
    Client[External Client] -- "REST API (Clerk Auth)" --> Backend[Go Backend: Chi/Hub]
    Client -- "WebSocket (Real-time)" --> Backend
    
    subgraph "Core Engine"
        Backend -- "Persistence" --> Mongo[(MongoDB Atlas)]
        Backend -- "Cache/Presence" --> Redis[(Redis Local/Cloud)]
        Backend -- "Identity" --> Clerk[Clerk Auth Management]
    end
    
    subgraph "Documentation & Contracts"
        Backend -- "Self-Docs" --> Swagger[Swagger UI / OpenAPI]
    end
```

---

## 2. The Real-time Engine (Hub-and-Spoke)

Echo uses a **Hub-and-Spoke** architecture for real-time delivery. The Hub manages client subscriptions and broadcasts messages to the correct channels.

```mermaid
sequenceDiagram
    participant C1 as Client A
    participant H as Go WebSocket Hub
    participant R as Redis (Presence)
    participant M as MongoDB (Store)
    participant C2 as Client B

    C1->>H: Send Message {RoomID: "A1"}
    H->>M: Save Message Object
    H->>R: Update Last Seen
    H->>H: Find all clients in Room "A1"
    H-->>C1: Ack (Sent)
    H-->>C2: Broadcast {Message}
```

---

## 3. Storage & Cache Strategy

Echo balances speed and durability by using MongoDB for message history and Redis for volatile, high-speed data.

- **MongoDB**: Primary store for User Profiles, Room Metadata, and Message History. Uses TTL (Time-To-Live) indexes for ephemeral stories.
- **Redis**: Stores the "Online/Offline" status of users and handles rate-limiting to protect the API from abuse.

---

## 4. Authentication Flow (Clerk Integration)

Echo leverages **Clerk** for robust, enterprise-grade authentication. The backend verifies identities via JWT (JSON Web Tokens).

```mermaid
graph LR
    A[Client] -- "1. Login / Signup" --> B[Clerk Provider]
    B -- "2. Return Session JWT" --> A
    A -- "3. API Request + Bearer JWT" --> C[Go Backend]
    C -- "4. Verify JWT (Clerk Middleware)" --> D{Valid?}
    D -- "Yes" --> E[Process Request]
    D -- "No" --> F[401 Unauthorized]
```

---

## 5. API Contract (Swagger)

The "Source of Truth" for all backend endpoints is the Swagger documentation.
- **Local URL**: `http://localhost:8001/swagger/index.html`
- **Dynamic Doc Generation**: Built from Go source code comments using `swaggo`.
