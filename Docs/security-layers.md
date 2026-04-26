# Echo — Private Room: 7 Security Layers

**Document:** Security Architecture
**Project:** Echo Secure Messaging Platform
**Version:** 1.0
**Date:** 2026-03-30

---

## Overview

Every Private Room in Echo is protected by 7 independent, stackable security layers.
Each layer operates at a different point in the request/session lifecycle.
Disabling one layer does NOT remove the others.

```
Request comes in
        │
        ▼
┌──────────────────┐
│  Layer 1: JWT    │  ← Must pass to even reach the server
└────────┬─────────┘
         ▼
┌──────────────────┐
│  Layer 2:        │  ← Room-level passkey check
│  Passkey         │
└────────┬─────────┘
         ▼
┌──────────────────┐
│  Layer 3: 2FA    │  ← Optional per-room TOTP requirement
│  Gate            │     (Owner can enable/disable)
└────────┬─────────┘
         ▼
┌──────────────────┐
│  Layer 4:        │  ← Invite-only mode (disables search)
│  Invite Link     │
└────────┬─────────┘
         ▼
┌──────────────────┐
│  Layer 5:        │  ← All messages encrypted at rest
│  AES-256 at rest │     using room-specific AES key
└────────┬─────────┘
         ▼
┌──────────────────┐
│  Layer 6: ECDH   │  ← Ephemeral E2E keys per session
│  E2E Keys        │     (even server can't read)
└────────┬─────────┘
         ▼
┌──────────────────┐
│  Layer 7: Auto-  │  ← Messages deleted from DB after TTL
│  Destruct TTL    │     (Owner configures: 1h / 24h / 7d)
└──────────────────┘
```

---

## Layer 1 — JWT Authentication

| Property | Detail |
|----------|--------|
| **What it does** | Every WebSocket connection and REST API call must carry a valid JWT access token |
| **Where enforced** | Go middleware — before any handler runs |
| **Token lifetime** | Access token: 15 minutes · Refresh token: 30 days |
| **On failure** | `401 Unauthorized` — client silently refreshes or redirects to login |
| **Cannot be bypassed** | No — this is non-optional for ALL rooms |

**Flow:**
```
Client sends: Authorization: Bearer <access_token>
Server validates: signature, expiry, user_id exists in DB
On pass: user identity is injected into request context
On fail: 401, client attempts refresh → if refresh also fails → redirect to /login
```

---

## Layer 2 — Room Passkey (Argon2id)

| Property | Detail |
|----------|--------|
| **What it does** | User must provide the room passkey to join |
| **How stored** | Argon2id hash in PostgreSQL — original passkey never stored |
| **Where enforced** | `POST /api/rooms/:id/join` handler |
| **Max attempts** | **3 attempts** then 15-minute lockout (tracked in memory/Redis) |
| **Can owner bypass** | Yes — owner is exempt from passkey on join |
| **Cannot be bypassed** | No — required for all non-owner joins |

> **Note on Argon2id:** This is a free, open-source Go library (`golang.org/x/crypto/argon2`). Zero cost.

**Flow:**
```
User submits passkey → server hashes with same salt → compare hashes
Match: issue room session token
No match: increment attempt counter
After 3 failures: return { locked_until: "..." } error
```

---

## Layer 3 — 2FA Gate (Optional, owner-configurable)

| Property | Detail |
|----------|--------|
| **What it does** | Requires a valid TOTP 6-digit code to enter the room |
| **Default state** | OFF — owner must explicitly enable it |
| **Authenticator apps** | Google Authenticator, Authy, any TOTP-compatible app |
| **Where enforced** | After Layer 2 passes, server checks if room has `require_2fa = true` |
| **On failure** | Prompt user to enter TOTP code — 3 attempts then lockout |

**Flow:**
```
Layer 2 passes →
  if room.require_2fa == true:
    → return { requires_2fa: true }
    → client shows TOTP input screen
    → user enters 6-digit code
    → server validates against user's TOTP secret
    → pass: proceed to room
    → fail: increment counter, 3 fails = lockout
  else:
    → proceed directly to room
```

---

## Layer 4 — Unique Invite Link / Invite-Only Mode

| Property | Detail |
|----------|--------|
| **What it does** | Owner can switch room from "discoverable" to "invite-only" |
| **Invite link format** | `Echo.app/join/abc123xyz` (UUID-based, single-use or multi-use) |
| **Discoverable mode** | Room can be found via Room ID search in World Chat |
| **Invite-only mode** | Room hidden from all search — only reachable via invite link |
| **Owner control** | Can revoke and regenerate invite links at any time |

**Invite Link Types:**
```
Single-use link:   expires after 1 person uses it
Multi-use link:    works for N people (owner sets limit)
Time-limited link: expires after X hours
Permanent link:    never expires (until revoked)
```

---

## Layer 5 — AES-256 Encryption at Rest

| Property | Detail |
|----------|--------|
| **What it does** | All message content is encrypted before being written to PostgreSQL |
| **Key storage** | Each room has a unique AES-256 key, itself encrypted with the server master key |
| **Key rotation** | Key rotates automatically when owner changes the passkey |
| **What's encrypted** | Message content, file metadata, attachment names |
| **What's NOT encrypted** | Timestamps, user IDs, message IDs (needed for indexing) |

**Flow:**
```
Message arrives at server →
  Encrypt content with room_aes_key (AES-256-GCM)
  Store ciphertext + IV in PostgreSQL
  
Message retrieved →
  Fetch ciphertext from DB
  Decrypt with room_aes_key
  Send plaintext to client over encrypted WSS connection
```

---

## Layer 6 — ECDH Ephemeral E2E Keys

| Property | Detail |
|----------|--------|
| **What it does** | Each session generates a new ECDH key pair for true end-to-end encryption |
| **Key lifetime** | One session — keys are discarded when user leaves |
| **What it protects against** | Server-side compromise — even the server cannot read messages |
| **Fallback** | If ECDH handshake fails, falls back to Layer 5 (AES-256 at rest) |
| **Shown to user** | Encryption status bar shown at top of every chat and call (WhatsApp-style) |

**Key Exchange Flow:**
```
User A joins room → generates (pub_A, priv_A)
User A sends pub_A to server (plaintext — that's fine)
User B joins room → receives pub_A, generates (pub_B, priv_B)
User B computes shared_secret = ECDH(priv_B, pub_A)
User A computes shared_secret = ECDH(priv_A, pub_B)
Both now have same shared_secret without server ever seeing it
Messages encrypted with shared_secret using AES-256-GCM
```

---

## Layer 7 — Auto-Destruct (Message TTL)

| Property | Detail |
|----------|--------|
| **What it does** | Messages are permanently deleted from the database after a configured time |
| **Default state** | OFF — owner must enable it |
| **TTL options** | 1 hour · 6 hours · 24 hours · 7 days · 30 days |
| **Scope** | Applies to all messages in the room (can be per-message in future) |
| **Implementation** | PostgreSQL scheduled cleanup job + `deleted_at` timestamp |
| **User visibility** | Each message shows a countdown timer indicating when it will self-destruct |

**Flow:**
```
Message saved to DB with: expires_at = NOW() + room.ttl_duration
Background Go goroutine runs every 15 minutes:
  DELETE FROM messages WHERE expires_at < NOW()
Client receives msg_deleted event for expired messages
UI replaces content with "This message has self-destructed"
```

---

## Encryption Status Bar (WhatsApp-style)

Shown at the top of every Private Room chat and every call:

```
┌─────────────────────────────────────────────────────┐
│  🔒  End-to-End Encrypted · AES-256 · ECDH Active   │
└─────────────────────────────────────────────────────┘
```

Color states:
- 🟢 **Green bar** — All 7 layers active
- 🟡 **Yellow bar** — Some layers disabled (owner choice)
- 🔴 **Red bar** — Only basic encryption (Layer 1+5 only)

---

## Layer Summary Table

| Layer | Name | Required | Configurable | Protects Against |
|-------|------|----------|--------------|------------------|
| 1 | JWT Auth | ✅ Always | ❌ No | Unauthenticated access |
| 2 | Passkey | ✅ Always | ❌ No (but owner exempt) | Unauthorized users |
| 3 | 2FA Gate | ❌ Off by default | ✅ Yes (per room) | Compromised accounts |
| 4 | Invite Link | ❌ Off by default | ✅ Yes (per room) | Discovery / search attacks |
| 5 | AES-256 at rest | ✅ Always | ❌ No | DB breach |
| 6 | ECDH E2E | ✅ Always | ❌ No | Server-side compromise |
| 7 | Auto-Destruct | ❌ Off by default | ✅ Yes (per room) | Long-term data exposure |
