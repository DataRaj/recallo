# Gotel Reserves

A hotel reservation API built with Go, Fiber, and MongoDB.

## Tech Stack

- **Go 1.26** — language runtime
- **Fiber v3** — HTTP framework
- **MongoDB v2 Driver** — database layer

## Project Structure

```
.
├── server.go              # entrypoint, routes, Fiber config
├── handlers/
│   └── api_handlers.go    # request handlers (User)
├── db/
│   ├── db.go              # shared DB utilities
│   └── collections/
│       └── user.go        # UserStore interface + Mongo implementation
├── types/
│   └── user.go            # domain models
├── makefile               # build/run/test shortcuts
├── go.mod
└── go.sum
```

## API

All endpoints are prefixed with `/api/v1`.

| Method | Endpoint      | Description          |
| ------ | ------------- | -------------------- |
| GET    | `/users`      | List all users       |
| GET    | `/user/:id`   | Get a user by ID     |

### Response Format

**Success** — returns JSON body directly.

**Error** — returns a structured error:

```json
{
  "success": false,
  "message": "error description"
}
```

## Architecture

The codebase follows a clean separation of concerns:

- **`types`** — Plain domain structs with BSON/JSON tags. No logic, no dependencies.
- **`db/collections`** — Data access layer. Defines a `UserStore` interface and its MongoDB implementation (`MongoUserStore`). All database queries live here.
- **`handlers`** — HTTP handlers. Accept a store interface via constructor injection. No direct database access.
- **`server.go`** — Wires everything together: connects to Mongo, initializes stores and handlers, registers routes, starts the server.

Handlers depend on the `UserStore` interface, not the concrete Mongo implementation. This makes testing and swapping storage backends straightforward.

## Prerequisites

- Go 1.26+
- MongoDB running on `localhost:27017`

## Usage

```bash
# build
make build

# run (default :5000)
make run

# run on a custom port
./bin/api -listenAddr :8080

# test
make test
```

## Configuration

| Flag           | Default  | Description              |
| -------------- | -------- | ------------------------ |
| `-listenAddr`  | `:5000`  | Address the server binds to |

MongoDB connection URI is currently hardcoded to `mongodb://localhost:27017`. Database name is `gotel-reservation`.
