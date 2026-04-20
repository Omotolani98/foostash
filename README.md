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

## Install

```bash
brew install Omotolani98/foostash/foostash
```

Other install options (source, release binaries) in [docs/install.md](docs/install.md).

## Quick start

```bash
# self-host the server
docker compose up -d

# register as the first admin
foostash register --server http://localhost:8400 --email you@example.com --org "Acme"

# scaffold a project and start storing secrets
foostash init --project myapp
foostash set DB_HOST=localhost DB_PORT=5432 --env dev
foostash run --env dev -- go run ./cmd/server
```

Full walkthroughs in [docs/getting-started.md](docs/getting-started.md) and [docs/daily-workflow.md](docs/daily-workflow.md).

## Documentation

- [Install](docs/install.md)
- [Self-host the server](docs/self-hosting.md)
- [Getting started](docs/getting-started.md)
- [Daily workflow](docs/daily-workflow.md)
- [Admin operations](docs/admin.md)
- [Configuration](docs/configuration.md)
- [Command reference](docs/commands.md)
- [Security model](docs/security.md)
- [Development](docs/development.md)

## License

MIT
