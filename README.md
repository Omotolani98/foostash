# Foostash

An open-source, developer-first secrets and environment manager. CLI-only, local-first, per-project and global config. Replaces `.env` files with an encrypted, versioned secrets store.

## Why Foostash?

| Pain Point | Foostash Solution |
|-----------|---------------|
| `.env` files committed to git | Encrypted local store, never plaintext on disk |
| No way to compare environments | `foostash diff dev staging` |
| Sharing secrets over Slack/email | Export/import password-protected bundles |
| Different `.env` per environment | One command: `foostash pull --env prod` |
| No version history | Every change is versioned locally |

## Installation

### Binary (recommended)

Download from [releases](https://github.com/Omotolani98/foostash/releases):

```bash
# macOS
brew install foostash/tap/foostash

# Linux
curl -sL https://foostash.sh | bash

# Windows (PowerShell)
irm https://foostash.sh | iex
```

### Build from source

```bash
go build -o foostash ./cmd/foostash
./foostash --help
```

## Quick Start

```bash
# Initialize a project
foostash init --project myapp
# Creates .foostash.yaml

# Set secrets
foostash set DB_HOST=localhost DB_PORT=5432 --env dev
foostash set STRIPE_KEY=sk_test_xxx --env prod --secret

# Pull secrets (dotenv format)
foostash pull --env dev
# DB_HOST=localhost
# DB_PORT=5432

# Run with secrets injected
foostash run --env dev -- go run main.go
```

## Commands

| Command | Description |
|---------|------------|
| `foostash init` | Initialize a project |
| `foostash set KEY=VALUE...` | Set secrets |
| `foostash get KEY` | Get a secret |
| `foostash delete KEY` | Delete a secret |
| `foostash pull` | Pull secrets as env vars |
| `foostash run -- CMD` | Run command with secrets |
| `foostash diff ENV1 ENV2` | Compare two environments |
| `foostash envs` | List environments |
| `foostash envs create NAME` | Create environment |
| `foostash envs clone SRC DEST` | Clone environment |
| `foostash keys` | List keys in env |
| `foostash history KEY` | Show version history |
| `foostash rollback KEY --version N` | Rollback to version |
| `foostash import FILE` | Import from .env or .foostash bundle |
| `foostash export` | Export password-protected bundle |

## Examples

### Compare environments

```bash
$ foostash diff dev prod

  Key          dev              prod
  ─────────────────────────────────────
+ SENTRY_DSN   —                dsn://...
- DEBUG_MODE   true             —
~ DB_HOST      localhost        prod-db.internal
= APP_NAME     myapp            myapp
```

### Export & share with teammate

```bash
# Teammate A: Export encrypted bundle
foostash export --env prod
# Enter encryption password: ********

# Teammate B: Import bundle
foostash import prod-prod.foostash --env prod
# Enter decryption password: ********
```

### Version rollback

```bash
$ foostash history DB_HOST --env prod
  Version  Value              Set At
  ────────────────────────────────────
  3        prod-db.internal   2026-04-09 14:30
  2        staging-db       2026-04-05 09:15
  1        localhost      2026-04-01 10:00

$ foostash rollback DB_HOST --env prod --version 2
rolled back DB_HOST to version 2
```

## Configuration

### Global config: `~/.foostash/config.yaml`

```yaml
version: 1
defaults:
  env: dev
  format: dotenv
```

### Project config: `.foostash.yaml` (committed to git)

```yaml
project: myapp
default_env: dev
environments:
  - dev
  - staging
  - prod
```

### Master key

Foostash auto-generates `~/.foostash/master.key` on first run. Override with:

```bash
export FOOSTASH_MASTER_KEY="base64-encoded-32-byte-key"
```

## Self-Host with Docker

Run the foostash HTTP server and Postgres with one command:

```bash
docker compose up -d
```

The API listens on `:8400`. Data persists in the `foostash-pgdata` volume. Override the defaults via env vars (`FOOSTASH_PG_URL`, `FOOSTASH_LISTEN_ADDR`). To run the server directly from source:

```bash
make build && ./bin/foostash serve
```

## Architecture

Secrets are stored as AES-256-GCM encrypted JSON files in `~/.foostash/projects/{project}/{env}.enc`. Each file contains current values and full version history per key (capped at 50 versions).

No server. No database. No account. Everything is local.

## License

MIT