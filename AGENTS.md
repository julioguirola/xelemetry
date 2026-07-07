# Xelemetry — Agent Guide

Go API for tracking office power availability (electrical outages in Cuba). Clients poll or use WebSocket to register when power is on.

## Stack

- **Go 1.26** — `go.mod` module `xelemetry`
- **Echo v5** — uses `*echo.Context` (pointer), unlike v4
- **SQLite** via GORM (generic helper `gorm.G[T]`)
- **zerolog** (with `ConsoleWriter{Out: os.Stderr}`) bridged via `slog` for Echo's logger
- **gorilla/websocket** for WS endpoint
- **go-playground/validator/v10** for request validation

## Entrypoints

| Binary | Path | Purpose |
|--------|------|---------|
| API server | `cmd/api/main.go` | HTTP + WebSocket server |
| Migration | `cmd/migration/main.go` | Drops tables + AutoMigrate (destructive) |

All handler logic is inline in `cmd/api/main.go` (no router/controller split). Models are in `internal/models.go`.

## Dev commands

```sh
# 1. Migration must run first (destroys and recreates tables)
go run cmd/migration/main.go

# 2. Start API
go run cmd/api/main.go
```

`PORT` env var is required. Default: `1323` (from `.env`).

## Docker deploy

```sh
docker compose up -d --build
docker compose exec api ./migration
```

`PORT` env var is passed from `.env` via `docker-compose.yml`.

## Key facts

- SQLite DB file: `checks.db` (gitignored)
- Migration calls `DropTable` then `AutoMigrate` — **always destructive**
- `GET /check` and `GET /uptime` default `limit=40`, range 1–100; filter via `from`, `to`, `location_id`
- `GET /ws` tracks connection duration server-side (reads messages but ignores content); uptime saved on disconnect
- No tests, no CI, no formatter/linter config, no Makefile
