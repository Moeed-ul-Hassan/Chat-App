# Echo Backend

> REST API backend for chat, rooms, stories, and authentication — built for reliability and real-time responsiveness.

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org)
[![MongoDB](https://img.shields.io/badge/MongoDB-Atlas-47A248?style=flat&logo=mongodb)](https://www.mongodb.com)
[![Chi](https://img.shields.io/badge/Router-Chi-informational?style=flat)](https://github.com/go-chi/chi)
[![Swagger](https://img.shields.io/badge/Docs-Swagger-85EA2D?style=flat&logo=swagger)](https://swagger.io)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)

Echo backend powers JWT-protected REST endpoints, real-time WebSocket messaging, and persistent storage for users, conversations, rooms, and stories. It integrates Clerk for auth, MongoDB for persistence, and optionally Redis for caching, all exposed through self-documenting Swagger endpoints.

---

## Features

**For users and integrators:**

- 🔐 **JWT-protected REST API** via Clerk — zero-trust auth on all sensitive routes
- 💬 **Real-time WebSocket events** for chat and DM delivery
- 🗄️ **MongoDB-backed persistence** for users, conversations, messages, rooms, and stories
- 🔑 **Google OAuth2** via Goth — social login support alongside Clerk
- 📧 **Transactional email** via Resend API
- ⚡ **Optional Redis** for response caching and rate-limit enforcement
- 📖 **Swagger UI** served by the app — always in sync with handler annotations

**For operators and contributors:**

- Health, readiness, and metrics endpoints out of the box
- Docker-ready, deployable to GCP Cloud Run via CI/CD
- Feature-sliced structure — new resource types follow existing handler/model patterns

---

## Architecture

Detailed diagrams and design decisions live in the [`Architecture`](./Architecture) directory:

| Document | What's covered |
|---|---|
| [System Design](./Architecture/system-design.md) | Go backend internals, WebSocket fanout model, data layer strategy |
| [Infrastructure & Deployment](./Architecture/infrastructure.md) | Docker setup, GCP Cloud Run, CI/CD pipeline |

The main README intentionally stays lean — dive into those docs when you need the deeper picture.

---

## Project Structure

The codebase follows a feature-sliced layout. Each domain (auth, chat, room, story, user) owns its handler, models, and tests. Shared infrastructure lives in `core/`.

```
github.com/Moeed-ul-Hassan/chatapp

backend/
├── cmd/
│   └── server/
│       ├── main.go                  # Entrypoint — wires routes, middleware, DB
│       └── uploads/stories/         # Uploaded story media (runtime)
│
├── core/
│   ├── api/contracts.go             # Shared request/response types
│   ├── auth/
│   │   ├── clerk.go                 # Clerk JWT verification
│   │   └── goth.go                  # Google OAuth2 provider setup
│   ├── authz/identity.go            # Identity extraction from verified claims
│   ├── db/mongo.go                  # MongoDB client and connection lifecycle
│   ├── geoip/geoip.go               # IP geolocation (optional enrichment)
│   ├── middleware/
│   │   ├── auth.go                  # JWT enforcement middleware
│   │   ├── common.go                # Logging, recovery, CORS
│   │   └── redis_rate_limit.go      # Redis-backed rate limiting
│   └── redis/cache.go               # Redis client and cache helpers
│
├── features/
│   ├── admin/handler.go             # Admin-only routes
│   ├── auth/
│   │   ├── handler.go               # Auth endpoints (sync, session, OAuth callback)
│   │   ├── model.go
│   │   └── auth_test.go
│   ├── chat/
│   │   ├── handler.go               # Chat REST endpoints
│   │   ├── models.go
│   │   └── websocket.go             # WebSocket hub and connection management
│   ├── room/
│   │   ├── handler.go
│   │   └── models.go
│   ├── story/
│   │   ├── handler.go
│   │   └── models.go
│   └── user/
│       ├── handler.go
│       ├── model.go
│       └── user_test.go
│
├── docs/
│   ├── docs.go                      # Swaggo generated — do not edit manually
│   ├── swagger.json
│   └── swagger.yaml
│
├── Dockerfile
├── go.mod
├── go.sum
└── README.md
```

**Conventions for contributors:**

- Each feature package contains a `handler.go` (route handlers), `models.go` (DB/domain types), and optionally `*_test.go`
- Shared types used across features go in `core/api/contracts.go`
- Middleware is applied at the router level in `cmd/server/main.go`, not inside individual handlers

---

## Stack

| Layer | Technology | Why |
|---|---|---|
| Language | Go | Concurrency primitives make WebSocket handling and concurrent request processing straightforward |
| Router | Chi | Lightweight, idiomatic, composable middleware without the overhead of a full framework |
| Database | MongoDB | Flexible document model suits evolving chat/room schemas without migrations |
| WebSocket | Gorilla WebSocket | Stable, battle-tested implementation with full RFC 6455 support |
| Auth | Clerk SDK + Goth | Clerk handles JWT issuance and verification; Goth provides Google OAuth2 support |
| Token signing | PASETO | Secure alternative to JWT for session tokens |
| Email | Resend API | Transactional email delivery for notifications |
| API Docs | Swaggo | Generates Swagger spec directly from Go annotations — docs stay coupled to code |
| GeoIP | geoip | Optional IP enrichment for user context |

---

## Getting Started

### Prerequisites

- Go 1.21+
- A running MongoDB instance (local or Atlas)
- A Clerk account and secret key
- Google Cloud Console credentials *(for OAuth2 — get from APIs & Services → Credentials)*
- A Resend account and API key *(for email)*
- Redis *(optional — only required for caching and rate-limit routes)*

### 1. Configure environment

```bash
cp backend/.env.example backend/.env
```

Open `backend/.env` and fill in the values below. Required fields must be set before the server will start correctly.

```env
# MongoDB
MONGODB_URI=mongodb://...          # Required — full connection string (local or Atlas)
DB_NAME=                           # Required — database name to use
# MONGO_HOST=                      # Alternative to URI if using host/port separately
# MONGO_PORT=

# Server
PORT=8001                          # Defaults to 8001
FRONTEND_URL=http://localhost:3001 # Used for CORS and OAuth redirect URIs
GO_ENV=development                 # development | production

# Auth — Clerk
CLERK_SECRET_KEY=sk_...            # Required — from your Clerk dashboard

# Auth — Google OAuth2 (from Google Cloud Console → APIs & Services → Credentials)
GOOGLE_KEY=                        # Required for OAuth2 login
GOOGLE_SECRET=                     # Required for OAuth2 login
SESSION_SECRET=                    # Required — random string for session signing

# Token signing
PASETO_SECRET=                     # Required — secret key for PASETO token signing

# Email
RESEND_API_KEY=                    # Required for transactional email
RESEND_FROM_EMAIL=                 # Sender address (e.g. no-reply@yourdomain.com)
```

### 2. Start the server

```bash
cd backend
go run ./cmd/server/main.go
```

The server starts on the port defined in `PORT` (default `:8001`).

### 3. Verify your setup

```bash
curl http://localhost:8001/health   # Should return 200 OK
curl http://localhost:8001/ready    # Returns 200 only when Mongo is reachable
```

Then open [http://localhost:8001/swagger/index.html](http://localhost:8001/swagger/index.html) to explore all available endpoints interactively.

---

## Key Endpoints

| Endpoint | Purpose |
|---|---|
| `GET /health` | Liveness probe — confirms the server process is up |
| `GET /ready` | Readiness probe — returns 200 only when MongoDB is reachable |
| `GET /metrics` | Prometheus-compatible metrics |
| `GET /swagger/index.html` | Interactive API documentation |
| `GET /ws` | WebSocket endpoint for real-time chat and DM events |

> **Readiness vs. Health:** Use `/health` for liveness checks and `/ready` for readiness gates. In Cloud Run or Kubernetes, wire these to the appropriate probe configs.

---

## Development Workflow

### Running tests

```bash
cd backend
go test ./...
```

Tests currently only exist in `features/auth/` and `features/user/`. Integration test coverage is a known gap — see [Roadmap](#roadmap).

### Formatting

**Unix/Linux/macOS:**
```bash
cd backend
gofmt -w $(find . -name '*.go')
```

**Windows (PowerShell):**
```powershell
cd backend
gofmt -w (Get-ChildItem -Recurse -Filter *.go | % { $_.FullName })
```

### Regenerating Swagger docs

If you modify handler annotations, regenerate the Swagger spec before committing:

```bash
cd backend
swag init -g cmd/server/main.go
```

The generated files in `docs/` should be committed alongside your handler changes.

---

## Known Limitations

### WebSocket fanout is in-process

Real-time events are broadcast within a single server process (`features/chat/websocket.go`). This works correctly for single-instance deployments but will not fan out across multiple nodes.

**Impact:** Horizontal scaling is not supported for WebSocket delivery without additional infrastructure.  
**Workaround:** Deploy as a single instance (the current default on Cloud Run).  
**Planned fix:** Move to Redis pub/sub for cross-instance event delivery.

---

## Roadmap

- **Redis pub/sub fanout** — replace in-process WebSocket broadcast to unblock horizontal scaling
- **Standardized API error envelopes** — consistent error response shapes across all handlers
- **Expanded integration tests** — cover authorization edge cases and WebSocket reconnect behavior
- **Request tracing** — OpenTelemetry instrumentation once the ops environment is ready

---

## Contributing

1. Fork the repository and create a feature branch from `main`
2. Follow the existing feature-slice pattern — new domains get their own directory under `features/` with `handler.go`, `models.go`, and tests
3. Shared types belong in `core/api/contracts.go`, not in individual feature packages
4. Run `go test ./...` and `gofmt` before opening a pull request
5. For significant structural changes, open an issue first to discuss approach

Refer to the [Architecture docs](./Architecture) to understand design boundaries before making changes that touch `core/`.

---

## Deployment

Deployment is handled through Docker and GCP Cloud Run. See [Infrastructure & Deployment](./Architecture/infrastructure.md) for:

- Dockerfile structure and build process
- Cloud Run service configuration
- CI/CD pipeline setup and environment variable management

When deploying, ensure all required env vars are set as secrets/environment variables in your Cloud Run service — particularly `CLERK_SECRET_KEY`, `MONGODB_URI`, `PASETO_SECRET`, and `SESSION_SECRET`.

---

## Notes

- **Redis is optional** in local development — the app runs without it; `redis_rate_limit.go` middleware degrades gracefully when no Redis URL is configured
- **Ready check requires Mongo** — `/ready` will return non-200 if the database connection is unavailable
- **Swagger files are generated** — if you change handler annotations, re-run `swag init` before committing; the generated `docs/` files should be version controlled
- **Story uploads** are written to `cmd/server/uploads/stories/` at runtime — ensure this path is writable and excluded from version control if needed
- **`GO_ENV`** controls environment-specific behavior — always set to `production` in deployed environments