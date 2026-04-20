# Configuration

## Global config — `~/.foostash/config.yaml`

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

## Project config — `.foostash.yaml` (commit this)

```yaml
project: myapp
default_env: dev
environments:
  - dev
  - staging
  - prod
```

## Environment variables

### Client

| Var                   | Purpose                                                                                                                      |
| --------------------- | ---------------------------------------------------------------------------------------------------------------------------- |
| `FOOSTASH_MASTER_KEY` | Overrides the master key file. Base64-encoded 32 bytes. Don't mix with existing `.enc` files encrypted under a different key. |

### Server (`foostash serve` / Docker image)

| Var                    | Default                                                                | Purpose                  |
| ---------------------- | ---------------------------------------------------------------------- | ------------------------ |
| `FOOSTASH_PG_URL`      | `postgres://foostash:foostash@localhost:5432/foostash?sslmode=disable` | Postgres connection URL. |
| `FOOSTASH_LISTEN_ADDR` | `:8400`                                                                | HTTP listen address.     |
