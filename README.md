# Foostash

An open-source, developer-first secrets and environment manager. Replace `.env` files with a centralized, encrypted store you interact with via CLI and SDKs.

## Quickstart (self-hosted with Docker)

```bash
export FOOSTASH_MASTER_KEY=$(openssl rand -base64 32)
export FOOSTASH_JWT_SECRET=$(openssl rand -base64 32)

docker compose up -d --build
docker compose exec foostash foostash migrate up
```

The API is now listening on `http://localhost:8400`.

## Quickstart (local dev without Docker)

```bash
export FOOSTASH_DB_URL="postgres://foostash:foostash@localhost:5432/foostash?sslmode=disable"
export FOOSTASH_JWT_SECRET="$(openssl rand -base64 32)"
export FOOSTASH_MASTER_KEY="$(go run ./cmd/server genkey)"
export FOOSTASH_MIGRATIONS_DIR="./migrations"

go run ./cmd/server migrate up
go run ./cmd/server serve
```

## CLI usage

```bash
go build -o foostash ./cmd/cli

./foostash register              # create org + user, saves token to ~/.foostash/config.yaml
./foostash init --project myapp  # writes .foostash.yaml
./foostash set DB_HOST=localhost DB_PORT=5432 --env dev
./foostash pull --env dev
./foostash pull --env dev --format dotenv > .env
./foostash run --env dev -- go run main.go
```

## Server commands

```
foostash serve           Run the HTTP API server
foostash migrate up      Apply pending migrations
foostash migrate down N  Revert N migrations
foostash genkey          Generate a base64 AES-256 master key
```

## Environment variables

| Variable | Description |
|---|---|
| `FOOSTASH_PORT` | HTTP port (default: 8400) |
| `FOOSTASH_DB_URL` | Postgres DSN (required) |
| `FOOSTASH_REDIS_URL` | Redis URL (optional in Phase 1) |
| `FOOSTASH_MASTER_KEY` | Base64 32-byte AES key (required unless file set) |
| `FOOSTASH_MASTER_KEY_FILE` | Path to file containing the master key |
| `FOOSTASH_JWT_SECRET` | HMAC secret for dashboard JWTs (required) |
| `FOOSTASH_MIGRATIONS_DIR` | Path to migrations directory (default: `./migrations`) |

## Status

Phase 1 (core OSS): API server with auth + projects + environments + secrets, CLI with login/init/set/pull/run, AES-256-GCM encryption at rest, raw-SQL Postgres store, self-host via Docker Compose.
