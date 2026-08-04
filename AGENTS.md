# Repository Guidelines

## What this is

Foostash is an encrypted, versioned secrets manager: a single Go CLI binary (`cmd/foostash`, includes `foostash serve`) plus a server-only binary (`cmd/foostash-server`), backed by Postgres. Go 1.25.8. Two Go modules: repo root and `sdk/go` (independent module, tagged separately as `sdk/go/v*`).

## Commands

- `make build` / `make build-server` — build `bin/foostash` / `bin/foostash-server` with version ldflags into `cli.Version/Commit/Date`.
- `make test` — `go test ./... -v` for the root module only. It does **not** run SDK tests.
- `make sdk-test` / `make sdk-vet` — SDK tests/vet, run from `sdk/go`. Run both when touching `sdk/go/**` (that's all the SDK CI workflow runs: vet then test).
- Focused test: `go test ./internal/store -run TestName -v`; for the SDK, `cd sdk/go && go test -run TestName -v`.
- `make run-server` — runs `cmd/foostash-server` against local Postgres. Needs Postgres up first (e.g. `docker compose up -d postgres`).
- `docker compose up -d` — full self-hosted stack (Postgres 16 + server from the Dockerfile, HTTP on `:8400`).

Gotcha: `make run-server` exports `FOOSTASH_LISTEN_ADDR`, but `cmd/foostash-server` actually reads `PORT` for its listen address (`FOOSTASH_LISTEN_ADDR` is only read by `foostash serve`). The Makefile export is ignored; the binary falls back to `:8400`.

## Architecture map

- `cli/` — Cobra commands, one file per subcommand. Keep these thin; logic goes in `internal/`. `cli.App` (in `root.go`) is built in `PersistentPreRunE` by loading the master key and crypto engine; commands in the skip-list there (`init`, `register`, `serve`, `vault`, …) get no `App`. New secret-touching subcommands take `*App` and register in `NewRootCmd`'s `AddCommand`.
- `internal/server` — Chi router + middleware + handlers; `boot.go` connects pgx, applies migrations, wires services. `internal/service` — domain logic. `internal/repo` — pgx persistence.
- `internal/migrations` — embedded `*.sql`, applied in filename order at every server startup. Files must be idempotent (`IF NOT EXISTS`); there is no separate migrate command and no down-migrations.
- `internal/sshsrv` + `internal/tui` — SSH-served TUI (Wish + Bubble Tea v2). `foostash serve` enables it on `:2222` by default; `cmd/foostash-server` only starts SSH when `FOOSTASH_SSH_ADDR` is set.
- `internal/billing` — SaaS-only (Lemon Squeezy), gated by `FOOSTASH_BILLING`. Default is noop: no billing routes are mounted for self-hosters. Keep billing code out of self-host hot paths.
- `web/` — static landing page only; `vercel.json` deploys the server-only binary to Vercel.
- `CLAUDE.md` is **stale** (claims there is no server/SSH code). Trust this file and the code, not CLAUDE.md. `docs/development.md` has an accurate package list.

## Invariants to preserve

- The server must never see plaintext secret values. Clients encrypt with the org master key (AES-256-GCM) before upload; the server stores only ciphertext + nonce (`internal/repo/secrets.go`). Any new secret-handling path must keep this.
- Local secret files are written only through `internal/store` (`Save` encrypts, writes atomically via tmp+rename, mode `0600`).
- Every HTTP API request is SSH-signature authenticated (`internal/sshauth`); register/join endpoints buffer the body (`middleware.BufferBody`) because the key isn't known yet.
- TUI code uses Bubble Tea **v2** (`charm.land/bubbletea/v2`): `View() tea.View` (return `tea.NewView(...)`), sizes arrive via `tea.WindowSizeMsg`, not the v1 API.

## Testing

All current tests are pure unit tests (temp dirs, `httptest` fakes) — no Postgres or external services needed. `go test ./...` at root covers `internal/…` only; SDK tests live behind `make sdk-test`.

## Releases

Tag `v*` → GitHub Actions runs GoReleaser (binaries + Homebrew tap, needs `HOMEBREW_TAP_GITHUB_TOKEN`) and builds/pushes the GHCR image. SDK releases are separate subdirectory tags (`sdk/go/v0.x.y`). `go.work` for cross-module dev is gitignored on purpose.

## Security

Never commit plaintext secrets, master keys, `.env` files, or `~/.foostash` data. Server config is env-only: `FOOSTASH_PG_URL`, `FOOSTASH_LISTEN_ADDR`/`PORT`, `FOOSTASH_SSH_ADDR`, `FOOSTASH_SSH_HOST_KEY`, `FOOSTASH_BILLING` + `FOOSTASH_LS_*`.
