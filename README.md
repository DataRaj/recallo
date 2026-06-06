# gotel-reserves

A hotel reservation API built with Go, using **chi** for routing, **MongoDB** as the primary database, and a **PostgreSQL** layer scaffolded for future expansion.

## Tech Stack

| Layer        | Technology                                  |
| ------------ | ------------------------------------------- |
| Language     | Go 1.26                                     |
| HTTP Router  | `go-chi/chi` v5 (idiomatic `net/http`)      |
| Primary DB   | MongoDB v2 Driver                           |
| Secondary DB | PostgreSQL (scaffolded via `database/sql`)  |
| Config       | `cleanenv` + `.env` file                    |

## Project Structure

```
.
├── cmd/
│   └── api/
│       └── main.go              # entrypoint — server bootstrap & graceful shutdown
├── server.go                    # Chi router setup, middleware, route registration
├── handlers/
│   └── api_handlers.go          # HTTP handlers (User)
├── db/
│   ├── db.go                    # shared DB utilities (ObjectID helper)
│   ├── sql.db.go                # PostgreSQL connection pool setup (scaffolded)
│   └── collections/
│       └── user.go              # UserStore interface + MongoUserStore implementation
├── internals/
│   └── configs/
│       └── config.go            # Config struct + LoadConfig() via cleanenv
├── types/
│   └── user.go                  # Domain models (User)
├── config/
│   └── config.env               # Environment variables file
├── makefile                     # build / run / test shortcuts
├── go.mod
└── go.sum
```

## API

All endpoints are prefixed with `/api/v1`.

| Method | Endpoint        | Description      |
| ------ | --------------- | ---------------- |
| GET    | `/api/v1/users`     | List all users   |
| GET    | `/api/v1/users/{id}` | Get user by ID   |

### Response Format

**Success** — returns JSON body directly.

**Error** — returns a structured error envelope:

```json
{
  "success": false,
  "message": "error description"
}
```

Unmatched routes return a `404` with the same envelope.

## Architecture

The project follows a clean layered architecture:

- **`types`** — Plain domain structs with BSON/JSON tags. No logic, no dependencies.
- **`db/collections`** — Data access layer. Defines the `UserStore` interface and its concrete `MongoUserStore` implementation. All MongoDB queries live here.
- **`handlers`** — HTTP handlers. Receive a `UserStore` via constructor injection; no direct DB access.
- **`server.go`** — Wires everything together: connects to MongoDB, initializes stores & handlers, registers Chi routes with middleware.
- **`cmd/api/main.go`** — Application entrypoint. Loads config, starts the HTTP server in a goroutine, and handles graceful shutdown on `SIGINT`/`SIGTERM` with a 10 s drain timeout.
- **`internals/configs`** — Centralised config loading. Reads from a `.env` file (path resolved via `-config` flag → `CONFIG_PATH` env var → `config/dev.env` fallback) using `cleanenv`.
- **`db/sql.db.go`** — PostgreSQL connection pool scaffolding (connection limits, lifetime, WAL pragmas). Wired but not yet integrated into request handlers.

Handlers depend on the `UserStore` **interface**, not the concrete Mongo type — swapping storage backends or writing unit tests requires no handler changes.

## Configuration

Config is loaded by `internals/configs.LoadConfig()` using the `cleanenv` library.

| Environment Variable | Default                  | Description                        |
| -------------------- | ------------------------ | ---------------------------------- |
| `ENV`                | `dev`                    | Runtime environment                |
| `HTTP_ADDRESS`       | `192.168.0.102:8080`     | Host and port the server binds to  |
| `DB_PATH`            | `postgresql/dev`         | Path for the PostgreSQL data dir   |
| `DB_NAME`            | `db.dev`                 | PostgreSQL database file name      |
| `JWT_SECRET_KEY`     | `sha25612864321684210`   | Secret key for JWT signing         |

Config file resolution order:
1. `-config <path>` CLI flag
2. `CONFIG_PATH` environment variable
3. `config/dev.env` (default fallback)

MongoDB URI is currently hardcoded in `server.go` to `mongodb://localhost:27017`, database `gotel-reservation`.

## Chi Middleware

The following middleware is applied globally on every request:

- `middleware.RequestID` — attaches a unique request ID to every request
- `middleware.Logger` — structured request logging
- `middleware.Recoverer` — recovers from panics and returns a `500`

## Prerequisites

- Go 1.26+
- MongoDB running on `localhost:27017`

## Usage

```bash
# build binary → bin/api
make build

# run the built binary
make run

# run with a custom config file
./bin/api -config config/config.env

# run all tests
make test
```

