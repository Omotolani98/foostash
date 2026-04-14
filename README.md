# Foostash

Encrypted, versioned secrets and environment manager. A single-binary Go CLI backed by a self-hostable HTTP server, authenticated with SSH keys. Replaces `.env` files, Slack-shared credentials, and "wiki with passwords" workflows.

```
┌────────────┐   SSH-signed HTTPS   ┌────────────┐   pgx     ┌────────────┐
│  foostash  │ ───────────────────► │  foostash  │ ────────► │  Postgres  │
│    CLI     │                      │   serve    │           │            │
└────────────┘                      └────────────┘           └────────────┘
   local master.key                   orgs, users, projects,
   (AES-256-GCM)                      envs, invites, audit
```

- **Local-first, zero-knowledge.** Secret values are encrypted locally with your master key (AES-256-GCM). The server never sees plaintext.
- **SSH key auth.** No passwords. Your existing `~/.ssh/id_ed25519` becomes your credential.
- **Self-hostable.** One `docker compose up -d` gets you a running server + Postgres.
- **Team-ready.** Orgs, admin/developer roles, invite tokens, audit log.

## Contents

- [Install](#install)
- [Self-host the server](#self-host-the-server)
- [Register the first admin](#register-the-first-admin)
- [Invite teammates](#invite-teammates)
- [Daily workflow](#daily-workflow)
- [Admin operations](#admin-operations)
- [Configuration](#configuration)
- [Command reference](#command-reference)
- [Security model](#security-model)
- [Development](#development)

## Install

### From source (current recommended path)

```bash
go install github.com/Omotolani98/foostash/cmd/foostash@latest
foostash -v
```

### From a release binary

Download a prebuilt binary from [Releases](https://github.com/Omotolani98/foostash/releases) and place `foostash` on your `PATH`.

> Homebrew tap and install script are on the roadmap; not available yet.

## Self-host the server

### Quick start with Docker Compose

The repo ships a `docker-compose.yml` wiring the published GHCR image to a Postgres container.

```bash
git clone https://github.com/Omotolani98/foostash.git
cd foostash

# pull the published image and start both services
docker compose pull
docker compose up -d

# confirm it's healthy
curl http://localhost:8400/v1/health
# {"status":"ok","version":"v0.2.3","database":"ok"}
```

Defaults:

| Setting             | Value                                                                     |
| ------------------- | ------------------------------------------------------------------------- |
| API port            | `8400` (host and container)                                               |
| Postgres            | `postgres:16-alpine`, user `foostash`, password `foostash`, db `foostash` |
| Data volume         | `foostash-pgdata` (Docker-managed named volume)                           |
| Image               | `ghcr.io/omotolani98/foostash:latest`                                     |
| DB URL passed to app | `postgres://foostash:foostash@postgres:5432/foostash?sslmode=disable`     |

### Pin a specific release

```bash
FOOSTASH_IMAGE_TAG=v0.2.2 docker compose up -d
```

### Build the image locally

```bash
docker compose up -d --build
```

### Run the server without Docker

```bash
# requires a reachable Postgres
export FOOSTASH_PG_URL="postgres://foostash:foostash@localhost:5432/foostash?sslmode=disable"
make build
./bin/foostash serve --addr :8400
```

Or via the Makefile helpers:

```bash
make docker-build    # builds the compose image with version ldflags
make docker-up       # docker compose up -d
make docker-down     # docker compose down
```

### Operational notes

- **Migrations run at startup.** Every invocation of `foostash serve` applies pending SQL migrations idempotently — no separate migrate step.
- **Healthcheck.** `GET /v1/health` pings the DB with a 2s timeout. The compose file uses it for container health; load balancers can too.
- **Data lifecycle.** `docker compose down` keeps the `foostash-pgdata` volume. Use `docker compose down -v` to wipe it.
- **TLS.** The server speaks plain HTTP. Terminate TLS in front (Caddy, nginx, Cloudflare, a managed load balancer) before exposing to the internet.

## Register the first admin

The first user to call `/v1/auth/register` against a fresh server becomes the org's admin. All subsequent users join via invite.

```bash
# generate an SSH key if you don't already have one
ssh-keygen -t ed25519 -f ~/.ssh/id_ed25519 -C "you@example.com"

foostash register \
  --server http://localhost:8400 \
  --email you@example.com \
  --org "Acme"
# registered org=Acme email=you@example.com role=admin
```

What happened:

1. The CLI signed the request with your SSH private key.
2. The server created the org, your user (role `admin`), and stored your public key.
3. `~/.foostash/config.yaml` now carries the server URL and your key fingerprint — future commands don't need `--server`.

A master key is generated on first encrypt at `~/.foostash/master.key` (mode `0600`). Back this up. Losing it means losing local-encrypted secrets.

## Invite teammates

Admins generate tokens; invitees consume them.

```bash
# admin side
foostash admin invite --email alice@acme.com --role developer
# invite created (expires 2026-04-21T12:00:00Z)
#
# Share this with alice@acme.com:
#   foostash join <token>
```

```bash
# invitee side — must have ~/.ssh/id_ed25519 present
foostash join <token> --server http://localhost:8400
# joined org=Acme email=alice@acme.com role=developer
```

Roles:

- **admin** — can create projects/envs, invite users, revoke users, read the audit log.
- **developer** — can read project/env metadata and (today) manage secrets on their own machine.

## Daily workflow

```bash
# scaffold a project (creates .foostash.yaml, commit it)
foostash init --project myapp

# add secrets to an environment
foostash set DB_HOST=localhost DB_PORT=5432 --env dev
foostash set STRIPE_KEY=sk_test_xxx --env prod --secret

# read them back
foostash get DB_HOST --env dev
foostash keys --env dev

# hydrate a process with them (no .env file written)
foostash run --env dev -- go run ./cmd/server

# or dump to stdout as dotenv
foostash pull --env dev

# compare environments
foostash diff dev prod

# version history + rollback
foostash history DB_HOST --env prod
foostash rollback DB_HOST --env prod --version 2

# environments
foostash envs                       # list
foostash envs create staging        # new env
foostash envs clone dev staging     # copy
```

### Import an existing `.env`

```bash
foostash import .env --env dev
```

### Share an environment out-of-band

```bash
# export a password-protected bundle
foostash export --env prod
# Enter encryption password: ********
# wrote myapp-prod.foostash

# recipient:
foostash import myapp-prod.foostash --env prod
# Enter decryption password: ********
```

### Sync secrets across machines

Add `--remote` to any of `set`, `get`, `pull`, `delete`, `history`, `rollback` to talk to the server instead of (or in addition to) the local encrypted store. Values are encrypted **on your machine** before upload — the server only holds ciphertext and nonces.

```bash
# machine A
foostash set STRIPE_KEY=sk_live_xxx --env prod --remote

# machine B (same ~/.foostash/master.key — copy it over securely)
foostash pull --env prod --remote
foostash history STRIPE_KEY --env prod --remote
foostash rollback STRIPE_KEY --env prod --version 2 --remote
```

Any authenticated org member (admin or developer) can read/write secrets for projects in their org. Without `--remote`, commands operate on the local `.enc` files only.

## Admin operations

All admin commands talk to the server and require an admin-role SSH key.

```bash
# list all users in your org
foostash admin users list

# promote or demote
foostash admin users role <user-id> admin
foostash admin users role <user-id> developer

# revoke access (immediate; the user's next request returns 401 user_revoked)
foostash admin users revoke <user-id>

# query the audit log
foostash admin audit --since 24h
foostash admin audit --action project.create
foostash admin audit --user <user-id> --limit 50
```

Audited actions today: `invite.create`, `user.change_role`, `user.revoke`, `project.create`, `project.delete`, `env.create`, `env.clone`, `env.delete`.

## Configuration

### Global config — `~/.foostash/config.yaml`

Written by `foostash register` / `foostash join`. Edit only if you know what you're doing.

```yaml
version: 1
defaults:
  env: dev
  format: dotenv
server: http://localhost:8400
identity:
  email: you@example.com
  ssh_key_path: /home/you/.ssh/id_ed25519
  key_fingerprint: SHA256:AbCdEf...
```

### Project config — `.foostash.yaml` (commit this)

```yaml
project: myapp
default_env: dev
environments:
  - dev
  - staging
  - prod
```

### Environment variables

**Client:**

| Var                     | Purpose                                                                         |
| ----------------------- | ------------------------------------------------------------------------------- |
| `FOOSTASH_MASTER_KEY`   | Overrides the master key file. Base64-encoded 32 bytes. Don't mix with existing `.enc` files encrypted under a different key. |

**Server (`foostash serve` / Docker image):**

| Var                     | Default                                                                | Purpose                  |
| ----------------------- | ---------------------------------------------------------------------- | ------------------------ |
| `FOOSTASH_PG_URL`       | `postgres://foostash:foostash@localhost:5432/foostash?sslmode=disable` | Postgres connection URL. |
| `FOOSTASH_LISTEN_ADDR`  | `:8400`                                                                | HTTP listen address.     |

## Command reference

```
Auth & server
  foostash register    Register the first admin for an org
  foostash login       Re-verify and refresh local identity
  foostash join        Consume an invite token
  foostash serve       Run the HTTP API server (self-host)

Projects & environments
  foostash init        Scaffold .foostash.yaml
  foostash envs        List / create / clone / delete environments

Secrets (local, per-project)
  foostash set         Set one or more keys
  foostash get         Read one key
  foostash delete      Remove a key
  foostash keys        List keys in an env
  foostash pull        Dump an env as dotenv to stdout
  foostash run         Run a command with secrets injected
  foostash diff        Compare two envs
  foostash history     Version history for a key
  foostash rollback    Restore a prior version
  foostash import      Load from .env or a .foostash bundle
  foostash export      Write a password-encrypted bundle

Admin
  foostash admin invite       Create an invite token
  foostash admin users list   List users in the org
  foostash admin users role   Change a user's role
  foostash admin users revoke Revoke a user
  foostash admin audit        Query the audit log
```

Run `foostash <cmd> --help` for flags.

## Security model

- **Crypto.** AES-256-GCM for local secret files and password-protected export bundles. 12-byte random nonce per write; authenticated ciphertext rejects tampering.
- **Master key.** 32 bytes, base64-encoded, stored at `~/.foostash/master.key` (mode `0600`) or passed via `FOOSTASH_MASTER_KEY`. Losing it means losing local data — no recovery path.
- **Wire auth.** Every request is signed with your SSH private key. The server verifies against the public key it registered for your fingerprint. Replay protection via a timestamp header (±5 min window).
- **Server.** Holds org/user/project/env/invite/audit metadata and SSH public keys. Does not hold secret values or private keys.
- **Revocation.** `admin users revoke` flips `revoked_at` on the user row; subsequent SSH-signed requests fail auth.

## Development

```bash
make build         # bin/foostash with version ldflags
make test          # go test ./... -v
make vet           # go vet ./...
make run-server    # go run ./cmd/foostash-server with default env vars
```

Repo layout:

```
cmd/
  foostash/          unified CLI + `foostash serve`
  foostash-server/   server-only entrypoint (same binary logic)
cli/                 Cobra commands
internal/
  crypto/            AES-GCM engine, master key, password-derived keys
  sshauth/           SSH signing / verification
  store/             local encrypted secret files
  secrets/           local service for set/get/versioning
  envs/              local environment manager
  diff/              env comparison
  runner/            process launcher with env injection
  config/            YAML config load/save
  apiclient/         client HTTP with SSH-signed requests
  repo/              pgx repositories (orgs, users, ssh_keys, invites, projects, envs, audit)
  service/           server-side domain logic
  server/            Chi router, middleware (auth, RBAC, audit, logging, recover), handlers
  migrations/        embedded SQL, applied at startup
```

## License

MIT
