# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

- `make build` — builds `bin/foostash` with version ldflags injected into `cli.Version/Commit/Date`.
- `make install` — `go install` with the same ldflags.
- `make test` — `go test ./... -v`.
- `make vet` — `go vet ./...`.
- Run a single test: `go test ./internal/store -run TestName -v`.
- Run the CLI during dev without building: `go run ./cmd/foostash <args>`.

Go 1.25.1. Release tooling is GoReleaser (`.goreleaser.yml`).

## Architecture

Entry point is `cmd/foostash/main.go`, which just calls `cli.NewRootCmd().Execute()`. Everything else lives in two top-level packages:

- `cli/` — Cobra command definitions (one file per subcommand: `set.go`, `get.go`, `pull.go`, `run.go`, `diff.go`, `envs.go`, `history.go`, `rollback.go`, `import.go`, `export.go`, etc.). `root.go` wires them together.
- `internal/` — all business logic. Commands in `cli/` are thin wrappers that call into these packages.

### Dependency injection via `cli.App`

`cli/root.go` defines an `App` struct holding `*store.Store`, `*secrets.Service`, `*envs.Manager`. The root command's `PersistentPreRunE` loads the master key, constructs the crypto engine, and populates `App` before any subcommand runs. Commands like `init`, `help`, `version`, `completion` skip this initialization. When adding a new subcommand that needs secrets, take `*App` and register it in `NewRootCmd()`'s `AddCommand` call.

### Internal package layout

- `internal/crypto` — AES-256-GCM engine (`engine.go`), master key load/generate at `~/.foostash/master.key` or `FOOSTASH_MASTER_KEY` env var (`key.go`), password-based key derivation for export bundles (`password.go`).
- `internal/store` — persistence layer. `SecretFile` holds `Secrets` (current values) and `History` (capped at 50 versions per key). `Store.Load`/`Save` handle the on-disk format: `[12-byte nonce][GCM ciphertext]`, written atomically via tmp+rename with `0600` perms. Path helpers: `EnvPath(project, env)` → `~/.foostash/projects/{project}/{env}.enc`, `GlobalsPath()` → `~/.foostash/globals.enc`.
- `internal/secrets` — `Service` is the main API for get/set/delete/list operations on secrets; handles versioning and history trimming.
- `internal/envs` — `Manager` for creating, listing, cloning environments.
- `internal/diff` — environment comparison logic and formatted output.
- `internal/runner` — spawns child processes with secrets injected into their environment (used by `foostash run`).
- `internal/config` — loads/writes `~/.foostash/config.yaml` (global) and `.foostash.yaml` (per-project, committed).

### Data model invariants

- Every mutation increments `SecretFile.Version` and the per-key `SecretEntry.Version`, and appends to `History[key]` (oldest entries trimmed beyond 50).
- Files are **never** written in plaintext. Any code path that persists secrets goes through `Store.Save`, which encrypts before writing.
- The master key is the root of trust for all local secret files. Export bundles (`foostash export`) re-encrypt under a user-supplied password (see `internal/crypto/password.go`) so bundles are portable across machines without sharing the master key.

### ARCHITECTURE.md vs. reality

`ARCHITECTURE.md` describes a broader long-term vision (SSH auth, TUI, self-hosted server, SaaS at foosta.sh). The current codebase is **CLI-only and local-first** — there is no server, network, or SSH code. Treat ARCHITECTURE.md as forward-looking design; don't assume features in it exist until you see them in `cli/` or `internal/`.
