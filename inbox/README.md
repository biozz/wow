# Inbox

Simple message queue server with atomic semantics. Producers POST messages, consumers GET them once in FIFO order.

## Tech Stack

- **Go** - stdlib HTTP server
- **Bun ORM** - database operations  
- **SQLite** - embedded storage with WAL mode
- **Token auth** - simple bearer token protection

## Features

- Atomic fetch-and-archive (at-most-once delivery)
- FIFO ordering by timestamp + ID
- REST API with health checks
- Single binary deployment

See [DESIGN.md](DESIGN.md) for detailed architecture and API docs.

Use `make run` to start with example config.
