# Self-host the server

## Quick start with Docker Compose

The repo ships a `docker-compose.yml` wiring the published GHCR image to a Postgres container.

```bash
git clone https://github.com/Omotolani98/foostash.git
cd foostash

docker compose pull
docker compose up -d

curl http://localhost:8400/v1/health
# {"status":"ok","version":"v0.2.3","database":"ok"}
```

Defaults:

| Setting              | Value                                                                     |
| -------------------- | ------------------------------------------------------------------------- |
| API port             | `8400` (host and container)                                               |
| Postgres             | `postgres:16-alpine`, user `foostash`, password `foostash`, db `foostash` |
| Data volume          | `foostash-pgdata` (Docker-managed named volume)                           |
| Image                | `ghcr.io/omotolani98/foostash:latest`                                     |
| DB URL passed to app | `postgres://foostash:foostash@postgres:5432/foostash?sslmode=disable`     |

## Pin a specific release

```bash
FOOSTASH_IMAGE_TAG=v0.2.2 docker compose up -d
```

## Build the image locally

```bash
docker compose up -d --build
```

## Run without Docker

```bash
export FOOSTASH_PG_URL="postgres://foostash:foostash@localhost:5432/foostash?sslmode=disable"
make build
./bin/foostash serve --addr :8400
```

Makefile helpers:

```bash
make docker-build    # builds the compose image with version ldflags
make docker-up       # docker compose up -d
make docker-down     # docker compose down
```

## Operational notes

- **Migrations run at startup.** Every `foostash serve` invocation applies pending SQL migrations idempotently — no separate migrate step.
- **Healthcheck.** `GET /v1/health` pings the DB with a 2s timeout. The compose file uses it for container health; load balancers can too.
- **Data lifecycle.** `docker compose down` keeps the `foostash-pgdata` volume. Use `docker compose down -v` to wipe it.
- **TLS.** The server speaks plain HTTP. Terminate TLS in front (Caddy, nginx, Cloudflare, a managed load balancer) before exposing to the internet.
