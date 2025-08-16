## Queue-like Message Server (Go + Bun + SQLite)

### Goals
- **Simple queue semantics**: producers POST text messages; consumers GET a sorted list; everything returned by GET is atomically archived.
- **States**: `new` → `archived` (one-way).
- **Deterministic ordering**: by `created_at ASC, id ASC`.
- **Minimal stack**: Go stdlib `net/http`, Bun ORM, SQLite; no frameworks.

### Non-goals
- Exactly-once delivery. This design implements at-most-once (on GET, messages are archived before client acks).
- Multi-node clustering. Single-node SQLite; can be supervised externally.

## API

- All endpoints require `Authorization: Bearer <token>`.

- **POST /v1/messages**
  - Request JSON: `{ "body": string }`
  - Returns 201 and JSON: `{ "id": number, "timestamp": string, "body": string }`

- **GET /v1/messages?limit=1**
  - Atomically moves up to `limit` messages from `new` → `archived` and returns them sorted.
  - Response: array of `{ id, timestamp, body }`.
  - Default `limit`: 1. Clients are expected to process messages one-by-one; higher limits may be unnecessary.

- **GET /health** → 200 if DB reachable.

  

Notes:
- GET is an atomic claim-and-archive. If client loses the response, those messages are gone (at-most-once). For at-least-once, see Extensions.

## Data Model

```sql
-- migrations/0001_init.sql
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS messages (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  body         TEXT NOT NULL,
  state        TEXT NOT NULL CHECK (state IN ('new','archived')) DEFAULT 'new',
  created_at   DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  archived_at  DATETIME
);

CREATE INDEX IF NOT EXISTS idx_messages_state_created ON messages(state, created_at, id);
```

Representation exposed to clients:
- `timestamp` = `created_at` ISO-8601 in UTC.

## Concurrency & Transaction Semantics

- Atomic GET uses a single `UPDATE ... WHERE id IN (SELECT ...) RETURNING` inside one transaction.
- Ordering is guaranteed by `ORDER BY created_at ASC, id ASC` in the picking subquery.

### Atomic GET SQL (SQLite ≥ 3.35)

```sql
WITH picked AS (
  SELECT id
  FROM messages
  WHERE state = 'new'
  ORDER BY created_at ASC, id ASC
  LIMIT ?1
)
UPDATE messages
SET state = 'archived', archived_at = (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
WHERE id IN (SELECT id FROM picked)
RETURNING id, created_at AS timestamp, text;
```

## Go Implementation Notes

- HTTP: stdlib `net/http` with `http.ServeMux`.
- DB: Bun with SQLite driver (`modernc.org/sqlite` or `github.com/mattn/go-sqlite3`; the former is pure Go, recommended for easy builds).
- Migrations: plain SQL files executed on startup via Bun or a tiny runner.

### Types

```go
type Message struct {
    ID         int64     `bun:",pk,autoincrement" json:"id"`
    Body       string    `bun:",notnull" json:"body"`
    State      string    `bun:",notnull" json:"-"`
    CreatedAt  time.Time `bun:"created_at,nullzero,notnull,default:CURRENT_TIMESTAMP" json:"timestamp"`
    ArchivedAt time.Time `bun:"archived_at,nullzero" json:"-"`
}
```

### Insert (POST /v1/messages)

```go
func handlePostMessage(w http.ResponseWriter, r *http.Request) {
    // require bearer token auth
    // read small JSON body {body}
    // insert
}
```

### Atomic fetch+archive (GET /v1/messages)

Prefer raw SQL with Bun to leverage RETURNING reliably on SQLite:

```go
func fetchAndArchive(ctx context.Context, db *bun.DB, limit int) ([]Message, error) {
    var out []Message
    err := db.RunInTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable}, func(ctx context.Context, tx bun.Tx) error {
        return tx.NewRaw(`
            WITH picked AS (
              SELECT id FROM messages
              WHERE state = 'new'
              ORDER BY created_at ASC, id ASC
              LIMIT ?
            )
            UPDATE messages
            SET state = 'archived', archived_at = (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
            WHERE id IN (SELECT id FROM picked)
            RETURNING id, created_at AS timestamp, text
        `, limit).Scan(ctx, &out)
    })
    return out, err
}
```

### Routing (no framework)

```go
mux := http.NewServeMux()
mux.HandleFunc("/v1/messages", func(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case http.MethodPost:
        handlePostMessage(w, r)
    case http.MethodGet:
        handleGetMessages(w, r)
    default:
        w.WriteHeader(http.StatusMethodNotAllowed)
    }
})
mux.HandleFunc("/health", handleHealth)
```

## Config

- `LISTEN_ADDR` (default `:8080`)
- `DB_PATH` (default `./queue.db`)
- `AUTH_TOKEN` (required; server rejects requests without `Authorization: Bearer <token>`)
- `GET_LIMIT_DEFAULT` (default 1)

## Security

- Required static bearer token for all endpoints.
- CORS disabled by default; enable only if needed.
- Run behind TLS-terminating reverse proxy or enable TLS in Go if necessary.

## Observability

- Structured logs (JSON) with request id, method, path, duration, status, row counts.

## Operational Notes

- Backups: copy `queue.db` and `queue.db-wal` while process running or run `.backup`.

## Testing

- Unit tests for:
  - Default GET limit of 1 and ordering
  - Large backlog pagination boundaries
- Property test: monotonic id/order; no re-delivery after archive

## Extensions (optional)

- **Ack mode**: add `visibility_timeout` + `claimed` state; add `POST /v1/acks` to transition `claimed`→`archived`.
- **Topics**: add `topic` column + per-topic retrieval.
- **Retention**: `DELETE FROM messages WHERE state='archived' AND archived_at < ?` via cron.

## Minimal Project Layout

```
cmd/server/main.go
internal/http/handlers.go
internal/storage/sqlite.go
internal/model/message.go
migrations/0001_init.sql
```

## Example cURL

```bash
curl -s -X POST localhost:8080/v1/messages \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer REPLACE_ME' \
  -d '{"body":"hello"}'

curl -s 'localhost:8080/v1/messages' \
  -H 'Authorization: Bearer REPLACE_ME'
```


## Future Ideas

- SQLite performance tuning and concurrency options:
  - `journal_mode=WAL`, `synchronous=NORMAL`, `busy_timeout=5000`
  - Use `BEGIN IMMEDIATE` for contested writers
  - Additional indexes or partial indexes as data grows
  - Periodic `VACUUM` to reclaim space
- Observability extras:
  - `/metrics` (Prometheus) for counts and latencies
  - Admin `/v1/stats` endpoint
- Higher batch retrieval limits and configurable max
- Idempotent POST via `dedupe_key` with `UNIQUE` and `ON CONFLICT DO NOTHING`
- Throughput/latency targets and benchmarks

