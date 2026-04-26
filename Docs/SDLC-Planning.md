****# Echo — SDLC Planning Phase Document

**Project:** Echo Secure Messaging Platform
**Phase:** 1 — Planning
**Version:** 2.0 (Updated after review comments)
**Date:** 2026-03-30

---

## 1. Project Vision

> **"A premium, secure, WhatsApp-grade messaging platform where public spaces are open and social like YouTube channels, and private rooms are fortresses — with voice/video parties (max 4), encrypted channels, full account ownership, and country-aware public spaces."**

---

## 2. Problem Statement

The current prototype (`main.go` + `ChatRoom.jsx`) is a proof-of-concept with no persistence, no real authentication, and no production-grade security. Users cannot:

- Create permanent accounts or stay logged in
- Own their conversation history
- Have truly private, named rooms with layered security
- Communicate via voice/video in groups
- Trust that their messages are encrypted at rest
- Use the app across sessions/devices

---

## 3. Confirmed Decisions

| Decision | Answer | Status |
|----------|--------|--------|
| **Database** | ✅ PostgreSQL — already installed | ✅ Done |
| **Email for OTP** | ✅ Resend.com (free tier, 100/day) | ✅ Done |
| **Voice/Video party max** | ✅ 4 people maximum — Peer-to-Peer mesh | 🕒 Pending |
| **Redis** | ✅ User handles Redis setup themselves | ✅ Started |
| **Deployment** | ✅ Deploy only when fully production-ready | 🕒 Future |
| **Argon2id cost** | ✅ FREE — it is a Go library | ✅ Done |
| **OAuth / Goth** | ❌ **Removed from scope (User decision)** | 🚫 Removed |

---

## 4. Stakeholder Requirements (Updated User Stories)

### 4.1 As a New User, I want to…

| ID | Story | Priority | Status |
|----|-------|----------|--------|
| U1 | Register with email + username + password | 🔴 Must Have | ✅ Done |
| U2 | Verify my email via OTP (Resend.com) before accessing the app | 🔴 Must Have | ✅ Done |
| U3 | Log in with my credentials | 🔴 Must Have | ✅ Done |
| U5 | Enable 2FA (TOTP via Google Authenticator) | 🟡 Should Have | 🟡 Partial |
| U6 | Stay logged in across sessions (JWT refresh token) | 🟡 Should Have | ✅ Done |
| U7 | Reset my password via email OTP | 🟡 Should Have | ✅ Done |

### 4.2 As a World Chat User, I want to…

| ID | Story | Priority | Status |
|----|-------|----------|--------|
| W1 | See channels relevant to my country (detected via IP geolocation) | 🔴 Must Have | ✅ Done |
| W2 | Browse country-specific public channels | 🔴 Must Have | ✅ Done |
| W3 | Send and receive messages in real time in my channel | 🔴 Must Have | ✅ Done |
| W4 | React to messages/content with emojis | 🟡 Should Have | 🕒 Pending |
| W5 | Search for private rooms using a dedicated Search button | 🔴 Must Have | ✅ Done |
| W6 | Edit or delete my own messages | 🔴 Must Have | ✅ Done |
| W7 | See a bar-style encryption indicator at the top of every chat | 🔴 Must Have | ✅ Done |
| W8 | Share files and images | 🟡 Should Have | 🕒 Pending |

> **Channel Model (YouTube-style):**
>
> - A channel is country-based (e.g., 🇵🇰 Pakistan, 🇺🇸 United States)
> - Country detected automatically by IP using a geolocation library
> - Users can watch/read/react/comment — they cannot manage or admin the channel
> - Channel admins are only internal Echo staff/moderators
> - NO voice/video calls in World Chat channels

### 4.3 As a Private Room Owner, I want to…

| ID | Story | Priority | Status |
|----|-------|----------|--------|
| P1 | Create a named private room with a passkey | 🔴 Must Have | ✅ Done |
| P2 | Get a unique, human-readable Room ID (e.g. `falcon-7482`) | 🔴 Must Have | ✅ Done |
| P3 | Generate a unique invite link (single-use, multi-use, etc.) | 🔴 Must Have | 🕒 Pending |
| P4 | Switch between discoverable and invite-only mode | 🔴 Must Have | 🕒 Pending |
| P5 | Start a Voice Party (group audio call — max 4 people) | 🔴 Must Have | 🕒 Pending |
| P6 | Start a Video Party (group video call — max 4 people) | 🔴 Must Have | 🕒 Pending |
| P7 | As party admin: mute, remove, or block other participants | 🔴 Must Have | 🕒 Pending |
| P8 | Enable message auto-destruct (TTL) | 🟡 Should Have | 🕒 Pending |
| P9 | Enable 2FA as a room-entry requirement | 🟡 Should Have | 🕒 Pending |
| P10 | Delete or edit my messages | 🔴 Must Have | ✅ Done |

### 4.4 As a Private Room Member, I want to…

| ID | Story | Priority |
|----|-------|----------|
| M1 | Find a room via Search → enter Room ID → enter passkey (max 3 attempts) | 🔴 Must Have |
| M2 | Join via a unique invite link shared by the owner | 🔴 Must Have |
| M3 | Send encrypted messages | 🔴 Must Have |
| M4 | See a WhatsApp-style encryption status bar at the top of every chat and call | 🔴 Must Have |
| M5 | Join an active Voice/Video Party | 🔴 Must Have |
| M6 | Edit or delete my own messages | 🔴 Must Have |
| M7 | See deleted messages replaced with: `🚫 This message was deleted` | 🔴 Must Have |
| M8 | Share files securely | 🟡 Should Have |

### 4.5 As any User, I want to…

| ID | Story | Priority | Status |
|----|-------|----------|--------|
| S1 | Access a full WhatsApp-like Settings panel | 🔴 Must Have | 🕒 Pending |
| S2 | Update my display name and bio | 🟡 Should Have | 🕒 Pending |
| S3 | Control my privacy (last seen, read receipts) | 🟡 Should Have | 🕒 Pending |
| S4 | Change my password | 🔴 Must Have | ✅ Done |
| S5 | View and terminate active sessions | 🟡 Should Have | 🕒 Pending |
| S6 | Delete my account permanently | 🟡 Should Have | 🕒 Pending |

---
****

## 5. Functional Requirements (Updated)

### 5.1 Authentication

| ID | Requirement |
|----|-------------|
| FR-A1 | System MUST support permanent account registration with email + username + password |
| FR-A2 | System MUST send OTP via **Resend.com** during registration and password reset |
| FR-A3 | OTP MUST expire after 10 minutes and be single-use |
| FR-A4 | Passwords MUST be hashed using **Argon2id** | ✅ Done |
| FR-A5 | System MUST issue JWT access tokens and refresh tokens | ✅ Done |
| FR-A6 | System MUST support TOTP 2FA via authenticator apps | ✅ Done |
| FR-A8 | System MUST allow viewing and revoking active login sessions | 🕒 Pending |

### 5.2 World Chat Channels

| ID | Requirement |
|----|-------------|
| FR-W1 | World Chat MUST detect user's country from their IP address on login |
| FR-W2 | Users MUST be shown channels relevant to their detected country |
| FR-W3 | Channels are YouTube-style: users watch, react, comment — only admins manage them |
| FR-W4 | Voice and video calling MUST be completely hidden and disabled in World Chat |
| FR-W5 | Users MUST be able to edit their messages (shown with "edited" label) |
| FR-W6 | Deleted messages MUST be replaced with: `🚫 This message was deleted` |
| FR-W7 | Room search MUST work via a dedicated Search button → Room ID input → if found show passkey field |

### 5.3 Private Rooms

| ID | Requirement |
|----|-------------|
| FR-P1 | Users MUST be able to create a named private room with a custom passkey |
| FR-P2 | System MUST generate a unique, human-readable Room ID |
| FR-P3 | Passkeys MUST be verified with Argon2id — never stored or transmitted in plaintext |
| FR-P4 | Passkey entry MUST be limited to **3 attempts** then lockout | 🕒 Pending |
| FR-P5 | System MUST generate unique invite links (single-use, multi-use, time-limited, permanent) |
| FR-P6 | Room owners MUST be able to toggle invite-only mode (disables search discovery) |
| FR-P7 | Message content in Private Rooms MUST be stored encrypted with AES-256 |
| FR-P8 | Users MUST be able to edit and delete messages |
| FR-P9 | Deleted messages MUST display: `🚫 This message was deleted` |
| FR-P10 | Room owners CAN enable auto-destruct TTL (configurable) |
| FR-P11 | All 7 security layers MUST be implemented (see `Docs/security-layers.md`) |

### 5.4 Voice & Video Parties

| ID | Requirement |
|----|-------------|
| FR-V1 | Voice and Video Parties MUST only be available inside Private Rooms |
| FR-V2 | Maximum **4 participants** per party (P2P mesh architecture) |
| FR-V3 | **Only the party admin (host) can mute, remove, or block other participants** |
| FR-V4 | Regular members can only mute/unmute themselves and toggle their own camera |
| FR-V5 | System MUST show a "Party Active" indicator to all room members |
| FR-V6 | Encryption status bar MUST be shown at top of all calls |

### 5.5 Settings

| ID | Requirement |
|----|-------------|
| FR-S1 | Settings MUST include: Profile, Account, Privacy, Security, Notifications, Appearance, Chats |
| FR-S2 | Profile: display name and bio only (no avatar in this phase) |
| FR-S3 | Privacy: last seen, read receipts, profile visibility |
| FR-S4 | Security: 2FA toggle, active sessions, trusted devices |

---

## 6. Non-Functional Requirements

| ID | Category | Requirement |
|----|----------|-------------|
| NFR-1 | **Performance** | WebSocket message delivery < 100ms on LAN |
| NFR-2 | **Performance** | API responses < 300ms (p95) |
| NFR-3 | **Security** | All API endpoints require valid JWT (except auth routes) |
| NFR-4 | **Security** | Rate limiting: max 3 wrong passkeys → 15 min lockout |
| NFR-5 | **Security** | HTTP cookies: `HttpOnly`, `Secure`, `SameSite=Strict` |
| NFR-6 | **Security** | CORS: only allow frontend origin |
| NFR-7 | **Scalability** | Redis Pub/Sub for WebSocket fanout (user installs Redis themselves) |
| NFR-8 | **Reliability** | PostgreSQL is source of truth; Redis is cache only |
| NFR-9 | **Usability** | Fully responsive, mobile-first UI |
| NFR-10 | **Usability** | Ethereal Echo design system throughout |
| NFR-11 | **Maintainability** | Layered backend architecture (handlers / services / models) |
| NFR-12 | **Privacy** | No analytics, no tracking, no external telemetry |

---

## 7. System Architecture

### 7.1 Layered Architecture

```
┌──────────────────────────────────────────────────────────┐
│                    CLIENT (React + Vite)                  │
│  Pages: Login · Register · OTP · 2FA · Chat · Settings   │
│  State: Zustand (auth) + React Query (server state)       │
└───────────────────────────┬──────────────────────────────┘
                            │ HTTPS (REST) + WSS (WebSocket)
┌───────────────────────────▼──────────────────────────────┐
│                     GO HTTP SERVER                        │
│  ┌─────────────┐  ┌──────────────┐  ┌─────────────────┐  │
│  │  Auth API   │  │  Rooms API   │  │  Messages API   │  │
│  │ /api/auth/* │  │ /api/rooms/* │  │ /api/messages/* │  │
│  └─────────────┘  └──────────────┘  └─────────────────┘  │
│  ┌──────────────────────────────────────────────────────┐  │
│  │             WebSocket Hub (/ws)                      │  │
│  │  Chat · Presence · Typing · Reactions · Signaling    │  │
│  └──────────────────────────────────────────────────────┘  │
│  Middleware: JWT Auth · Rate Limit · CORS · Logger         │
└───────────┬───────────────────────────┬──────────────────┘
            │                           │
  ┌─────────▼──────┐         ┌──────────▼──────┐
  │  PostgreSQL     │         │  Redis (user     │
  │  (source of     │         │  installs)       │
  │   truth)        │         │  • WebSocket     │
  │  Users          │         │    Pub/Sub       │
  │  Rooms          │         │  • Rate limits   │
  │  Messages       │         │  • Session data  │
  │  Channels       │         │  • Message cache │
  └────────────────┘          └─────────────────┘
```

### 7.2 Room Search UX Flow

```
World Chat → User clicks [Search Room] button
    ↓
Modal opens: "Enter Room ID"
  [ falcon-7482          ] [Search]
    ↓
If room found:
  Shows room card: Name · Member count · Security level
  [ Enter Passkey: ____________ ] [Join]  ← passkey field appears
  Max 3 wrong attempts → 15-min lockout

If room not found:
  "No room found with that ID"
  
If room is invite-only:
  "This room is private. You need an invite link."
```

### 7.3 Invite Link System

```
Invite link types:
  /join/abc123xyz?type=single     → expires after 1 use
  /join/abc123xyz?type=multi&n=10 → expires after 10 uses
  /join/abc123xyz?type=timed&h=24 → expires after 24 hours
  /join/abc123xyz?type=permanent  → never expires (until revoked)

Anyone who clicks the link:
  → Must still be logged in (Layer 1)
  → Must still enter passkey (Layer 2)
  → Owner can revoke the link at any time
```

### 7.4 Encryption Status Bar

Shown at top of EVERY private room chat and EVERY call:

```
┌─────────────────────────────────────────────────────────┐
│  🔒  End-to-End Encrypted · AES-256 · ECDH Active        │
└─────────────────────────────────────────────────────────┘
```

Color states:

- 🟢 Green — All active security layers passing
- 🟡 Yellow — Some optional layers disabled by owner
- 🔴 Red — Basic encryption only

---

## 8. Technology Stack

| Component | Chosen | Reason |
|-----------|--------|--------|
| **Backend language** | Go | Already in use, excellent concurrency |
| **HTTP router** | `chi` | Lightweight, idiomatic Go, great middleware support |
| **WebSocket** | `gorilla/websocket` | Already in use, battle-tested |
| **Database** | PostgreSQL | ✅ Already installed. Relational, ACID, UUID support |
| **DB driver** | `pgx/v5` | Fastest Go Postgres driver |
| **Cache / Pub-Sub** | Redis | User installs themselves (for learning) — integrated later |
| **Password hashing** | Argon2id (`golang.org/x/crypto`) | Free Go library. OWASP recommended |
| **JWT** | `golang-jwt/jwt/v5` | Standard, well-maintained |
| **OAuth / Social Login** | **Goth** | Supports Google, GitHub, many providers |
| **Email OTP** | **Resend.com** (`resend-go` SDK) | Free tier (100/day), simplest API, no credit card |
| **IP Geolocation** | `ip-api.com` (free) | Detect country for channels | ✅ Done |
| **Frontend** | React + Vite | Already in use | ✅ Done |
| **State management** | Zustand | Lightweight auth/global state | ✅ Done |
| **Server state** | TanStack Query | REST data caching | ✅ Done |
| **Routing** | React Router v6 | Standard SPA routing | ✅ Done |
| **Animations** | Framer Motion | Already in use | ✅ Done |

---

## 9. Database Schema

```sql
-- Permanent user accounts
CREATE TABLE users (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  username     TEXT UNIQUE NOT NULL,
  email        TEXT UNIQUE NOT NULL,
  password     TEXT,                        -- Argon2id hash (NULL for OAuth users)
  display_name TEXT,
  bio          TEXT,
  country_code CHAR(2),                     -- Detected on first login via IP
  totp_secret  TEXT,
  totp_enabled BOOLEAN DEFAULT FALSE,
  verified     BOOLEAN DEFAULT FALSE,
  created_at   TIMESTAMPTZ DEFAULT NOW(),
  last_seen    TIMESTAMPTZ
);

-- OAuth providers (for Goth)
CREATE TABLE oauth_accounts (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     UUID REFERENCES users(id) ON DELETE CASCADE,
  provider    TEXT NOT NULL,               -- 'google', 'github'
  provider_id TEXT NOT NULL,
  UNIQUE(provider, provider_id)
);

-- Private rooms
CREATE TABLE rooms (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  room_id     TEXT UNIQUE NOT NULL,        -- e.g. 'falcon-7482'
  name        TEXT NOT NULL,
  passkey     TEXT NOT NULL,               -- Argon2id hash
  owner_id    UUID REFERENCES users(id),
  is_private  BOOLEAN DEFAULT TRUE,
  invite_only BOOLEAN DEFAULT FALSE,
  require_2fa BOOLEAN DEFAULT FALSE,
  ttl_hours   INTEGER,                     -- NULL = no auto-destruct
  aes_key     TEXT NOT NULL,               -- AES-256 key (encrypted with server master key)
  created_at  TIMESTAMPTZ DEFAULT NOW()
);

-- Invite links
CREATE TABLE invite_links (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  room_id     UUID REFERENCES rooms(id) ON DELETE CASCADE,
  token       TEXT UNIQUE NOT NULL,
  type        TEXT NOT NULL,               -- 'single' | 'multi' | 'timed' | 'permanent'
  uses_limit  INTEGER,                     -- NULL = unlimited
  uses_count  INTEGER DEFAULT 0,
  expires_at  TIMESTAMPTZ,                 -- NULL = never
  created_at  TIMESTAMPTZ DEFAULT NOW()
);

-- World chat channels (country-based)
CREATE TABLE channels (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  country_code CHAR(2) NOT NULL,
  name         TEXT NOT NULL,              -- e.g. 'Pakistan General'
  description  TEXT,
  created_at   TIMESTAMPTZ DEFAULT NOW()
);

-- Messages (rooms + channels)
CREATE TABLE messages (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  room_id    UUID REFERENCES rooms(id),     -- NULL for channel messages
  channel_id UUID REFERENCES channels(id),  -- NULL for room messages
  user_id    UUID REFERENCES users(id),
  content    TEXT,                          -- Encrypted if room message
  type       TEXT DEFAULT 'text',           -- 'text' | 'image' | 'file' | 'voice'
  file_name  TEXT,
  edited     BOOLEAN DEFAULT FALSE,
  deleted    BOOLEAN DEFAULT FALSE,
  expires_at TIMESTAMPTZ,                   -- For auto-destruct
  created_at TIMESTAMPTZ DEFAULT NOW(),
  edited_at  TIMESTAMPTZ
);

-- Active login sessions
CREATE TABLE sessions (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     UUID REFERENCES users(id) ON DELETE CASCADE,
  token_hash  TEXT NOT NULL,               -- hash of refresh token
  device_info TEXT,
  ip_address  TEXT,
  created_at  TIMESTAMPTZ DEFAULT NOW(),
  expires_at  TIMESTAMPTZ NOT NULL
);
```

---

## 10. Risk Register

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Resend.com free tier exceeded | Low | Medium | 100/day is plenty for dev/testing |
| P2P mesh fails at 4 participants | Low | Medium | WebRTC mesh works reliably at 4 — tested limitation |
| IP geolocation inaccurate (VPN) | High | Low | Show detected country + let user manually change |
| Argon2id hashing too slow | Low | Low | Tune memory/iterations params (default is fast enough) |
| Room ID collision | Very Low | Low | Verify uniqueness in DB before confirming |
| Passkey brute force | Medium | High | 3-attempt lockout + rate limiting by IP |
| JWT token leak | Low | High | HttpOnly cookies + short 15-min access token TTL |
| Redis not ready for Milestone 1 | High | Low | Milestone 1 works without Redis (in-memory fallback) |

---

## 11. Milestone Plan (Updated)

```
Milestone 1 — Auth Foundation (90% Complete)
  Backend: Register, Login, JWT, OTP via Resend, 2FA
  Frontend: Login, Register, OTP verify, 2FA setup screens

Milestone 2 — World Chat Channels (95% Complete)
  Backend: IP geolocation, country channels, message persistence, edit/delete
  Frontend: Country-based channel list, message actions, encryption bar

Milestone 3 — Private Rooms (70% Complete)
  Backend: Room creation, passkey (Argon2id), 3-attempt lockout (Pending), invite links (Pending),
           Room search API, 7 security layers
  Frontend: Room search flow, room creation, passkey entry, invite link UI

Milestone 4 — Settings (10% Complete)
  Backend: Profile update, session management, privacy flags
  Frontend: WhatsApp-like settings (Profile, Account, Privacy, Security,
           Notifications, Appearance, Chats)

Milestone 5 — Redis Integration (30% Complete)
  Backend: Message cache, presence (Done), rate limiting, Pub/Sub fanout

Milestone 6 — Voice & Video Parties (5% Complete)
  Backend: P2P WebRTC signaling, party admin controls
  Frontend: VoiceParty.jsx, VideoParty.jsx, party admin UI

Total: ~23–31 days
Deploy: After Milestone 6 is fully tested
```

---

## 12. Definition of Done

A milestone is complete when:

- [ ] All Must Have stories for that milestone pass manual testing
- [ ] No unhandled panics in the Go backend
- [ ] All API endpoints return correct HTTP status codes
- [ ] UI shows correct state for all success AND error paths
- [ ] No sensitive data (passwords, tokens, keys) logged to console
- [ ] Encryption status bar shows correctly in all relevant views

---

## 13. Referenced Documents

| Document | Location |
|----------|----------|
| Security Layers Detail | `Docs/security-layers.md` |
| This SDLC Plan | `Docs/SDLC-Planning.md` |
| Architecture Diagrams | `Architecture/` (existing) |
