# Foostash — Architecture Document

> An open-source, developer-first secrets and environment manager.
> CLI-only · Local-first · Per-project and global config

---

## 1. Product Overview

**Foostash** replaces `.env` files with an encrypted, versioned secrets store managed entirely from the command line. Secrets are organized by **project → environment** (e.g., `myapp/staging`), diffable across environments, and stored locally with AES-256-GCM encryption.

No server. No dashboard. No account. Just a CLI and a config file.

### Value Proposition

| Pain Point | Foostash Solution |
|---|---|
| `.env` files committed to git | Encrypted local store, never plaintext on disk |
| No way to compare environments | `foostash diff dev staging` |
| Sharing secrets over Slack/email | Export/import encrypted bundles |
| Different `.env` per environment | One command: `foostash pull --env prod` |
| No version history | Every change is versioned locally |

---

## 2. User Workflows

### Init & Configure

```
$ foostash init
  → Creates .foostash.yaml in current directory, links to a project name
  → Prompts for project name if not provided

$ foostash init --project payments-api
  → Explicit project name
```

### Setting Secrets

```
# Set secrets for a specific environment
$ foostash set DB_HOST=localhost DB_PORT=5432 --env dev

# Set secrets for the default environment (from .foostash.yaml)
$ foostash set STRIPE_KEY=sk_test_abc123

# Set a secret interactively (hidden input, for passwords)
$ foostash set DB_PASSWORD --env prod --secret

# Set global secrets (available to all projects)
$ foostash set --global GITHUB_TOKEN=ghp_abc123
```

### Pulling Secrets

```
# Print all secrets for an environment to stdout
$ foostash pull --env dev

# Output as dotenv format
$ foostash pull --env dev --format dotenv

# Write to .env file
$ foostash pull --env dev --format dotenv > .env

# Pull as JSON
$ foostash pull --env dev --format json

# Pull a single key
$ foostash get DB_HOST --env dev
```

### Running with Secrets

```
# Inject secrets as env vars and run a command
$ foostash run --env dev -- go run main.go

# Inject secrets and run with global secrets merged
$ foostash run --env dev --with-globals -- docker compose up
```

### Diffing Environments

```
# Compare two environments
$ foostash diff dev staging

  Key          dev              staging
  ─────────────────────────────────────────
+ SENTRY_DSN   —                dsn://...
- DEBUG_MODE   true             —
~ DB_HOST      localhost        staging-db.internal
~ DB_PORT      5432             5433
= APP_NAME     myapp            myapp

# Diff output legend:
#   + only in right
#   - only in left
#   ~ different values
#   = identical

# Show only differences (hide identical keys)
$ foostash diff dev staging --only-changes

# Diff as JSON (for scripting)
$ foostash diff dev staging --format json
```

### Environment Management

```
# List all environments for current project
$ foostash envs

# Create a new environment
$ foostash envs create staging

# Clone an environment (copy all secrets)
$ foostash envs clone dev staging

# Delete an environment
$ foostash envs delete staging

# List all keys in an environment
$ foostash keys --env dev
```

### Import / Export

```
# Import from an existing .env file
$ foostash import .env --env dev

# Import from .env.staging
$ foostash import .env.staging --env staging

# Export encrypted bundle (for sharing with teammate)
$ foostash export --env prod --out prod-secrets.foostash

# Import encrypted bundle
$ foostash import prod-secrets.foostash --env prod
```

### Version History

```
# Show version history for a key
$ foostash history DB_HOST --env dev

  Version  Value              Set At                Set By
  ──────────────────────────────────────────────────────────
  3        prod-db.internal   2026-04-09 14:30:00   —
  2        staging-db         2026-04-05 09:15:00   —
  1        localhost          2026-04-01 10:00:00   —

# Rollback a key to a previous version
$ foostash rollback DB_HOST --env dev --version 2
```

---

## 3. Architecture

```
┌──────────────────────────────────────────────────┐
│                  CLI (Cobra)                      │
│                                                   │
│  set / pull / run / diff / envs / import/export   │
└──────────────┬───────────────────────────────────┘
               │
               ▼
┌──────────────────────────────────────────────────┐
│              Core Domain                          │
│                                                   │
│  secrets/    → get, set, list, diff, versions     │
│  envs/       → create, clone, delete, list        │
│  projects/   → init, resolve current project      │
│  crypto/     → AES-256-GCM encrypt/decrypt        │
│  config/     → load/save global + project config  │
└──────────────┬───────────────────────────────────┘
               │
               ▼
┌──────────────────────────────────────────────────┐
│              Storage (Local Files)                │
│                                                   │
│  ~/.foostash/                                     │
│  ├── config.yaml       (global config + secrets)  │
│  ├── master.key        (master encryption key)    │
│  └── projects/                                    │
│      └── payments-api/                            │
│          ├── dev.enc   (encrypted secrets)        │
│          ├── staging.enc                          │
│          └── prod.enc                             │
└──────────────────────────────────────────────────┘
```

No database. No server. Everything is local encrypted files.

---

## 4. Config Design

### Global Config: `~/.foostash/config.yaml`

Created on first `foostash set --global` or `foostash init`. Stores global secrets and CLI preferences.

```yaml
# ~/.foostash/config.yaml

# Global secrets available to all projects (encrypted values stored in globals.enc)
# This file only holds config, not secret values

version: 1

defaults:
  env: dev                    # default environment when --env is omitted
  format: dotenv              # default output format for pull

# Global secrets are stored encrypted in ~/.foostash/globals.enc
# Managed via `foostash set --global KEY=VALUE`
```

### Project Config: `.foostash.yaml`

Created by `foostash init` in the project root. Committed to git.

```yaml
# .foostash.yaml (committed to git)

project: payments-api
default_env: dev

environments:
  - dev
  - staging
  - prod
```

This file contains **no secrets** — only project metadata. It tells Foostash which project directory maps to which secret store in `~/.foostash/projects/`.

### Master Key: `~/.foostash/master.key`

Auto-generated on first use. 32 bytes, base64 encoded. Permissions set to `0600`.

```
# ~/.foostash/master.key (auto-generated, never committed)
# chmod 0600
K7xP2mN9qR4tV6wY8zA1bC3dE5fG7hJ9kL0mN2pQ4r=
```

Override with environment variable:

```bash
export FOOSTASH_MASTER_KEY="base64-encoded-32-byte-key"
```

---

## 5. Storage Format

### Encrypted Secret Files

Each environment's secrets are stored as a single encrypted JSON blob:

```
~/.foostash/projects/payments-api/dev.enc
```

Decrypted structure:

```json
{
  "version": 5,
  "updated_at": "2026-04-09T14:30:00Z",
  "secrets": {
    "DB_HOST": {
      "value": "localhost",
      "version": 3,
      "updated_at": "2026-04-09T14:30:00Z"
    },
    "DB_PORT": {
      "value": "5432",
      "version": 1,
      "updated_at": "2026-04-01T10:00:00Z"
    }
  },
  "history": {
    "DB_HOST": [
      { "version": 3, "value": "prod-db.internal", "set_at": "2026-04-09T14:30:00Z" },
      { "version": 2, "value": "staging-db", "set_at": "2026-04-05T09:15:00Z" },
      { "version": 1, "value": "localhost", "set_at": "2026-04-01T10:00:00Z" }
    ]
  }
}
```

The entire file is encrypted with AES-256-GCM. On every read/write:
1. Read file → decrypt → unmarshal JSON
2. Modify in memory
3. Marshal JSON → encrypt → write file

### Global Secrets

Same format, stored at `~/.foostash/globals.enc`. Merged with project secrets during `pull` and `run` (project secrets take precedence).

### File Layout

```
~/.foostash/
├── config.yaml                  # CLI preferences, defaults
├── master.key                   # 32-byte AES key (base64, 0600 perms)
├── globals.enc                  # Global secrets (encrypted JSON)
└── projects/
    ├── payments-api/
    │   ├── dev.enc
    │   ├── staging.enc
    │   └── prod.enc
    └── bawo-core/
        ├── dev.enc
        └── prod.enc
```

---

## 6. Encryption Design

```go
// internal/crypto/engine.go

type Engine struct {
    aead cipher.AEAD
}

func NewEngine(masterKey []byte) (*Engine, error) {
    if len(masterKey) != 32 {
        return nil, fmt.Errorf("master key must be 32 bytes, got %d", len(masterKey))
    }
    block, err := aes.NewCipher(masterKey)
    if err != nil {
        return nil, err
    }
    aead, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }
    return &Engine{aead: aead}, nil
}

func (e *Engine) Encrypt(plaintext []byte) ([]byte, error) {
    nonce := make([]byte, e.aead.NonceSize()) // 12 bytes
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return nil, err
    }
    // nonce is prepended to ciphertext
    return e.aead.Seal(nonce, nonce, plaintext, nil), nil
}

func (e *Engine) Decrypt(data []byte) ([]byte, error) {
    nonceSize := e.aead.NonceSize()
    if len(data) < nonceSize {
        return nil, fmt.Errorf("ciphertext too short")
    }
    nonce, ciphertext := data[:nonceSize], data[nonceSize:]
    return e.aead.Open(nil, nonce, ciphertext, nil)
}
```

Nonce is prepended to the ciphertext in the `.enc` file (no separate nonce storage needed).

### Master Key Lifecycle

```
First run:
  1. Check FOOSTASH_MASTER_KEY env var
  2. Check ~/.foostash/master.key file
  3. If neither exists, generate 32 random bytes, write to ~/.foostash/master.key (0600)

Key rotation (future):
  $ foostash admin rotate-key
  → Decrypts all .enc files with old key, re-encrypts with new key
```

---

## 7. Diff Engine

The diff command compares two environment files and categorizes every key:

```go
// internal/diff/diff.go

type DiffResult struct {
    OnlyLeft    []string             // keys only in left env
    OnlyRight   []string             // keys only in right env
    Different   map[string][2]string // key → [leftVal, rightVal]
    Identical   []string             // same key, same value
}

func Compare(left, right map[string]string) DiffResult {
    result := DiffResult{
        Different: make(map[string][2]string),
    }

    for k, lv := range left {
        rv, exists := right[k]
        if !exists {
            result.OnlyLeft = append(result.OnlyLeft, k)
        } else if lv != rv {
            result.Different[k] = [2]string{lv, rv}
        } else {
            result.Identical = append(result.Identical, k)
        }
    }

    for k := range right {
        if _, exists := left[k]; !exists {
            result.OnlyRight = append(result.OnlyRight, k)
        }
    }

    return result
}
```

### Diff Output Formats

**Table (default):**
```
$ foostash diff dev staging

  Key          dev              staging
  ─────────────────────────────────────────
+ SENTRY_DSN   —                dsn://...
- DEBUG_MODE   true             —
~ DB_HOST      localhost        staging-db.internal
= APP_NAME     myapp            myapp
```

**JSON (for scripting):**
```json
{
  "left": "dev",
  "right": "staging",
  "only_left": ["DEBUG_MODE"],
  "only_right": ["SENTRY_DSN"],
  "different": {
    "DB_HOST": { "left": "localhost", "right": "staging-db.internal" }
  },
  "identical": ["APP_NAME"]
}
```

### Value Masking

By default, diff shows actual values. Use `--mask` to redact:

```
$ foostash diff dev staging --mask

  Key          dev              staging
  ─────────────────────────────────────────
+ SENTRY_DSN   —                ****
- DEBUG_MODE   ****             —
~ DB_HOST      ****             ****
= APP_NAME     myapp            myapp
```

---

## 8. Secret Merging (Global + Project)

When running `foostash pull` or `foostash run`, secrets are merged in this order (last wins):

```
1. Global secrets     (~/.foostash/globals.enc)
2. Project secrets    (~/.foostash/projects/{project}/{env}.enc)
3. Existing env vars  (from the shell, only for `run`)
```

Project secrets override globals. Shell env vars override everything (so you can always override a secret without touching Foostash).

```go
func MergeSecrets(globals, project map[string]string, includeEnv bool) map[string]string {
    merged := make(map[string]string)

    for k, v := range globals {
        merged[k] = v
    }
    for k, v := range project {
        merged[k] = v // project overrides global
    }

    if includeEnv {
        // existing shell env vars are NOT overwritten
        // they take highest precedence when used in `run`
    }

    return merged
}
```

---

## 9. Project Structure

```
foostash/
├── cmd/
│   └── foostash/
│       └── main.go                # CLI entrypoint
│
├── internal/
│   ├── config/
│   │   ├── config.go              # Load/save ~/.foostash/config.yaml
│   │   └── project.go             # Load/save .foostash.yaml
│   │
│   ├── crypto/
│   │   ├── engine.go              # AES-256-GCM encrypt/decrypt
│   │   ├── engine_test.go
│   │   └── key.go                 # Master key loading/generation
│   │
│   ├── store/
│   │   ├── store.go               # Read/write encrypted .enc files
│   │   ├── store_test.go
│   │   └── globals.go             # Global secrets store
│   │
│   ├── secrets/
│   │   ├── service.go             # Business logic: set, get, pull, merge
│   │   └── service_test.go
│   │
│   ├── diff/
│   │   ├── diff.go                # Compare two secret maps
│   │   ├── diff_test.go
│   │   └── format.go              # Table and JSON formatters
│   │
│   ├── envs/
│   │   ├── manager.go             # Create, clone, delete, list environments
│   │   └── manager_test.go
│   │
│   └── runner/
│       └── runner.go              # Exec child process with injected env vars
│
├── cli/
│   ├── root.go                    # Cobra root command + global flags
│   ├── init.go                    # foostash init
│   ├── set.go                     # foostash set
│   ├── get.go                     # foostash get
│   ├── pull.go                    # foostash pull
│   ├── run.go                     # foostash run
│   ├── diff.go                    # foostash diff
│   ├── envs.go                    # foostash envs (create/clone/delete/list)
│   ├── import.go                  # foostash import
│   ├── export.go                  # foostash export
│   ├── history.go                 # foostash history
│   ├── rollback.go                # foostash rollback
│   └── keys.go                    # foostash keys
│
├── go.mod                         # module github.com/foostash/foostash
├── go.sum
├── Makefile
├── ARCHITECTURE.md
├── README.md
└── LICENSE
```

---

## 10. CLI Commands Reference

| Command | Description |
|---|---|
| `foostash init` | Initialize a project in the current directory |
| `foostash set KEY=VAL [KEY=VAL...]` | Set one or more secrets |
| `foostash set --global KEY=VAL` | Set a global secret |
| `foostash set KEY --secret` | Set a secret with hidden input |
| `foostash get KEY` | Get a single secret value |
| `foostash pull` | Pull all secrets for an environment |
| `foostash run -- CMD` | Run a command with secrets injected |
| `foostash diff ENV1 ENV2` | Compare two environments |
| `foostash envs` | List environments |
| `foostash envs create NAME` | Create an environment |
| `foostash envs clone SRC DEST` | Clone an environment |
| `foostash envs delete NAME` | Delete an environment |
| `foostash keys` | List all keys in an environment |
| `foostash import FILE` | Import from .env file or encrypted bundle |
| `foostash export` | Export encrypted bundle |
| `foostash history KEY` | Show version history for a key |
| `foostash rollback KEY --version N` | Rollback a key to a previous version |

### Global Flags

| Flag | Description |
|---|---|
| `--env, -e` | Target environment (default: from `.foostash.yaml` or config) |
| `--project, -p` | Target project (default: from `.foostash.yaml` in cwd) |
| `--format, -f` | Output format: `dotenv`, `json`, `table` |
| `--global, -g` | Operate on global secrets |
| `--mask` | Redact secret values in output |
| `--verbose, -v` | Verbose output |

---

## 11. Implementation Roadmap

### Phase 1: Core (Week 1–2)

- [ ] Go module setup, project structure
- [ ] Master key generation and loading (`internal/crypto/key.go`)
- [ ] AES-256-GCM encryption engine (`internal/crypto/engine.go`)
- [ ] Encrypted store: read/write `.enc` files (`internal/store/`)
- [ ] Config loading: `~/.foostash/config.yaml` and `.foostash.yaml`
- [ ] CLI commands: `init`, `set`, `get`, `pull`
- [ ] `foostash run` — exec with injected env vars
- [ ] Global secrets support (`--global` flag)

### Phase 2: Diff & Environments (Week 3)

- [ ] Diff engine (`internal/diff/`)
- [ ] `foostash diff` command with table and JSON output
- [ ] `foostash envs` — create, clone, delete, list
- [ ] `foostash keys` — list keys in an environment
- [ ] Value masking (`--mask` flag)

### Phase 3: History & Import/Export (Week 4)

- [ ] Version history tracking in `.enc` files
- [ ] `foostash history` and `foostash rollback`
- [ ] `foostash import` — from `.env` files
- [ ] `foostash export` — encrypted bundles for sharing
- [ ] `foostash import` — import encrypted bundles

### Phase 4: Polish (Week 5)

- [ ] Comprehensive error messages
- [ ] Shell completions (bash, zsh, fish)
- [ ] `Makefile` with build, test, install targets
- [ ] Cross-platform builds (Linux, macOS, Windows)
- [ ] README with installation instructions and usage examples
- [ ] Homebrew formula
- [ ] Release automation (GoReleaser)

### Future

- [ ] Team sharing via encrypted bundles + asymmetric keys
- [ ] Git hooks integration (prevent `.env` commits)
- [ ] Secret rotation reminders (local cron/notification)
- [ ] SDK mode (import as Go library)
- [ ] Server mode (optional, for team use)

---

## 12. Tech Stack

| Component | Technology | Reason |
|---|---|---|
| Language | Go 1.22 | Single binary, cross-platform |
| CLI framework | Cobra | Standard for Go CLIs |
| Encryption | AES-256-GCM (stdlib) | Zero dependencies, industry standard |
| Config | YAML (`gopkg.in/yaml.v3`) | Human-readable, familiar |
| Storage | Local encrypted files | No dependencies, works offline |
| Build | GoReleaser | Cross-platform binaries + Homebrew |

---

## 13. Competitive Positioning

| Tool | Gap Foostash Fills |
|---|---|
| `.env` files | No encryption, no versioning, no diff |
| direnv | No encryption, no history, no cross-env diff |
| SOPS | Complex, tied to cloud KMS, YAML/JSON only |
| age + scripts | Manual, no structure, no diff |
| Doppler | Requires account, server, not local-first |
| Infisical | Heavy, requires server for basic use |
| 1Password CLI | Tied to 1Password subscription |

**Foostash niche:** Developer who wants encrypted `.env` management with zero infrastructure, a single binary, and proper environment diffing.

---

## 14. Open Questions

1. **Encrypted sharing format** — What metadata to include in `.foostash` export bundles? Needs project name, env name, encrypted payload. Should it use asymmetric encryption (recipient's public key) or a shared passphrase?

2. **Git integration** — Should `foostash init` auto-add `.env` and `*.enc` patterns to `.gitignore`? Probably yes.

3. **Conflict resolution** — If two people export/import the same environment, how to merge? Start with last-write-wins, consider three-way merge later.

4. **Max history depth** — Store all versions forever, or cap at N versions per key to keep `.enc` files small? Default to 50, configurable.

5. **Shell integration** — Should there be a `foostash shell` that drops you into a subshell with all secrets loaded? Nice DX but scope creep for MVP.
