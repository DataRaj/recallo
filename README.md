# gotel-reserves

A hotel reservation API built with Go, using **chi** for routing and **PostgreSQL** as the primary database. Includes JWT-based authentication, a custom middleware stack, and auto-migrated schema on startup.

## Tech Stack

| Layer        | Technology                                         |
| ------------ | -------------------------------------------------- |
| Language     | Go 1.26                                            |
| HTTP Router  | `go-chi/chi` v5 (idiomatic `net/http`)             |
| Primary DB   | PostgreSQL via `database/sql` + `lib/pq` driver    |
| Auth         | JWT (`golang-jwt/jwt` v5) + bcrypt password hashing |
| Config       | `cleanenv` + `.env` file                           |

> **Note:** MongoDB integration (`db/collections/`) is still present in the codebase for historical reference but is no longer wired into the active request path. PostgreSQL is the sole operational database.

## Project Structure

```
.
├── cmd/
│   └── api/
│       └── main.go                  # Entrypoint — config, DB init, middleware, graceful shutdown
├── db/
│   ├── db.go                        # Shared DB utilities
│   ├── sql.db.go                    # PostgreSQL pool init, schema migration, indexes
│   └── collections/
│       └── user.go                  # MongoUserStore (legacy, not active)
├── internals/
│   ├── configs/
│   │   └── config.go                # Config struct + LoadConfig() via cleanenv
│   ├── handlers/
│   │   └── login.go                 # POST /api/v1/auth/login handler
│   ├── middleware/
│   │   ├── authentication.go        # JWT AuthenticateMiddleware (Bearer token validation)
│   │   ├── cors.go                  # CORS middleware
│   │   └── muxlogger.go             # Request logger middleware
│   ├── models/
│   │   └── user.go                  # User struct + GetUserByEmail / CreateUser (PostgreSQL)
│   ├── routes/
│   │   ├── routes.go                # Route registration (ServeMux)
│   │   ├── healthcheck.go           # GET /api/v1/healthcheck handler
│   │   └── user-register.go         # POST /api/v1/auth/register handler
│   ├── utils/
│   │   ├── api-response.go          # JSON response helper
│   │   ├── hash.go                  # bcrypt password hashing
│   │   ├── jwt.go                   # JWT generation & verification
│   │   └── refresh-token.go         # Refresh token utilities
│   ├── lib/
│   │   └── pq.driver.go             # Low-level pq error handling helpers
│   └── realtime/                    # (reserved for WebSocket / SSE work)
├── types/
│   └── user.go                      # Legacy domain model (MongoDB BSON, not active)
├── postgresql/
│   └── dev/                         # Local PostgreSQL data directory
├── config/
│   └── config.env                   # Environment variables (git-ignored)
├── env.example                      # Environment variable template
├── makefile                         # build / run / test shortcuts
├── go.mod
└── go.sum
```

## API

### Public Endpoints

| Method | Endpoint                  | Description              |
| ------ | ------------------------- | ------------------------ |
| GET    | `/api/v1/healthcheck`     | Server health check      |
| POST   | `/api/v1/auth/register`   | Register a new user      |
| POST   | `/api/v1/auth/login`      | Login and obtain tokens  |

### Protected Endpoints *(require `Authorization: Bearer <token>` header)*

> Protected routes are gated by `AuthenticateMiddleware`. No protected routes are registered yet — the middleware is wired and ready.

### Request — Register User

```http
POST /api/v1/auth/register
Content-Type: application/json
```
```json
{
  "name": "John Doe",
  "email": "john@example.com",
  "password": "s3cr3t"
}
```

### Request — Login

Requires the `X-Platform` header set to `web` or `mobile`.

```http
POST /api/v1/auth/login
Content-Type: application/json
X-Platform: web
```
```json
{
  "email": "john@example.com",
  "password": "s3cr3t"
}
```

### Response Format

**Register success:**
```json
{
  "success": true,
  "message": "User has been created successfully",
  "data": null
}
```

**Login success:**
```json
{
  "success": true,
  "message": "login successful",
  "data": {
    "user": { "id": 1, "name": "John Doe", "email": "john@example.com" },
    "access_token": "<jwt>",
    "refresh_token": "<opaque-token>"
  }
}
```

**Error (any endpoint):**
```json
{
  "success": false,
  "message": "error description",
  "data": null
}
```

Unmatched routes return a `404` with the error envelope above.

## Architecture

The project follows a clean layered architecture:

- **`internals/configs`** — Centralised config loading via `cleanenv`. Reads from a `.env` file resolved by `-config` flag → `CONFIG_PATH` env var → `config/dev.env` fallback.
- **`db/sql.db.go`** — PostgreSQL connection pool with tunable limits. On `InitDB()`, automatically creates and migrates all tables (`users`, `privates`, `messages`) and their indexes — no separate migration tool required.
- **`internals/models`** — Data access layer backed by PostgreSQL. `GetUserByEmail` and `CreateUser` query the global `db.DB` pool directly.
- **`internals/routes`** — HTTP handlers registered on a `net/http.ServeMux`. Handlers call model functions and use `utils.JSON` for uniform responses.
- **`internals/middleware`** — Reusable middleware chain:
  - `AuthenticateMiddleware` — validates `Bearer` JWT tokens, extracts `userId`, `name`, and `X-Platform` claims into request context.
  - `cors.go` — cross-origin request handling.
  - `muxlogger.go` — structured request logging.
- **`internals/utils`** — Utility functions: uniform JSON responses, bcrypt hashing, JWT sign/verify, refresh token helpers.
- **`cmd/api/main.go`** — Application entrypoint. Loads config, initialises the PostgreSQL pool, wraps the `ServeMux` from `internals/routes` with logging middleware, and runs the HTTP server with graceful shutdown.
- **`internals/handlers`** — Feature-level HTTP handlers living outside the routes package. `login.go` validates credentials, issues a JWT access token and an opaque refresh token, and persists the refresh token to the database.

## Database Schema

Tables are auto-created at startup by `db.InitDB()`.

### `users`
| Column                      | Type      | Notes                  |
| --------------------------- | --------- | ---------------------- |
| `id`                        | INTEGER   | PK, identity           |
| `name`                      | TEXT      | NOT NULL               |
| `email`                     | TEXT      | NOT NULL, UNIQUE       |
| `password`                  | TEXT      | bcrypt hash, NOT NULL  |
| `refresh_token_web`         | TEXT      |                        |
| `refresh_token_web_at`      | TIMESTAMP |                        |
| `refresh_token_mobile`      | TEXT      |                        |
| `refresh_token_mobile_at`   | TIMESTAMP |                        |
| `created_at`                | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP |

### `privates` (private conversations)
| Column       | Type      | Notes                            |
| ------------ | --------- | -------------------------------- |
| `id`         | INTEGER   | PK, identity                     |
| `user1_id`   | INTEGER   | FK → users, user1_id < user2_id  |
| `user2_id`   | INTEGER   | FK → users                       |
| `created_at` | TIMESTAMP |                                  |

### `messages`
| Column         | Type      | Notes                              |
| -------------- | --------- | ---------------------------------- |
| `id`           | INTEGER   | PK, identity                       |
| `from_id`      | INTEGER   | FK → users                         |
| `private_id`   | INTEGER   | FK → privates                      |
| `message_type` | TEXT      | NOT NULL                           |
| `content`      | TEXT      | NOT NULL                           |
| `delivered`    | INTEGER   | 0/1 flag, DEFAULT 0                |
| `read`         | INTEGER   | 0/1 flag, DEFAULT 0                |
| `created_at`   | TIMESTAMP |                                    |

## Authentication

JWT tokens are signed with `HS256` and expire after **30 minutes**.

Token claims include:
- `user_id` — database user ID
- `name` — display name
- `X-Platform` — `"web"` or `"mobile"`

The `X-Platform` header (or token claim) must match on every protected request. Mismatches are rejected with `401 Unauthorized`.

## Configuration

Config is loaded by `internals/configs.LoadConfig()` using `cleanenv`.

| Environment Variable | Default                                                         | Description                        |
| -------------------- | --------------------------------------------------------------- | ---------------------------------- |
| `ENV`                | `dev`                                                           | Runtime environment                |
| `HTTP_ADDRESS`       | `0.0.0.0:8080`                                                  | Host and port the server binds to  |
| `DATABASE_URL`       | `postgres://postgres:password@localhost:5432/gotel?sslmode=disable` | PostgreSQL connection DSN      |
| `JWT_SECRET_KEY`     | *(required, no default)*                                        | Secret key for JWT signing         |

Copy `env.example` to `config/dev.env` and fill in your values:

```bash
cp env.example config/dev.env
```

Config file resolution order:
1. `-config <path>` CLI flag
2. `CONFIG_PATH` environment variable
3. `config/dev.env` (default fallback)

## Chi Middleware

The following middleware is applied globally on every request via the Chi router:

- `middleware.RequestID` — attaches a unique request ID to every request
- `middleware.Logger` — structured request logging
- `middleware.Recoverer` — recovers from panics and returns a `500`

## Prerequisites

- Go 1.26+
- PostgreSQL running and accessible via `DATABASE_URL`

## Usage

```bash
# copy and configure environment
cp env.example config/dev.env

# build binary → bin/api
make build

# run the built binary (reads config/dev.env by default)
make run

# run with a custom config file
./bin/api -config config/config.env

# run with a custom listen address
./bin/api -listenAddr :9000

# run all tests
make test
```
