# foostash-go-sdk

Read-only Go SDK for the [foostash](https://github.com/Omotolani98/foostash) secrets manager. Use it to load and watch project secrets from a foostash server at runtime, without shipping the `foostash` CLI binary with your service.

## Install

```bash
go get github.com/Omotolani98/foostash-go-sdk
```

The SDK module lives inside the main foostash repo under `sdk/go/`, but is released with independent subdirectory tags (e.g. `sdk/go/v0.1.0`).

## Usage

```go
package main

import (
    "context"
    "log"
    "os"
    "time"

    foostash "github.com/Omotolani98/foostash-go-sdk"
)

func main() {
    client, err := foostash.New(foostash.Config{
        ServerURL:  "https://foostash.example.com",
        Project:    "payments-api",
        Env:        "prod",
        SSHKeyPath: "/var/run/secrets/foostash-key",
        MasterKey:  os.Getenv("FOOSTASH_MASTER_KEY"),
    })
    if err != nil {
        log.Fatal(err)
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    secrets, err := client.Pull(ctx)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("loaded %d secrets", len(secrets))
}
```

### Watching for changes

```go
ch, err := client.Watch(ctx, 30*time.Second)
if err != nil {
    log.Fatal(err)
}
for snap := range ch {
    reloadConfig(snap.Secrets)
}
```

`Watch` polls the server on the given interval (clamped to a 5-second floor) and emits a new `Snapshot` whenever any key's version changes. The initial snapshot is emitted synchronously before the first tick, so the first receive returns your current config. The channel closes when the context is canceled. Transient network failures do **not** close the channel — attach a `*slog.Logger` with `foostash.WithLogger(l)` to observe them.

## Requirements

You need two things before the SDK can authenticate:

1. **An SSH key registered with the server.** Use the `foostash` CLI once (`foostash register` or `foostash join`) to provision a service-account key, then mount the unencrypted private key into your service. Passphrase-protected keys are not supported — they're designed for humans, not long-running services.
2. **The org's master key** (base64-encoded, 32 bytes). Distribute this via your secret delivery mechanism of choice (KMS, env var, file mount). The SDK never reads `~/.foostash/`.

## Errors

The SDK returns sentinel errors you can branch on with `errors.Is`:

- `foostash.ErrNotFound` — 404 from the server
- `foostash.ErrUnauthorized` — 401
- `foostash.ErrForbidden` — 403
- `foostash.ErrInvalidKeyLength` — master key didn't decode to 32 bytes

Everything else is returned wrapped; use `errors.As` with `*apiclient.APIError` if you need the raw status/code.

## Scope

v1 is **read-only**: `Get`, `Pull`, `Watch` for project secrets. Writes (`Set`, `Delete`, `Rollback`), history access, and the vault namespace stay on the CLI for now. Open an issue on the main foostash repo if you need these in the SDK.

## Development

```bash
# from the repo root
make sdk-test    # runs go test ./... inside sdk/go
make sdk-vet     # runs go vet ./... inside sdk/go
```

When working on both the SDK and server in the same change, add a `go.work` at the repo root:

```
go 1.25

use (
  .
  ./sdk/go
)
```

`go.work` is `.gitignore`d — it's a local development convenience only.

## Releases

Tag the SDK independently of the main binary:

```bash
git tag sdk/go/v0.1.0
git push origin sdk/go/v0.1.0
```

Consumers pull with `go get github.com/Omotolani98/foostash-go-sdk@v0.1.0`.
