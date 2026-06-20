# Airstage × LiveKit — Deep Technical Architecture Guide

> Purpose: Architectural mindmap for full LiveKit integration. Theory-first, structure-second. No code to copy — mental models to internalize before writing a single line.

---

## 1. What LiveKit Actually Is (Mental Model First)

LiveKit is a **Selective Forwarding Unit (SFU)** exposed as a managed cloud service. Understanding what an SFU does versus alternatives is foundational.

### 1.1 Media Architecture Topology

**Mesh (naive):** Every participant sends media to every other participant. N participants = N×(N-1) upload streams per client. Collapses at 4+ people. No server involvement.

**MCU (Multipoint Control Unit):** Server receives all streams, mixes them into one composite stream, sends one stream back per client. Server does all decoding+encoding. Centralized CPU bottleneck. Older Zoom-style architecture.

**SFU (Selective Forwarding Unit) — what LiveKit is:** Server receives streams from each publisher. For each subscriber, server selects which streams to forward (based on subscription state, bandwidth, simulcast layer). Server does NOT decode or re-encode. It forwards RTP packets. This is why LiveKit Cloud can handle hundreds of participants per room without linear CPU scaling — the server is a smart router, not a transcoder.

**Implication for your architecture:** Your Go backend never sees a single audio or video byte. The SFU is the LiveKit Cloud infrastructure. Your backend's job is: token authority, room lifecycle control via API, and webhook event processing.

### 1.2 WebRTC Signaling vs. Media Path

WebRTC has two distinct channels:

**Signaling channel:** How peers negotiate what codec to use, what ICE candidates exist, what tracks are offered. LiveKit manages this entirely over its own WebSocket connection between the client SDK and LiveKit Cloud. Your backend is not in this path.

**Media channel:** Actual RTP/RTCP packets carrying audio/video. Flows between client and LiveKit Cloud via DTLS-SRTP (encrypted UDP). Your backend is not in this path.

**Your backend only touches:** the REST/gRPC API calls to LiveKit Room Service (room CRUD, participant ops), token generation, and inbound webhooks. This is the clean architectural boundary that makes the system scalable — media complexity is fully delegated.

---

## 2. The Token System — Authority Delegation

### 2.1 Why Tokens Exist

LiveKit uses a **capability-based security model**. The token is a signed JWT that grants specific permissions to a specific identity in a specific room. LiveKit Cloud trusts your token because it was signed with your API Secret, which only your backend knows.

The token is the authorization artifact — it answers: "Can this identity join this room, and with what permissions?" Your backend is the **token authority**. This is architecturally equivalent to OAuth's authorization server role.

### 2.2 Token Anatomy

A LiveKit token carries three categories of information:

**Identity:** The `sub` (subject) claim = the participant's identity string in the room. You set this to your `user_id` (UUID). LiveKit uses this as the stable identifier for participant operations (mute, remove, promote). It must be unique per participant per room.

**VideoGrant:** A structured claims object embedded in the JWT that specifies exact capabilities. The grant is NOT a role name — it is a set of boolean flags. `can_publish`, `can_subscribe`, `room_admin`, `room_create`, `room_join`. Your backend translates role (host/speaker/viewer) into the correct grant flag combination at token generation time. Once the token is issued, these permissions are fixed for the session lifetime.

**Metadata:** An opaque string attached to the participant, visible to all room participants via the SDK. You serialize JSON here: `{username, plan, avatar_url}`. Frontend SDKs can read this to render participant UI without an extra API call.

**Token lifetime:** Set to 4–6 hours (room session duration). Token expiry does not disconnect a participant mid-session — the SDK maintains the session. Expiry only prevents new joins with the same token.

### 2.3 Token Generation as a Security Boundary

The endpoint `GET /rooms/:id/token` is where plan enforcement happens. Before generating any token:

1. Verify the room exists in your DB
2. Verify the room status is `draft` or `live` (not `ended`)
3. Verify participant count < plan maximum
4. Determine role: if `user_id == room.host_id` → host grant; else → viewer grant by default
5. Verify the requesting user's plan satisfies room requirements

This is why token generation must go through your backend and cannot be done client-side — the client cannot be trusted to self-assign `room_admin: true`.

---

## 3. Room Lifecycle — Two State Machines Running in Parallel

### 3.1 The Dual State Problem

When you create a room, you have two representations of it:

- **Your DB record:** `status = 'draft'`, `livekit_room_name = 'room_abc123'`
- **LiveKit Cloud:** No room exists yet (room is created on first participant join OR on explicit API call)

These two representations can diverge. Reconciliation is the mechanism that re-syncs them. Understanding this divergence is critical to building a reliable system.

### 3.2 LiveKit Room Creation vs. Your DB Record

**Explicit creation (your architecture):** You call `RoomService.C# Airstage × LiveKit — Deep Technical Architecture Guide

> Purpose: Architectural mindmap for full LiveKit integration. Theory-first, structure-second. No code to copy — mental models to internalize before writing a single line.

---

## 1. What LiveKit Actually Is (Mental Model First)

LiveKit is a **Selective Forwarding Unit (SFU)** exposed as a managed cloud service. Understanding what an SFU does versus alternatives is foundational.

### 1.1 Media Architecture Topology

**Mesh (naive):** Every participant sends media to every other participant. N participants = N×(N-1) upload streams per client. Collapses at 4+ people. No server involvement.

**MCU (Multipoint Control Unit):** Server receives all streams, mixes them into one composite stream, sends one stream back per client. Server does all decoding+encoding. Centralized CPU bottleneck. Older Zoom-style architecture.

**SFU (Selective Forwarding Unit) — what LiveKit is:** Server receives streams from each publisher. For each subscriber, server selects which streams to forward (based on subscription state, bandwidth, simulcast layer). Server does NOT decode or re-encode. It forwards RTP packets. This is why LiveKit Cloud can handle hundreds of participants per room without linear CPU scaling — the server is a smart router, not a transcoder.

**Implication for your architecture:** Your Go backend never sees a single audio or video byte. The SFU is the LiveKit Cloud infrastructure. Your backend's job is: token authority, room lifecycle control via API, and webhook event processing.

### 1.2 WebRTC Signaling vs. Media Path

WebRTC has two distinct channels:

**Signaling channel:** How peers negotiate what codec to use, what ICE candidates exist, what tracks are offered. LiveKit manages this entirely over its own WebSocket connection between the client SDK and LiveKit Cloud. Your backend is not in this path.

**Media channel:** Actual RTP/RTCP packets carrying audio/video. Flows between client and LiveKit Cloud via DTLS-SRTP (encrypted UDP). Your backend is not in this path.

**Your backend only touches:** the REST/gRPC API calls to LiveKit Room Service (room CRUD, participant ops), token generation, and inbound webhooks. This is the clean architectural boundary that makes the system scalable — media complexity is fully delegated.

---

## 2. The Token System — Authority Delegation

### 2.1 Why Tokens Exist

LiveKit uses a **capability-based security model**. The token is a signed JWT that grants specific permissions to a specific identity in a specific room. LiveKit Cloud trusts your token because it was signed with your API Secret, which only your backend knows.

The token is the authorization artifact — it answers: "Can this identity join this room, and with what permissions?" Your backend is the **token authority**. This is architecturally equivalent to OAuth's authorization server role.

### 2.2 Token Anatomy

A LiveKit token carries three categories of information:

**Identity:** The `sub` (subject) claim = the participant's identity string in the room. You set this to your `user_id` (UUID). LiveKit uses this as the stable identifier for participant operations (mute, remove, promote). It must be unique per participant per room.

**VideoGrant:** A structured claims object embedded in the JWT that specifies exact capabilities. The grant is NOT a role name — it is a set of boolean flags. `can_publish`, `can_subscribe`, `room_admin`, `room_create`, `room_join`. Your backend translates role (host/speaker/viewer) into the correct grant flag combination at token generation time. Once the token is issued, these permissions are fixed for the session lifetime.

**Metadata:** An opaque string attached to the participant, visible to all room participants via the SDK. You serialize JSON here: `{username, plan, avatar_url}`. Frontend SDKs can read this to render participant UI without an extra API call.

**Token lifetime:** Set to 4–6 hours (room session duration). Token expiry does not disconnect a participant mid-session — the SDK maintains the session. Expiry only prevents new joins with the same token.

### 2.3 Token Generation as a Security Boundary

The endpoint `GET /rooms/:id/token` is where plan enforcement happens. Before generating any token:

1. Verify the room exists in your DB
2. Verify the room status is `draft` or `live` (not `ended`)
3. Verify participant count < plan maximum
4. Determine role: if `user_id == room.host_id` → host grant; else → viewer grant by default
5. Verify the requesting user's plan satisfies room requirements

This is why token generation must go through your backend and cannot be done client-side — the client cannot be trusted to self-assign `room_admin: true`.

---

## 3. Room Lifecycle — Two State Machines Running in Parallel

### 3.1 The Dual State Problem

When you create a room, you have two representations of it:

- **Your DB record:** `status = 'draft'`, `livekit_room_name = 'room_abc123'`
- **LiveKit Cloud:** No room exists yet (room is created on first participant join OR on explicit API call)

These two representations can diverge. Reconciliation is the mechanism that re-syncs them. Understanding this divergence is critical to building a reliable system.

### 3.2 LiveKit Room Creation vs. Your DB RecordreateRoom`on room creation. This pre-creates the LiveKit room with your constraints (max_participants, empty_timeout, metadata). The room exists in LiveKit before anyone joins. This is the`auto_create: false, explicit_creation: true` mode.

**Why explicit creation matters:** It lets you enforce plan limits at the LiveKit infrastructure level, not just at the token level. Even if someone had a valid token, LiveKit would reject the join if `max_participants` is reached. Defense in depth — plan enforcement at both token generation AND room configuration.

**`empty_timeout`:** If all participants leave, LiveKit keeps the room open for this many seconds before destroying it. Set to 300s (5min). This handles brief disconnections without destroying the room.

**`departure_timeout`:** After a participant disconnect, LiveKit waits this long before considering them gone. Set to 60s. During this window, reconnection is seamless without re-joining.

### 3.3 Room State Transitions via Webhooks

The definitive room state is driven by LiveKit webhooks, not by your API calls. This is the event-sourcing principle applied to room lifecycle:

```
Your API call: POST /rooms → DB: status='draft' → LiveKit: CreateRoom
LiveKit fires: room_started → DB: status='live', started_at=now
[participants join and interact]
Host calls: DELETE /rooms/:id → LiveKit: DeleteRoom
LiveKit fires: room_finished → DB: status='ended', ended_at=now → enqueue transcription job
```

The `room_finished` webhook is your authoritative signal that the room is done. Never mark a room as ended based solely on your API call — wait for the webhook confirmation. This handles edge cases where the API call succeeds but the room teardown has side effects (egress still running, participants still connected).

---

## 4. Webhook Architecture — Event-Driven State Machine

### 4.1 Webhook Delivery Guarantees and What They Mean

LiveKit delivers webhooks **at-least-once with retries**. This means:

- You will receive every event eventually (not guaranteed in order)
- You may receive the same event multiple times
- Network partition can delay delivery significantly

**Implication:** Your webhook handler must be **idempotent** for every event type. Processing the same event twice must produce the same final state as processing it once. This is achieved by storing the `event_id` (LiveKit's unique ID per webhook) in `webhook_events` table and checking before processing: `ON CONFLICT (event_id) DO NOTHING`.

### 4.2 Event Ordering and Out-of-Order Delivery

LiveKit does not guarantee webhook order. `participant_left` may arrive before `participant_joined`. `egress_ended` may arrive before `egress_started`. Your handler must be defensive:

**Pattern: Upsert, not insert.** For participant records: `INSERT INTO room_participants ... ON CONFLICT (room_id, identity) DO UPDATE SET ...`. This handles both orders correctly.

**Pattern: Status only advances, never regresses.** For room status: only update if the new status is "higher" in the FSM. `room_finished` should only set `status='ended'` if current status is `'live'`, not if it's already `'ended'`. Use conditional UPDATE: `WHERE status != 'ended'`.

### 4.3 Webhook Receiver Architecture Shape

The webhook handler has one job: **receive, validate, persist, and dispatch fast**. It must return HTTP 200 within LiveKit's timeout window (typically 5s). All actual processing is async.

**Synchronous (in the handler, blocking):**

1. Read raw body (needed before any parsing for HMAC)
2. Validate HMAC signature against API Secret
3. Check replay: query `webhook_events` for event_id
4. If new: INSERT into `webhook_events` (idempotency record)
5. Dispatch to lightweight in-process handler (only DB writes, no external HTTP calls)
6. Return 200

**Asynchronous (enqueued to Redis, processed by workers):**

- Transcription job enqueue (room_finished)
- Summary job enqueue (transcript completed)
- Notification job enqueue (summary completed)
- Any call to Deepgram, OpenAI, Spaces

The rule: **never call an external HTTP API inside a webhook handler**. If Deepgram is slow (10s response), your webhook times out, LiveKit retries, you process twice.

### 4.4 HMAC Signature Validation — The Protocol

LiveKit signs webhooks using JWT with a `sha256` claim. The validation protocol:

1. Read the raw `Authorization` header — this is a LiveKit API token (JWT signed with your API Secret)
2. Parse and verify the JWT using your API Secret → you get a `ClaimGrants` struct
3. `ClaimGrants.SHA256` is a hex-encoded SHA-256 hash of the raw request body
4. Compute SHA-256 of the raw body bytes you received
5. Compare hex strings — they must match exactly
6. If either step fails → reject with 401, log the attempt, do NOT process

This protocol ensures: (a) the sender knows your API Secret (authenticity), and (b) the body was not tampered in transit (integrity). Raw body must be read BEFORE any JSON parsing — parsing may normalize whitespace, breaking the hash.

---

## 5. Egress — Recording Architecture

### 5.1 What Egress Is and How It Works

Egress is LiveKit's system for extracting media from a room and sending it somewhere. It is a separate managed process that joins the room as a participant, subscribes to all tracks, encodes them, and writes to a storage target.

**Room Composite Egress:** Records the entire room as a single mixed video file. The egress process renders the room as if a user were viewing it (with layout), captures the screen, and encodes to MP4. This produces one file regardless of participant count. You specify the layout template.

**The key insight:** You do not run any media software. You tell LiveKit "start recording this room, put the output here (S3 bucket + path)". LiveKit's egress infrastructure handles ffmpeg, muxing, encoding, upload. You receive the result via webhook.

### 5.2 Egress State Machine and Your System

When `StartRoomCompositeEgress` API call succeeds, you get back an `egress_id`. This is your handle to track recording state.

```
API call: StartRoomCompositeEgress → egress_id returned
  → INSERT recordings (egress_id, status='pending', room_id)

Webhook: egress_started → UPDATE recordings SET status='recording' WHERE egress_id=?
Webhook: egress_updated → (optional) UPDATE progress metadata
Webhook: egress_ended (success) → UPDATE recordings SET status='completed', storage_url, size_bytes
                                 → enqueue transcription job
Webhook: egress_ended (failure) → UPDATE recordings SET status='failed'
                                 → enqueue retry or move to DLQ
```

**`storage_url`** in `egress_ended` webhook is the full S3 key path. Store this in your DB. When a user requests the recording, generate a presigned GET URL from this key — never expose the raw S3 URL.

### 5.3 Why Presigned URLs and Not Public Bucket

S3 presigned URLs give you **per-request, time-bounded, authenticated access** to private objects. Benefits:

- Bucket remains private — no accidental public exposure
- URL expires (15min) — stolen URL has limited blast radius
- Every access goes through your API (authenticate → check plan → presign → return)
- You can revoke access by not presigning (e.g., if subscription lapses)
- Audit trail: you know when each user accessed a recording

The flow: client calls `GET /rooms/:id/recording` → your backend verifies auth + plan → calls `s3.PresignGetObject(key, 15min)` → returns the signed URL → client fetches directly from Spaces CDN.

---

ssed a recording

The flow: client calls `GET /rooms/:id/recording` → your backend verifies auth + plan → calls `s3.PresignGetObject(key, 15min)` → returns the signed URL → client f

## 6. Participant Management — LiveKit Room Service API

### 6.1 The Three Participant Operations and Their Semantics

**RemoveParticipant:** Immediately disconnects the participant from the room. They can rejoin if they have a valid token — this is an eviction, not a ban. To prevent rejoin, you must also invalidate their token (no mechanism in LiveKit for this — you do it by tracking `removed_participants` in your DB and checking at token generation).

**MutePublishedTrack:** Forces a track to be server-side muted. The participant's SDK still thinks they're publishing, but LiveKit stops forwarding that track to subscribers. This is a moderator control, not a user control. The participant will see their local preview but recipients see nothing.

**UpdateParticipant (promote/demote):** Updates the participant's `VideoGrant` mid-session without reconnection. This is how you promote a viewer to speaker. You call `UpdateParticipant` with a new permissions object. LiveKit applies it to the live session — the participant immediately gains/loses publish rights. This is the most powerful participant operation.

### 6.2 Participant Data and Track State

LiveKit tracks participant state: which tracks are published, subscription state per subscriber, participant metadata. When you call `ListParticipants`, you get a real-time snapshot of who is in the room and what tracks they have.

Your `room_participants` DB table is NOT a source of truth for live state — it is an attendance log. The source of truth for live participant state is LiveKit's API. Your DB captures historical data: when they joined, when they left, their role.

---

## 7. Frontend ↔ Backend ↔ LiveKit — Data Flow Topology

### 7.1 The Three Independent Communication Channels

Understanding that three channels co-exist simultaneously is key to avoiding architectural confusion:

**Channel 1: Frontend ↔ Your Backend (REST/WebSocket)**

- Auth, room creation, token fetch, chat (private messages), user presence
- Your existing WS hub handles private chat here
- All business logic, persistence, authorization

**Channel 2: Frontend ↔ LiveKit Cloud (WebRTC + LiveKit WebSocket)**

- All real-time media: audio, video, screen share
- Room chat via LiveKit Data Channel (send/receive arbitrary binary/text to room participants)
- Participant state events (track published, participant joined) delivered to frontend via SDK events
- Your backend is NOT in this channel

**Channel 3: LiveKit Cloud → Your Backend (Webhooks)**

- State change events: room lifecycle, participant lifecycle, egress lifecycle
- One-directional: LiveKit → you
- Your backend updates DB and enqueues async jobs based on these events

**The architectural principle:** These three channels are decoupled. Frontend doesn't wait for your backend's webhook processing before using room features. LiveKit doesn't wait for your DB write before delivering media. Each channel is independently reliable.

### 7.2 Room Chat: LiveKit Data Channel vs. Your WS Hub

You have two chat systems that must coexist:

**Private chat (user-to-user, cross-room):** Your existing WS hub. Stored in Postgres `messages` table. Works whether or not a LiveKit room is active.

**Room chat (in-room, ephemeral):** LiveKit Data Channel. The LiveKit SDK provides a `sendData()` method that broadcasts arbitrary bytes to room participants. Your backend is NOT involved — frontend sends data to LiveKit, LiveKit forwards to all subscribers. No DB storage by default (optional: you can receive room chat via webhook if you implement a LiveKit Agent that joins the room, but this is future scope).

**Why this split:** Room chat should feel real-time with sub-100ms delivery. Routing through your backend adds latency and creates a single point of failure. Data Channel gives you peer-like speed with SFU reliability.

---

## 8. Deployment Topology — DigitalOcean + LiveKit Cloud

### 8.1 Network Topology and Trust Boundaries

```
Internet
  │
  ├── Users (browsers/apps)
  │     │
  │     ├── HTTPS/WSS → Nginx (DO Droplet :443)
  │     │                 │
  │     │                 └── Proxy → Go Backend (:8080)
  │     │                              │
  │     │                              ├── Postgres (Neon, external)
  │     │                              ├── Redis (Droplet :6379, loopback only)
  │     │                              └── LiveKit API (HTTPS, outbound)
  │     │
  │     └── WebRTC/WSS → LiveKit Cloud (external, direct)
  │
  └── LiveKit Cloud
        └── Webhooks → HTTPS → Nginx → Go Backend (:8080)
```

**Trust boundary:** Redis must ONLY be accessible on `127.0.0.1`. It must NEVER be exposed on `0.0.0.0` without authentication — Redis by default has no auth and no encryption. UFW rules enforce this: port 6379 is not in the allowed list.

**LiveKit as outbound dependency:** Your Go backend calls LiveKit's API (Room Service, Egress) over HTTPS. LiveKit calls your backend (webhooks) over HTTPS. This mutual relationship means both need to be reachable. Your droplet needs outbound internet access (default) and inbound HTTPS on 443.

### 8.2 Nginx as Protocol Termination Layer

Nginx handles three distinct protocol concerns before traffic reaches Go:

**TLS termination:** All inbound traffic is HTTPS. Nginx decrypts TLS, proxies plaintext HTTP to Go on loopback. Go never sees TLS — simpler Go server config, TLS managed by certbot/Nginx.

**WebSocket upgrade:** The `Upgrade: websocket` + `Connection: upgrade` headers must be passed through. Without explicit Nginx config, Nginx strips these headers and the WebSocket handshake fails. The WS proxy location must set these headers explicitly.

**Static file serving:** The React frontend (built to `dist/`) is served directly by Nginx from disk — never hits Go. This offloads static asset serving from your Go process entirely.

### 8.3 systemd as Process Supervisor

systemd provides: automatic restart on crash, environment variable injection, resource limits, journal log capture, and dependency ordering (`After=redis.service`).

**Graceful shutdown sequence:** SIGTERM → Go server calls `srv.Shutdown(ctx)` with 20s timeout → existing requests drain → WS connections receive shutdown event and close → systemd sees process exit 0. `TimeoutStopSec=25s` gives systemd 25 seconds before sending SIGKILL. Your graceful shutdown logic must complete within this window.

**Why `Type=simple` not `Type=forking`:** Go programs don't fork. They're a single process. `Type=simple` tells systemd the process is the main service process itself — no PID file needed.

### 8.4 LiveKit Cloud Project Configuration

LiveKit Cloud projects isolate API keys, webhooks, egress config, and usage metrics. You need:

**Separate projects for dev vs. prod:** Never use production API keys in local dev. LiveKit Cloud free tier supports multiple projects. Create `airstage-dev` and `airstage-prod`.

**Webhook endpoint:** Must be publicly reachable HTTPS. In local dev, use `ngrok http 8080` to get a temporary public URL. In production, this is `https://api.yourdomain.com/webhooks/livekit`.

**Webhook retry behavior:** LiveKit retries failed webhooks (non-200 response or timeout) with exponential backoff. If your webhook handler is slow (DB contention, Redis unavailable), LiveKit will retry. Your idempotency mechanism (`webhook_events` table) handles duplicate delivery from retries.

**Egress S3 config:** LiveKit Egress needs write access to your Spaces bucket. You pass S3 credentials inside the `StartRoomCompositeEgress` API call payload (not stored in LiveKit console). The egress process uses these credentials to upload directly. Your Go backend is not in the upload path — LiveKit does the upload, you get notified via `egress_ended` webhook when it's complete.

---

## 9. Job Queue — Architectural Theory

### 9.1 Why Redis LIST and Not Redis Streams or Postgres SKIP LOCKED

Three realistic options for job queuing at this scale:

**Redis LIST (LPUSH/BRPOP):** Simple, zero setup, fast. BRPOP blocks workers until a job arrives — no polling. LIFO by default (use RPUSH/BLPOP for FIFO). Ephemeral — jobs lost if Redis crashes without persistence. Acceptable because Postgres is the durable mirror.

**Redis Streams:** Persistent, consumer groups, message acknowledgment, replay. More complex than needed for MVP. Use when you need multi-consumer fan-out or audit replay.

**Postgres `SKIP LOCKED`:** `SELECT ... FOR UPDATE SKIP LOCKED` implements a queue using your existing Postgres. Zero additional infrastructure. Slower than Redis for high-throughput jobs. Excellent for low-volume jobs where you want full transactionality.

**Your choice (Redis LIST):** Correct for MVP. The Postgres mirror provides durability. The reconciler re-queues from Postgres on crash. This gives you Redis performance with Postgres durability without the complexity of Redis Streams or Postgres SKIP LOCKED at high throughput.

### 9.2 Worker Pool Sizing and Backpressure

**Concurrency = how many jobs of each type run simultaneously.** Too low → queue builds up. Too high → overwhelms external APIs (Deepgram rate limits, OpenAI rate limits, Postgres connection pool exhaustion).

Deepgram and OpenAI both have rate limits measured in concurrent requests and requests-per-minute. Setting worker concurrency to 3 for transcription means max 3 concurrent Deepgram API calls. Stay under their free/paid tier limits.

**Backpressure:** If the queue depth keeps growing, workers can't keep up. Monitoring `job_queue_depth` metric per queue type tells you this. Response: increase concurrency (if external API allows) or scale to separate worker process.

### 9.3 Exponential Backoff and Why

Transient failures (network blip, API rate limit, Deepgram momentarily unavailable) are common. Immediately retrying creates thundering herd: all failed jobs retry simultaneously, hammering the recovering service, causing more failures.

Exponential backoff spreads retries over time: attempt 1 → wait 2s → attempt 2 → wait 4s → attempt 3 → wait 8s → attempt 4 → wait 16s → attempt 5 → DLQ.

**The DLQ is not failure — it's observability.** A job in the DLQ means: 5 attempts failed, human inspection needed. You query `jobs:dlq` LIST in Redis to see failed job payloads. You inspect, fix the root cause, then manually re-enqueue. Without DLQ, jobs silently disappear.

---

## 10. Reconciliation — The Safety Net Theory

### 10.1 Why Webhooks Are Not Enough

Webhooks can fail to deliver for reasons outside your control: your server was down during delivery window, network partition between LiveKit and your droplet, your webhook handler returned 500 and retry window expired.

Result: your DB state diverges from reality. Room marked `live` but LiveKit room is gone. Recording marked `recording` but egress was never started. Job marked `pending` but never queued in Redis (Redis restarted, job lost from LIST).

Reconciliation is the **periodic correction mechanism** that detects and repairs these divergences.

### 10.2 Reconciliation Checks and Their Logic

**Stuck live rooms:** Query rooms WHERE `status='live' AND started_at < now() - max_duration`. Call LiveKit `GetRoom` — if not found, mark `ended`. If found, verify participant count matches.

**Stuck recordings:** Query recordings WHERE `status='recording' AND created_at < now() - 2h`. Call LiveKit `ListEgress(egress_id)` — if not found or status=`ended`, update DB. Enqueue transcription if completed.

**Orphaned Postgres jobs:** Query `job_queue` WHERE `status='pending' AND created_at < now() - 10min`. These were written to Postgres but never pushed to Redis (crash between the two writes). Re-push to Redis LIST. This is the Postgres-first pattern paying off.

**The reconciler runs every 5 minutes** — not every second. It's a correction mechanism, not a real-time system. Real-time delivery is webhooks. Reconciliation is the eventual consistency guarantee.

---

## 11. Package Design — Internal Boundary Rules

### 11.1 Dependency Direction (What Can Import What)

```
cmd/api/main.go
  └── internal/app        (wires everything)
        ├── internal/rooms       → internal/livekit (service only, not SDK directly)
        ├── internal/webhooks    → internal/rooms, internal/recordings, internal/jobs
        ├── internal/recordings  → internal/livekit (egress), pkg/storage
        ├── internal/transcripts → (external: Deepgram HTTP), internal/jobs
        ├── internal/summaries   → (external: OpenAI HTTP), internal/jobs
        ├── internal/jobs        → pkg/cache (Redis), pkg/database
        └── internal/chat        → pkg/cache (Redis pubsub), pkg/database

pkg/ packages import nothing from internal/
internal/ packages import from pkg/ only, never from cmd/
```

**The critical rule:** `internal/livekit` wraps the LiveKit SDK. No other package imports the LiveKit SDK directly. `internal/rooms` calls `internal/livekit.Service` interface. This means if LiveKit changes their SDK API, you fix it in one place.

### 11.2 Interface Declaration Location (Kennedy's Rule)

Interfaces are declared where they are **consumed**, not where they are **implemented**.

`internal/rooms/service.go` declares:

```
type LiveKitService interface {
    CreateRoom(ctx, req) (*Room, error)
    DeleteRoom(ctx, name) error
    GenerateToken(roomName, identity, role, metadata) (string, error)
}
```

`internal/livekit/service.go` implements this interface (implicitly — Go structural typing). Tests in `internal/rooms` use a fake `LiveKitService` without importing `internal/livekit` at all. This is the seam for testing.

### 11.3 Error Propagation Across Package Boundaries

Each domain package defines its own sentinel errors:

```
internal/rooms/errors.go:   ErrRoomNotFound, ErrRoomEnded, ErrPlanExceeded, ErrMaxParticipants
internal/auth/errors.go:    ErrInvalidCredentials, ErrSessionExpired, ErrTokenBlacklisted
internal/recordings/errors.go: ErrEgressFailed, ErrRecordingNotFound
```

The handler layer (`internal/app` or per-package `handler.go`) maps these to HTTP status codes:

```
ErrRoomNotFound      → 404
ErrRoomEnded         → 409 Conflict
ErrPlanExceeded      → 403 Forbidden
ErrMaxParticipants   → 409 Conflict
ErrInvalidCredentials → 401
```

`errors.Is()` unwraps the sentinel through `fmt.Errorf("rooms.CreateRoom: %w", ErrPlanExceeded)` chains. The handler calls `errors.Is(err, rooms.ErrPlanExceeded)` — it does not parse error strings.

---

## 12. Build Sequence Rationale — Why This Order

The 4-day sequence is dependency-ordered:

**Day 1 (Foundation):** Module rename + pkg/ layer + config expansion + `internal/livekit`. Nothing else compiles without these. The LiveKit service is the dependency of rooms, recordings, participants, webhooks.

**Day 2 (Control plane):** Rooms CRUD + webhooks + participants + jobs queue. This gives you a working room lifecycle: create → join → events → end. The job queue is needed by Day 3.

**Day 3 (Data pipeline):** Recording → transcription → summary. These are sequential dependencies — egress produces the file, transcription consumes the file, summary consumes the transcript. Workers are registered into the job pool built on Day 2.

**Day 4 (Deployment):** Infrastructure setup last, not first. Build on local first, deploy once it works end-to-end. Deploying broken code to production is wasteful — local end-to-end validation (ngrok for webhooks) proves the system before touching production servers.

**What is deliberately excluded from the 4-day scope:**

- Billing (Stripe) — Week 2
- Email notifications — Week 2
- Live captions (LiveKit Agents) — Phase 3
- Analytics — Phase 3
- Frontend (separate stream, not blocked on backend)

# Airstage — Implementation Plan (Part 2): Deployment & Build Sequence

---

## 1. DigitalOcean Droplet Setup

### 1.1 Droplet Spec

```
Plan:     Basic — $48/mo (4 vCPU, 8GB RAM, 160GB SSD, 5TB transfer)
Region:   Choose closest to your users (NYC3 or BLR1)
OS:       Ubuntu 24.04 LTS
Auth:     SSH key only (no password)
```

### 1.2 Initial Server Setup

```bash
# Run as root after first SSH
adduser deploy
usermod -aG sudo deploy
rsync --archive --chown=deploy:deploy ~/.ssh /home/deploy
ufw allow 22/tcp
ufw allow 80/tcp
ufw allow 443/tcp
ufw enable

# Install Go
wget https://go.dev/dl/go1.22.4.linux-amd64.tar.gz
tar -C /usr/local -xzf go1.22.4.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile.d/go.sh

# Install Redis (self-hosted)
apt install -y redis-server
# /etc/redis/redis.conf:
#   bind 127.0.0.1
#   requirepass YOUR_REDIS_PASSWORD
#   maxmemory 1gb
#   maxmemory-policy allkeys-lru
systemctl enable redis-server

# Install Nginx
apt install -y nginx certbot python3-certbot-nginx
systemctl enable nginx
```

### 1.3 Nginx Config

```nginx
# /etc/nginx/sites-available/airstage
server {
    listen 80;
    server_name api.yourdomain.com;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name api.yourdomain.com;

    ssl_certificate     /etc/letsencrypt/live/api.yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/api.yourdomain.com/privkey.pem;
    ssl_protocols       TLSv1.2 TLSv1.3;
    ssl_prefer_server_ciphers on;

    # Security headers
    add_header X-Frame-Options DENY;
    add_header X-Content-Type-Options nosniff;
    add_header Strict-Transport-Security "max-age=31536000" always;

    # HTTP API
    location /api/ {
        proxy_pass         http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header   Host $host;
        proxy_set_header   X-Real-IP $remote_addr;
        proxy_set_header   X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_read_timeout 30s;
    }

    # WebSocket upgrade
    location /api/v1/ws {
        proxy_pass         http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header   Upgrade $http_upgrade;
        proxy_set_header   Connection "upgrade";
        proxy_set_header   Host $host;
        proxy_read_timeout 3600s;  # keep WS alive
    }

    # Webhook (LiveKit calls this — no rate limiting)
    location /webhooks/ {
        proxy_pass         http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header   Host $host;
        proxy_read_timeout 30s;
    }

    # Block debug endpoint from public
    location /debug/ { deny all; }

    # Frontend (served as static build)
    location / {
        root /var/www/airstage;
        try_files $uri $uri/ /index.html;
    }
}
```

```bash
# Get SSL cert
certbot --nginx -d api.yourdomain.com
ln -s /etc/nginx/sites-available/airstage /etc/nginx/sites-enabled/
nginx -t && systemctl reload nginx
```

---

## 2. Systemd Service Units

### 2.1 API Server

```ini
# /etc/systemd/system/airstage-api.service
[Unit]
Description=Airstage API Server
After=network.target redis.service
Requires=redis.service

[Service]
Type=simple
User=deploy
Group=deploy
WorkingDirectory=/opt/airstage
EnvironmentFile=/opt/airstage/.env
ExecStart=/opt/airstage/bin/airstage
Restart=always
RestartSec=5s
StandardOutput=journal
StandardError=journal

# Graceful shutdown: SIGTERM → 20s drain
TimeoutStopSec=25s
KillSignal=SIGTERM
KillMode=process

# Resource limits
LimitNOFILE=65536
LimitNPROC=4096

[Install]
WantedBy=multi-user.target
```

```bash
systemctl enable airstage-api
systemctl start airstage-api
journalctl -u airstage-api -f  # follow logs
```

### 2.2 Environment File

```bash
# /opt/airstage/.env  (chmod 600, owned by deploy)
APP_ENV=production
APP_PORT=8080
APP_DOMAIN=api.yourdomain.com
DEBUG_ADDR=127.0.0.1:6060

DATABASE_URL=postgresql://USER:PASS@HOST/airstage?sslmode=require
REDIS_URL=redis://:YOUR_REDIS_PASSWORD@127.0.0.1:6379/0

JWT_SECRET=<64-char random>
JWT_REFRESH_SECRET=<64-char random>

LIVEKIT_HOST=wss://your-project.livekit.cloud
LIVEKIT_API_KEY=APIxxxxxxxxxx
LIVEKIT_API_SECRET=xxxxxxxxxxxxxxxxxx

DEEPGRAM_API_KEY=xxxxxxxxxxxx
OPENAI_API_KEY=sk-xxxxxxxxxxxx

SPACES_ENDPOINT=https://nyc3.digitaloceanspaces.com
SPACES_BUCKET=airstage-recordings
SPACES_ACCESS_KEY=xxxxxxxxxxxx
SPACES_SECRET_KEY=xxxxxxxxxxxx

SENTRY_DSN=https://xxx@xxx.ingest.sentry.io/xxx
```

---

## 3. DigitalOcean Spaces Setup

```bash
# Create space via doctl or web console
doctl compute space create airstage-recordings --region nyc3

# Bucket policy: private (no public access)
# Generate access keys: DO console → API → Spaces Keys

# Folder structure inside bucket:
# recordings/{room_id}/{egress_id}/{timestamp}.mp4
# thumbnails/{room_id}/thumb.jpg
```

**Signed URL generation** (in `pkg/storage/spaces.go`):

```go
func (c *Client) PresignGetURL(ctx context.Context, key string, ttl time.Duration) (string, error) {
    presignClient := s3.NewPresignClient(c.s3)
    req, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
        Bucket: aws.String(c.bucket),
        Key:    aws.String(key),
    }, s3.WithPresignExpires(ttl))
    if err != nil {
        return "", fmt.Errorf("storage.Presign: %w", err)
    }
    return req.URL, nil
}
// TTL = 15min for recordings (per security spec)
// Re-presign on each authenticated GET /rooms/:id/recording
```

---

## 4. LiveKit Cloud Setup

### 4.1 Account & Project

```
1. Sign up at cloud.livekit.io
2. Create project: "airstage-prod"
3. Tier: Build (Free) — 5000 WebRTC min/mo, 50GB transfer
4. Copy: Project URL (wss://...), API Key, API Secret
5. Set in .env: LIVEKIT_HOST, LIVEKIT_API_KEY, LIVEKIT_API_SECRET
```

### 4.2 Webhook Configuration

```
LiveKit Console → Settings → Webhooks → Add Webhook:
  URL: https://api.yourdomain.com/webhooks/livekit
  Events (check all):
    ✓ room_started        ✓ room_finished
    ✓ participant_joined  ✓ participant_left
    ✓ participant_connection_aborted
    ✓ track_published     ✓ track_unpublished
    ✓ egress_started      ✓ egress_updated  ✓ egress_ended
```

**HMAC validation** (your webhook handler):

```go
// internal/webhooks/handler.go
import "github.com/livekit/protocol/auth"

func validateSignature(body []byte, authHeader, apiSecret string) error {
    v, err := auth.ParseAPIToken(authHeader)
    if err != nil {
        return fmt.Errorf("invalid auth header: %w", err)
    }
    claims, err := v.Verify(apiSecret)
    if err != nil {
        return fmt.Errorf("signature verify failed: %w", err)
    }
    // claims.SHA256 must match sha256(body)
    hash := sha256.Sum256(body)
    if hex.EncodeToString(hash[:]) != claims.SHA256 {
        return fmt.Errorf("body hash mismatch")
    }
    return nil
}
```

### 4.3 Egress S3 Integration

```
LiveKit Egress uses your Spaces credentials to upload directly.
You pass S3Config in StartRoomCompositeEgress (see Part 1 §5).
LiveKit Cloud manages the egress process — you only get the webhook when done.
No ffmpeg, no media handling on your server.
```

### 4.4 LiveKit Frontend Connection Flow

```
1. User authenticates → gets JWT access token from your API
2. Frontend calls: GET /api/v1/rooms/:id/token  (your backend)
3. Backend generates LiveKit VideoGrant JWT → returns to frontend
4. Frontend: const room = new Room(); await room.connect(LIVEKIT_HOST, livekitToken)
5. Frontend connects DIRECTLY to LiveKit Cloud — your backend is out of media path
6. LiveKit fires webhooks to your server as state changes
```

---

## 5. Database Migrations

**Drop inline DDL in `db/sql.db.go` — move to SQL files:**

```bash
# Install golang-migrate
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Migration files naming: NNNNNN_description.{up,down}.sql
migrations/
  000001_create_users.up.sql
  000001_create_users.down.sql
  000002_create_sessions.up.sql
  000003_create_rooms.up.sql
  000004_create_room_participants.up.sql
  000005_create_messages.up.sql
  000006_create_recordings.up.sql
  000007_create_transcripts.up.sql
  000008_create_summaries.up.sql
  000009_create_subscriptions.up.sql
  000010_create_job_queue.up.sql
  000011_create_webhook_events.up.sql   # idempotency store
```

```sql
-- 000011_create_webhook_events.up.sql
CREATE TABLE IF NOT EXISTS webhook_events (
    event_id    TEXT PRIMARY KEY,         -- LiveKit's unique event ID
    event_type  TEXT NOT NULL,
    payload     JSONB NOT NULL,
    received_at TIMESTAMPTZ DEFAULT NOW()
);
-- No TTL here — small table, prune with reconciler after 30 days
```

**Makefile targets:**

```makefile
migrate-up:
	migrate -path=migrations -database=$(DATABASE_URL) up

migrate-down:
	migrate -path=migrations -database=$(DATABASE_URL) down 1

migrate-version:
	migrate -path=migrations -database=$(DATABASE_URL) version

migrate-create:
	migrate create -ext sql -dir migrations -seq $(name)
```

---

## 6. CI/CD Pipeline (GitHub Actions)

```yaml
# .github/workflows/ci.yml
name: CI
on:
  pull_request:
    branches: [main, develop]

jobs:
  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16
        env:
          POSTGRES_DB: airstage_test
          POSTGRES_PASSWORD: test
        options: >-
          --health-cmd pg_isready --health-interval 10s
      redis:
        image: redis:7
        options: --health-cmd "redis-cli ping"

    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: "1.22" }
      - run: go vet ./...
      - run: go test -race -count=1 -timeout=120s ./...
        env:
          DATABASE_URL: postgres://postgres:test@localhost/airstage_test?sslmode=disable
          REDIS_URL: redis://localhost:6379/1
```

```yaml
# .github/workflows/deploy.yml
name: Deploy
on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: "1.22" }

      - name: Build binary
        run: |
          CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
            go build -ldflags="-s -w" -o bin/airstage ./cmd/api

      - name: Deploy to droplet
        uses: appleboy/scp-action@v0.1.7
        with:
          host: ${{ secrets.DROPLET_IP }}
          username: deploy
          key: ${{ secrets.SSH_PRIVATE_KEY }}
          source: bin/airstage
          target: /opt/airstage/bin/

      - name: Restart service
        uses: appleboy/ssh-action@v1.0.3
        with:
          host: ${{ secrets.DROPLET_IP }}
          username: deploy
          key: ${{ secrets.SSH_PRIVATE_KEY }}
          script: |
            sudo /opt/airstage/bin/migrate-up.sh
            sudo systemctl restart airstage-api
            sleep 3
            systemctl is-active airstage-api
```

**Required GitHub Secrets:**

```
DROPLET_IP          = your droplet IP
SSH_PRIVATE_KEY     = deploy user's private key
```

---

## 7. Local Dev Setup

```yaml
# deployments/docker-compose.dev.yml
version: "3.8"
services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: airstage_dev
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
    ports: ["5432:5432"]
    volumes: [postgres_data:/var/lib/postgresql/data]

  redis:
    image: redis:7-alpine
    ports: ["6379:6379"]
    command: redis-server --loglevel warning

volumes:
  postgres_data:
```

```toml
# .air.toml — live reload
[build]
  cmd = "go build -o ./tmp/airstage ./cmd/api"
  bin = "./tmp/airstage"
  full_bin = "./tmp/airstage -config config/dev.env"
  include_ext = ["go"]
  exclude_dir = ["tmp", "vendor", "migrations"]
  delay = 500
[log]
  time = true
```

---

## 8. Day-by-Day Build Sequence (3–4 Days)

### Day 1 — Foundation

```
AM:
  □ Rename module: recallo → airstage
  □ Migrate db/sql.db.go → pkg/database/ (use sqlx, remove inline DDL)
  □ Create migrations/000001 through 000011
  □ Add pkg/cache/redis.go + wire in cmd/api/main.go
  □ Add config fields: Redis, LiveKit, Deepgram, OpenAI, Spaces

PM:
  □ internal/livekit: service.go, token.go, egress.go
  □ internal/rooms: model.go, repository.go, service.go (CreateRoom, EndRoom)
  □ internal/rooms: handler.go (POST /rooms, GET /rooms/:id, DELETE /rooms/:id)
  □ internal/rooms: GET /rooms/:id/token
  □ Test: create room → token → verify in LiveKit console
```

### Day 2 — Webhooks + Participants + Jobs

```
AM:
  □ internal/webhooks: handler.go (HMAC validate), dispatcher.go
  □ Register POST /webhooks/livekit
  □ Test: use ngrok locally, verify LiveKit webhook delivery + signature
  □ internal/participants: model, repo, service, handler
  □ Routes: GET/DELETE/PATCH /rooms/:id/participants

PM:
  □ internal/jobs: queue.go, worker.go, retry.go, dlq.go
  □ Wire job pool in main.go
  □ Test: enqueue a dummy job, verify BRPOP + retry + DLQ
  □ PlanGate middleware + wire to recording/transcript routes
```

### Day 3 — Recording + Transcription + Summary

```
AM:
  □ internal/recordings: service.go (StartRecording calls livekit.StartEgress)
  □ Webhook side-effects: egress_started → recording, egress_ended → completed + enqueue
  □ pkg/storage/spaces.go: upload + presign
  □ Test: start recording, stop, verify S3 upload via egress_ended webhook

PM:
  □ internal/transcripts: deepgram.go + worker registered in job pool
  □ internal/summaries: openai.go + worker registered in job pool
  □ End-to-end test: room_finished → transcript job → summary job
  □ internal/reconciler: reconciler.go, wire in main.go
```

### Day 4 — Deployment

```
AM:
  □ Provision DO Droplet, install Redis, Nginx
  □ Configure Nginx (see §3 above)
  □ Set up SSL via certbot
  □ Create /opt/airstage/.env with all production values
  □ Build binary, deploy, systemctl start airstage-api
  □ Configure LiveKit webhook URL in cloud console

PM:
  □ Run database migrations on Neon (production)
  □ GitHub Actions secrets + CI/CD pipeline
  □ Smoke test: register → login → create room → get token → connect LiveKit
  □ Smoke test: start recording → stop → verify egress_ended webhook → transcript queued
  □ Monitor: journalctl -u airstage-api -f
```

---

## 9. Deployment Checklist

```
Infrastructure
  □ Droplet running, SSH key auth only
  □ UFW: 22/80/443 open, 6060 blocked (debug only 127.0.0.1)
  □ Redis bound to 127.0.0.1, password set
  □ Nginx serving API + WS + static frontend
  □ SSL cert active, auto-renew cron (certbot renew)
  □ Spaces bucket: private, CORS set for frontend domain

LiveKit
  □ API Key + Secret in .env
  □ Webhook URL registered in LiveKit console
  □ Webhook events all checked
  □ Test: create room via API → visible in LiveKit dashboard

Application
  □ All env vars in /opt/airstage/.env (chmod 600)
  □ DB migrations run (migrate up)
  □ systemd service enabled + running
  □ Logs flowing: journalctl -u airstage-api

Validation
  □ GET /health/ready → {"status":"ready"}
  □ POST /auth/register → 201
  □ POST /auth/login → 200 + tokens
  □ POST /rooms → 201 + livekit_room_name populated
  □ GET /rooms/:id/token → LiveKit JWT
  □ POST /webhooks/livekit → 200 (test with livekit-cli)
```

---

## 10. Key Decoupling Patterns Summary

| Concern                          | Pattern                               | Why                                                             |
| -------------------------------- | ------------------------------------- | --------------------------------------------------------------- |
| Webhook → pipeline               | Redis queue (not direct call)         | Webhook must return fast; processing is async                   |
| LiveKit SDK                      | Interface wrapper (`livekit.Service`) | Testable, swappable if moving off LiveKit Cloud                 |
| Job persistence                  | Postgres mirror of Redis queue        | Redis is ephemeral; reconciler re-queues from Postgres on crash |
| Recording URL                    | Presigned S3 URL (15min TTL)          | Bucket stays private; access is auditable per-request           |
| Hub broadcast                    | Local + Redis PubSub dual-write       | Single-instance now; multi-instance later with no code change   |
| Config                           | Value struct passed explicitly        | No global singletons; every component testable in isolation     |
| External APIs (Deepgram, OpenAI) | Thin wrapper structs with `ctx`       | Circuit breaker can be added without touching callers           |
