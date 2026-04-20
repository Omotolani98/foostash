# Development

```bash
make build         # bin/foostash with version ldflags
make test          # go test ./... -v
make vet           # go vet ./...
make run-server    # go run ./cmd/foostash-server with default env vars
```

## Repo layout

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
  targets/           deployment target renderers (docker, k8s, gh actions)
  config/            YAML config load/save
  apiclient/         client HTTP with SSH-signed requests
  repo/              pgx repositories (orgs, users, ssh_keys, invites, projects, envs, audit)
  service/           server-side domain logic
  server/            Chi router, middleware (auth, RBAC, audit, logging, recover), handlers
  migrations/        embedded SQL, applied at startup
```

## Release process

Releases are cut with [GoReleaser](https://goreleaser.com/) via a GitHub Actions workflow on tag push.

```bash
git tag v0.3.0
git push origin v0.3.0
```

GoReleaser builds cross-platform binaries, publishes the GitHub release, and updates the [Homebrew tap](https://github.com/Omotolani98/homebrew-foostash) formula. The tap push requires a `HOMEBREW_TAP_GITHUB_TOKEN` secret in the release workflow with `contents:write` on `Omotolani98/homebrew-foostash`.
