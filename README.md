# Recallo

> **Real-time private messaging backend** built in Go — clean architecture, WebSocket hub, JWT auth, and file sharing over a Postgres-backed REST API.

---

## What is Recallo?

Recallo is a backend platform for real-time 1-toSt-1 private chat. Users can register, authenticate, open private conversations, send messages (via WebSocket), share files, and have offline delivery automatically reconciled when they reconnect — all without polling.

---

## Tech Stack

| Layer       | Technology                                            |
| ----------- | ----------------------------------------------------- |
| Language    | Go 1.25                                               |
| HTTP Router | `net/http` stdlib ServeMux (Go 1.22+ pattern routing) |
| Database    | Postgres via `database/sql` + `postgresql`        |
| Auth        | JWT (`golang-jwt/jwt` v5) + bcrypt password hashing   |
| WebSocket   | `coder/websocket`                                     |
| Config      | `cleanenv` + `.env` file                              |

---

## Project Structure

```
.
├── cmd/
│   └── api/
│       └── main.go              # Entrypoint — config, DB init, graceful shutdown
├── config/
│   └── config.env               # Environment variables (git-ignored)
├── db/
│   ├── db.go                    # GetDB() singleton accessor
│   └── sql.db.go                # Postgres pool init + auto-migration
├── internals/
│   ├── configs/
│   │   └── config.go            # Config struct + LoadConfig() via cleanenv
│   ├── handlers/
│   │   ├── login.go             # POST /auth/login — JWT + refresh token issuance
│   │   ├── user-session.go      # POST /auth/refresh-session, GET /auth/current-user
│   │   ├── user.go              # GET /users/{id}
│   │   ├── convo.go             # Private conversations + paginated messages
│   │   └── files.go             # Multipart file upload + static file serving
│   ├── middleware/
│   │   ├── authentication.go    # JWT Bearer token validation, context injection
│   │   ├── cors.go              # CORS headers
│   │   └── muxlogger.go         # Request logger
│   ├── models/
│   │   ├── user.go              # User CRUD (GetByEmail, Create, GetByID)
│   │   ├── private.go           # Private room CRUD (GetByUsers, Create, GetForUser)
│   │   └── message.go           # Message CRUD + batch delivery update
│   ├── realtime/
│   │   ├── hub.go               # WebSocket hub — online map, broadcast, connect/disconnect
│   │   ├── client.go            # Client struct — conn, send channel, safe close
│   │   └── event.go             # EventType constants + Event struct
│   ├── routes/
│   │   ├── routes.go            # All route registrations
│   │   └── healthcheck.go       # GET /healthcheck
│   ├── utils/
│   │   ├── api-response.go      # Uniform JSON envelope helper
│   │   ├── hash.go              # bcrypt hash + compare
│   │   ├── jwt.go               # JWT generate + verify
│   │   └── refresh-token.go     # Opaque refresh token generate + DB lookup
│   └── logger/
│       └── logger.go            # App-level logger instance
├── env.example                  # Environment variable template
├── Makefile                     # build / run / test shortcuts
├── rest.http                    # HTTP client test file (all routes)
├── go.mod
└── go.sum
```

---

## API Reference

### Public Endpoints

| Method | Endpoint                       | Description                   |
| ------ | ------------------------------ | ----------------------------- |
| GET    | `/api/v1/healthcheck`          | Server health check           |
| POST   | `/api/v1/auth/register`        | Register a new user           |
| POST   | `/api/v1/auth/login`           | Login — returns JWT + refresh |
| POST   | `/api/v1/auth/refresh-session` | Rotate refresh token          |

### Protected Endpoints _(require `Authorization: Bearer <token>` + `X-Platform: web|mobile`)_

| Method | Endpoint                                | Description                                 |
| ------ | --------------------------------------- | ------------------------------------------- |
| POST   | `/api/v1/auth/logout`                   | Invalidate refresh token                    |
| GET    | `/api/v1/auth/current-user`             | Get authenticated user profile              |
| GET    | `/api/v1/users/{id}`                    | Get user by ID                              |
| POST   | `/api/v1/private/join`                  | Open or create a private conversation       |
| GET    | `/api/v1/private/conversations`         | List all conversations for current user     |
| GET    | `/api/v1/private/{private_id}`          | Get a specific private room                 |
| GET    | `/api/v1/private/{private_id}/messages` | Paginated message history (`page`, `limit`) |
| POST   | `/api/v1/files/{private_id}`            | Upload file to a conversation               |
| GET    | `/api/v1/files/*`                       | Serve uploaded files (static)               |

### WebSocket

| Endpoint        | Description                                        |
| --------------- | -------------------------------------------------- |
| `WS /api/v1/ws` | Persistent connection — real-time events           |

---

## Request / Response Examples

### Register

```http
POST /api/v1/auth/register
Content-Type: application/json

{ "name": "Dattaraj", "email": "datta@example.com", "password": "secret123" }
```

### Login

```http
POST /api/v1/auth/login
Content-Type: application/json
X-Platform: web

{ "email": "datta@example.com", "password": "secret123" }
```

```json
{
  "success": true,
  "message": "login successful",
  "data": {
    "user": { "id": 1, "name": "Dattaraj", "email": "datta@example.com" },
    "access_token": "<jwt>",
    "refresh_token": "<opaque>"
  }
}
```

### Error envelope (all endpoints)

```json
{ "success": false, "message": "error description", "data": null }
```

---

## Database Schema

Tables are auto-created at startup by `db.InitDB()` — no migration tool needed.

### `users`

| Column                    | Type      | Notes                     |
| ------------------------- | --------- | ------------------------- |
| `id`                      | INTEGER   | PK, autoincrement         |
| `name`                    | TEXT      | NOT NULL                  |
| `email`                   | TEXT      | NOT NULL, UNIQUE          |
| `password`                | TEXT      | bcrypt hash, NOT NULL     |
| `refresh_token_web`       | TEXT      |                           |
| `refresh_token_web_at`    | TIMESTAMP |                           |
| `refresh_token_mobile`    | TEXT      |                           |
| `refresh_token_mobile_at` | TIMESTAMP |                           |
| `created_at`              | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP |

### `privates`

| Column       | Type      | Notes                              |
| ------------ | --------- | ---------------------------------- |
| `id`         | INTEGER   | PK, autoincrement                  |
| `user1_id`   | INTEGER   | FK → users; always `user1 < user2` |
| `user2_id`   | INTEGER   | FK → users                         |
| `created_at` | TIMESTAMP |                                    |

### `messages`

| Column         | Type      | Notes                   |
| -------------- | --------- | ----------------------- |
| `id`           | INTEGER   | PK, autoincrement       |
| `from_id`      | INTEGER   | FK → users              |
| `private_id`   | INTEGER   | FK → privates           |
| `message_type` | TEXT      | `text`, `file`, etc.    |
| `content`      | TEXT      | Message body / file URL |
| `delivered`    | INTEGER   | 0/1, DEFAULT 0          |
| `read`         | INTEGER   | 0/1, DEFAULT 0          |
| `created_at`   | TIMESTAMP |                         |

---

## Authentication

- JWT tokens signed with **HS256**, expire in **30 minutes**.
- Claims: `user_id`, `name`, `X-Platform`.
- `X-Platform` must match on every protected request (`web` or `mobile`).
- Refresh tokens are opaque, stored per-platform in the `users` table, rotated on every `/refresh-session` call.

---

## Delivery & Read Status

Recallo tracks two message states — same as WhatsApp's ✓ / ✓✓ / 🔵🔵 model:

| State       | Column      | When it's set                                                |
| ----------- | ----------- | ------------------------------------------------------------ |
| `delivered` | `delivered` | User connects via WebSocket — one batch SQL UPDATE fires     |
| `read`      | `read`      | Client explicitly marks messages read (via REST or WS event) |

On reconnect, `MarkAllIncomingMessagesAsDelivered(userID)` fires a **single atomic SQL UPDATE** covering all unread messages across all private rooms — no N+1 loops, no WebSocket event fanout to the sender.

---

## Real-time Hub

The `Hub` (in `internals/realtime/hub.go`) is the central registry of live WebSocket connections:

- **`Clients map[int64]map[*Client]struct{}`** — the online/offline source of truth. If a user ID exists in this map, they're online.
- **`RegisterClientConnection`** — adds a client, broadcasts `EventUserOnline`, and fires the delivery batch update.
- **`UnregisterClientConnection`** — removes the client, broadcasts `EventUserOffline`.
- **`BroadcastToAll`** — fan-out to every connected client.
- **`SendEventToUserIds`** — targeted push to specific users (skips the sender).

### WebSocket Event Types

| Event          | Direction       | Meaning                         |
| -------------- | --------------- | ------------------------------- |
| `online`       | Server → All    | A user connected                |
| `offline`      | Server → All    | A user disconnected             |
| `message`      | Server → User   | New incoming message            |
| `delivered`    | Server → Sender | Message confirmed delivered     |
| `read`         | Server → Sender | Message confirmed read          |
| `typing`       | Server → User   | Typing indicator                |
| `new_private`  | Server → User   | New private room opened         |
| `current_user` | Server → User   | Current online user list        |
| `error`        | Server → Client | Error processing request        |
| `heartbeat`    | Server ↔ Client | Keep-alive ping                 |
| `shutdown`     | Server → All    | Graceful server shutdown signal |

---

## File Uploads

Files are stored locally under `./files/chats/{private_id}/{sender_id}/{filename}`.
The upload endpoint returns a `file_url` string which can be stored as the message `content`.
Files are served via a static file server at `/api/v1/files/*`.

---

## Configuration

| Variable         | Default        | Description         |
| ---------------- | -------------- | ------------------- |
| `ENV`            | `dev`          | Runtime environment |
| `HTTP_ADDRESS`   | `0.0.0.0:8080` | Server bind address |
| `DATABASE_URL`   | `postgres://...`| Postgres conn str   |
| `JWT_SECRET_KEY` | _(required)_   | JWT signing secret  |

```bash
cp env.example config/dev.env
# fill in JWT_SECRET_KEY at minimum
```

Config resolution order: `-config` flag → `CONFIG_PATH` env → `config/dev.env` fallback.

---

## Usage

```bash
# Build binary → bin/recallo
make build

# Run (reads config/dev.env by default)
make run

# Run with custom config
./bin/recallo -config config/config.env

# Run tests
make test
```

---

## Prerequisites

- Go 1.25+
- PostgreSQL database running (tables are auto-created at startup, no migrations needed).
