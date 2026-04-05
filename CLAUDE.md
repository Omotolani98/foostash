# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project status

Foostash is an open-core secrets/env manager (CLI + SDK + API server, with a paid web dashboard). **The repo is a Phase 1 scaffold** — the directory tree and empty package files exist, but almost no code has been written yet. Most `.go` files currently contain only a `package` declaration. The authoritative design spec for everything that needs to be built lives in `ARCHITECTURE.md` (gitignored, but present locally) — read it before implementing anything non-trivial, and treat it as the source of truth for schema, API shape, package layout, and naming.

Module path: `github.com/Omotolani98/foostash` (Go 1.25.1). Note this differs from the `github.com/foostash/foostash` path used in ARCHITECTURE.md examples — use the real module path when writing imports.

The current working branch is `feat/phase-1`.

## Commands

No Makefile, Dockerfile, or CI config exists yet. Standard Go tooling applies:

```
go build ./...                         # build everything
go vet ./...                           # vet
go test ./...                          # run all tests
go test ./internal/crypto -run TestX   # run a single test
go run ./cmd/server                    # run the API server (once main.go is populated)
go run ./cmd/cli                       # run the CLI
```

The SDK at `sdk/go/` is intended to be a **separate Go module** (its own `go.mod`) so public consumers don't pull in `internal/`. It doesn't exist yet; when creating it, `cd sdk/go && go mod init ...` rather than adding files under the root module.

## Architecture

Monorepo with two Go modules (root server+CLI, and `sdk/go` as a standalone public SDK) plus a React dashboard under `web/`.

**Layered backend** — handlers never touch SQL directly:

```
HTTP handler (internal/handler)
    → service (internal/service)   ← business logic, encryption, audit
        → store (internal/store)   ← raw SQL via database/sql + pgx, no ORM
```

- `internal/store` uses **raw SQL with `database/sql` + `pgx`**. No ORM, no query builder, no store interfaces — keep repositories as concrete structs taking `*sql.DB` until there's a real reason to abstract.
- `internal/service` is where encryption (`internal/crypto`) and audit logging are invoked. Services are composed into handlers; handlers only parse requests, call a service, and write responses.
- `internal/router` wires everything using **Chi**. Auth middleware resolves identity (API key or JWT) into an `AuthContext` on the request context; **scope/permission checks happen in handlers, not middleware**.
- `internal/crypto` is AES-256-GCM with a single master key loaded from `FOOSTASH_MASTER_KEY` (base64, 32 bytes) or `FOOSTASH_MASTER_KEY_FILE`. Every secret gets a fresh 12-byte nonce stored alongside its ciphertext. **The master key must never be written to the database.**
- Audit logging is fire-and-forget from services (`audit.LogAsync`) — it must not block or fail the request.

**Data model core entities** (see ARCHITECTURE.md §4 for full schema): `organizations → users → projects → environments → secrets` (+ `secret_versions`, `api_keys`, `audit_logs`). API keys are stored as SHA-256 hashes with a `key_prefix` for display; the raw `fst_...` key is only shown once at creation. Redis caches key_hash→auth resolution with a short TTL (~60s) to avoid per-request Postgres hits.

**Migrations** live in `migrations/` as plain numbered `NNN_name.up.sql` / `.down.sql` files applied by a custom `foostash migrate` subcommand tracking a `schema_migrations` table — **do not introduce a migration framework** (golang-migrate, goose, etc.).

**CLI vs SDK vs server**:
- The CLI (`cmd/cli` + `cli/`, using Cobra) is a thin HTTP client against the server for normal commands (`login`, `pull`, `set`, `run`, `diff`). The exception is `foostash migrate`, which connects directly to Postgres for self-hosted admin.
- The Go SDK (`sdk/go/`) also talks to the server over HTTP and **must not import `internal/`** — it's a separately published module.
- `cli/client/` holds the HTTP client shared by CLI commands.

**API shape**: versioned under `/v1`, JSON, bearer token auth (`fst_...` API keys or JWTs). All errors use the envelope `{"error":{"code","message","status"}}` with standard codes: `bad_request`, `unauthorized`, `forbidden`, `not_found`, `conflict`, `rate_limited`, `internal`. Full endpoint list in ARCHITECTURE.md §5.

**Paid-tier features** (audit log viewer, team/RBAC, usage analytics, rotation reminders) live in the same codebase but behind plan checks — there is no separate "enterprise" fork.

## Conventions

- Wrap errors with `fmt.Errorf("context: %w", err)` at layer boundaries; let them propagate to the handler which maps them to the error envelope.
- Secret **values** must never appear in logs or audit entries — audit records only reference the `key` that was accessed.
- When adding a new endpoint: add the SQL to a store, the business logic to a service, the HTTP binding to a handler, and register the route in `internal/router`. Don't skip layers.