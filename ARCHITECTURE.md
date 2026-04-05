# Foostash — Architecture Document

> An open-source, developer-first secrets and environment manager.
> CLI + SDK · Self-hostable · Paid dashboard

---

## 1. Product Overview

**Foostash** replaces `.env` files with a centralized, encrypted secrets store that developers interact with via CLI and SDKs. Secrets are organized by **project → environment** (e.g., `myapp/staging`), versioned, and access-controlled.

### Value Proposition

| Pain Point | Foostash Solution |
|---|---|
| `.env` files committed to git | Secrets stored remotely, pulled at runtime |
| No audit trail | Full access log: who pulled what, when |
| Sharing secrets over Slack/email | Invite teammates, scoped access per environment |
| Different flows per environment | One CLI command: `foostash pull --env prod` |
| No rotation visibility | Version history + rotation reminders |

### Business Model: Open-Core

| Layer | What's Included | Pricing |
|---|---|---|
| **Core (OSS)** | Go API server, CLI, SDKs, self-host via Docker | Free forever |
| **Dashboard (Paid)** | Web UI, audit logs, team management, usage analytics | $4–10/mo per team |
| **Cloud (Hosted)** | Managed hosting of the full stack | $10+/mo |

---

## 2. User Workflows

### Developer Workflow (CLI)

```
$ foostash login
  → Authenticates via API key or browser OAuth

$ foostash init
  → Links current directory to a Foostash project (writes .foostash.yaml)

$ foostash set DB_HOST=localhost DB_PORT=5432 --env dev
  → Encrypts and stores secrets for the dev environment

$ foostash pull --env dev
  → Prints secrets to stdout (pipe to .env, source, or inject)

$ foostash pull --env dev --format dotenv > .env
  → Writes a .env file (gitignored)

$ foostash run --env dev -- go run main.go
  → Injects secrets as env vars and runs the command

$ foostash diff dev staging
  → Shows which keys differ between environments

$ foostash logs --env prod --last 24h
  → (Paid) Shows access audit log
```

### Developer Workflow (SDK)

```go
// Go SDK
import "github.com/foostash/foostash-go"

func main() {
    fs := foostash.New(foostash.Config{
        ProjectID: "myapp",
        Env:       "prod",
        // Token from FOOSTASH_TOKEN env var or config file
    })

    dbHost, err := fs.Get("DB_HOST")
    // or
    secrets, err := fs.GetAll()  // map[string]string
}
```

```java
// Java SDK (Spring Boot starter)
// application.yml:
// foostash:
//   project-id: myapp
//   env: prod
//   token: ${FOOSTASH_TOKEN}

@Value("${DB_HOST}")
private String dbHost;  // Injected via Foostash PropertySource
```

### Team Lead Workflow (Dashboard)

```
1. Create project "payments-api"
2. Define environments: dev, staging, prod
3. Invite teammates → assign roles (admin, developer, read-only)
4. View audit log: who accessed which secrets, when
5. Set rotation reminders for API keys
6. View usage: SDK pulls per day, active environments
```

---

## 3. System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        CLIENTS                                  │
│                                                                 │
│   ┌──────────┐    ┌──────────┐    ┌──────────┐   ┌───────────┐ │
│   │   CLI    │    │  Go SDK  │    │ Java SDK │   │  Web UI   │ │
│   │  (Go)    │    │          │    │ (Spring) │   │  (React)  │ │
│   └────┬─────┘    └────┬─────┘    └────┬─────┘   └─────┬─────┘ │
│        │               │               │               │       │
└────────┼───────────────┼───────────────┼───────────────┼───────┘
         │               │               │               │
         └───────────────┴───────┬───────┴───────────────┘
                                 │ HTTPS / API Key or JWT
                                 ▼
┌─────────────────────────────────────────────────────────────────┐
│                     API SERVER (Go + Chi)                       │
│                                                                 │
│   ┌────────────────┐  ┌────────────────┐  ┌──────────────────┐ │
│   │  Auth          │  │  Secrets       │  │  Audit           │ │
│   │  Middleware     │  │  Handler       │  │  Middleware       │ │
│   │  (API key/JWT) │  │  (CRUD + pull) │  │  (log every req) │ │
│   └────────────────┘  └────────────────┘  └──────────────────┘ │
│                                                                 │
│   ┌────────────────┐  ┌────────────────┐  ┌──────────────────┐ │
│   │  Projects      │  │  Environments  │  │  Team/RBAC       │ │
│   │  Handler       │  │  Handler       │  │  Handler (Paid)  │ │
│   └────────────────┘  └────────────────┘  └──────────────────┘ │
│                                                                 │
│   ┌───────────────────────────────────────────────────────────┐ │
│   │  Encryption Layer (AES-256-GCM)                           │ │
│   │  - Encrypt before write, decrypt on read                  │ │
│   │  - Master key from env var or KMS                         │ │
│   └───────────────────────────────────────────────────────────┘ │
└──────────────────────────┬──────────────────────────────────────┘
                           │
              ┌────────────┴────────────┐
              ▼                         ▼
   ┌──────────────────┐     ┌──────────────────┐
   │   PostgreSQL     │     │   Redis          │
   │                  │     │                  │
   │  - projects      │     │  - session cache │
   │  - environments  │     │  - rate limiting │
   │  - secrets       │     │  - API key cache │
   │  - audit_logs    │     │                  │
   │  - users/teams   │     │                  │
   │  - api_keys      │     │                  │
   └──────────────────┘     └──────────────────┘
```

### Deployment Topology

```
Self-hosted (Docker Compose):
  docker compose up -d
  → foostash-server (Go binary)
  → postgres
  → redis

Cloud-hosted (Managed):
  → Kubernetes on Hetzner (existing infra)
  → Managed Postgres (or self-hosted)
  → Redis
  → Dashboard served via CDN
```

---

## 4. Data Model

All database access is raw SQL via `database/sql` + `pgx` driver. No ORM.

### Schema

```sql
-- Extensions
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Core entities

CREATE TABLE organizations (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(100) NOT NULL,
    plan        VARCHAR(20) NOT NULL DEFAULT 'free',  -- free | team | cloud
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE users (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    email       VARCHAR(255) NOT NULL,
    password    VARCHAR(255),                          -- bcrypt hash, nullable for OAuth users
    role        VARCHAR(20) NOT NULL DEFAULT 'developer', -- admin | developer | readonly
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(org_id, email)
);

CREATE TABLE projects (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    slug        VARCHAR(100) NOT NULL,                 -- e.g. "payments-api"
    name        VARCHAR(255) NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(org_id, slug)
);

CREATE TABLE environments (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    slug        VARCHAR(50) NOT NULL,                  -- e.g. "dev", "staging", "prod"
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(project_id, slug)
);

CREATE TABLE secrets (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    environment_id  UUID NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    key             VARCHAR(255) NOT NULL,             -- e.g. "DB_HOST"
    encrypted_value BYTEA NOT NULL,                    -- AES-256-GCM ciphertext
    nonce           BYTEA NOT NULL,                    -- 12-byte GCM nonce, unique per value
    version         INTEGER NOT NULL DEFAULT 1,
    created_by      UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(environment_id, key)
);

CREATE TABLE secret_versions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    secret_id       UUID NOT NULL REFERENCES secrets(id) ON DELETE CASCADE,
    version         INTEGER NOT NULL,
    encrypted_value BYTEA NOT NULL,
    nonce           BYTEA NOT NULL,
    created_by      UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE api_keys (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    org_id          UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    key_hash        VARCHAR(255) NOT NULL,             -- SHA-256 of the raw key
    key_prefix      VARCHAR(12) NOT NULL,              -- "fst_" + first 8 chars, for display
    name            VARCHAR(100),                      -- human label, e.g. "CI deploy key"
    scopes          TEXT[] NOT NULL DEFAULT '{}',       -- {"secrets:read","secrets:write"}
    environment_ids UUID[] DEFAULT NULL,               -- restrict to specific envs, NULL = all
    last_used_at    TIMESTAMPTZ,
    expires_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Paid tier

CREATE TABLE audit_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id          UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id         UUID REFERENCES users(id) ON DELETE SET NULL,
    action          VARCHAR(50) NOT NULL,              -- "secrets.pull", "secrets.set", "secrets.delete"
    project_slug    VARCHAR(100),
    environment_slug VARCHAR(50),
    secret_key      VARCHAR(255),                      -- which key was accessed (never the value)
    ip_address      INET,
    user_agent      VARCHAR(500),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Indexes

CREATE INDEX idx_secrets_env_id ON secrets(environment_id);
CREATE INDEX idx_secret_versions_secret_id ON secret_versions(secret_id);
CREATE INDEX idx_api_keys_hash ON api_keys(key_hash);
CREATE INDEX idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX idx_audit_logs_org_created ON audit_logs(org_id, created_at DESC);
CREATE INDEX idx_audit_logs_project ON audit_logs(org_id, project_slug, created_at DESC);
CREATE INDEX idx_users_email ON users(email);
```

### Migration Strategy

Plain SQL migration files in `migrations/`, applied in order. No migration framework — just a `schema_migrations` table tracking applied versions.

```
migrations/
  001_initial_schema.up.sql
  001_initial_schema.down.sql
  002_add_audit_logs.up.sql
  002_add_audit_logs.down.sql
```

Applied via a simple `migrate` subcommand:

```
$ foostash migrate up
$ foostash migrate down 1
```

### Repository Pattern (Raw SQL)

Each domain has a repository struct that takes a `*sql.DB` and exposes query methods. No interfaces at this stage — keep it concrete until there's a real reason to abstract.

```go
// internal/store/secrets.go

type SecretRow struct {
    ID             string
    EnvironmentID  string
    Key            string
    EncryptedValue []byte
    Nonce          []byte
    Version        int
    CreatedBy      *string
    CreatedAt      time.Time
    UpdatedAt      time.Time
}

type SecretsStore struct {
    db *sql.DB
}

func NewSecretsStore(db *sql.DB) *SecretsStore {
    return &SecretsStore{db: db}
}

func (s *SecretsStore) GetByEnvironment(ctx context.Context, envID string) ([]SecretRow, error) {
    rows, err := s.db.QueryContext(ctx,
        `SELECT id, environment_id, key, encrypted_value, nonce, version, created_by, created_at, updated_at
         FROM secrets
         WHERE environment_id = $1
         ORDER BY key`, envID)
    if err != nil {
        return nil, fmt.Errorf("query secrets: %w", err)
    }
    defer rows.Close()

    var secrets []SecretRow
    for rows.Next() {
        var r SecretRow
        if err := rows.Scan(&r.ID, &r.EnvironmentID, &r.Key, &r.EncryptedValue,
            &r.Nonce, &r.Version, &r.CreatedBy, &r.CreatedAt, &r.UpdatedAt); err != nil {
            return nil, fmt.Errorf("scan secret: %w", err)
        }
        secrets = append(secrets, r)
    }
    return secrets, rows.Err()
}

func (s *SecretsStore) Upsert(ctx context.Context, envID, key string, encVal, nonce []byte, userID string) error {
    _, err := s.db.ExecContext(ctx,
        `INSERT INTO secrets (environment_id, key, encrypted_value, nonce, version, created_by)
         VALUES ($1, $2, $3, $4, 1, $5)
         ON CONFLICT (environment_id, key) DO UPDATE
         SET encrypted_value = EXCLUDED.encrypted_value,
             nonce = EXCLUDED.nonce,
             version = secrets.version + 1,
             updated_at = now()`, envID, key, encVal, nonce, userID)
    return err
}
```

---

## 5. API Design

Base URL: `https://api.foostash.dev/v1` (cloud) or `http://localhost:8400/v1` (self-hosted)

### Authentication

```
Authorization: Bearer fst_abc123...   (API key)
Authorization: Bearer eyJhbG...       (JWT for dashboard sessions)
```

API keys are hashed with SHA-256 before storage. On each request, the server hashes the incoming key and looks it up in `api_keys.key_hash`. Redis caches the resolved key → user/org/scopes mapping with a short TTL (60s) to avoid hitting Postgres on every request.

### Router Setup (Chi)

```go
func NewRouter(deps *Dependencies) *chi.Mux {
    r := chi.NewRouter()

    // Global middleware
    r.Use(middleware.RequestID)
    r.Use(middleware.RealIP)
    r.Use(middleware.Logger)
    r.Use(middleware.Recoverer)
    r.Use(middleware.Timeout(30 * time.Second))

    // Health
    r.Get("/healthz", deps.Health.Check)

    // Public
    r.Post("/v1/auth/register", deps.Auth.Register)
    r.Post("/v1/auth/login", deps.Auth.Login)

    // Authenticated routes
    r.Group(func(r chi.Router) {
        r.Use(deps.Auth.Middleware)  // validates API key or JWT

        // API keys
        r.Post("/v1/auth/api-keys", deps.Auth.CreateAPIKey)
        r.Get("/v1/auth/api-keys", deps.Auth.ListAPIKeys)
        r.Delete("/v1/auth/api-keys/{keyID}", deps.Auth.RevokeAPIKey)

        // Projects
        r.Get("/v1/projects", deps.Projects.List)
        r.Post("/v1/projects", deps.Projects.Create)
        r.Get("/v1/projects/{slug}", deps.Projects.Get)
        r.Delete("/v1/projects/{slug}", deps.Projects.Delete)

        // Environments
        r.Get("/v1/projects/{slug}/envs", deps.Environments.List)
        r.Post("/v1/projects/{slug}/envs", deps.Environments.Create)
        r.Delete("/v1/projects/{slug}/envs/{env}", deps.Environments.Delete)

        // Secrets
        r.Get("/v1/projects/{slug}/envs/{env}/secrets", deps.Secrets.Pull)
        r.Post("/v1/projects/{slug}/envs/{env}/secrets", deps.Secrets.Set)
        r.Put("/v1/projects/{slug}/envs/{env}/secrets/{key}", deps.Secrets.Update)
        r.Delete("/v1/projects/{slug}/envs/{env}/secrets/{key}", deps.Secrets.Delete)
        r.Get("/v1/projects/{slug}/envs/{env}/secrets/{key}/versions", deps.Secrets.Versions)

        // Diff
        r.Get("/v1/projects/{slug}/envs/{env}/diff/{otherEnv}", deps.Secrets.Diff)

        // Paid: Audit
        r.Get("/v1/audit-logs", deps.Audit.List)

        // Paid: Team
        r.Post("/v1/teams/invite", deps.Teams.Invite)
        r.Put("/v1/teams/{userID}/role", deps.Teams.UpdateRole)

        // Paid: Usage
        r.Get("/v1/usage", deps.Usage.Summary)
    })

    return r
}
```

### Request/Response Shapes

```
POST /v1/projects/{slug}/envs/{env}/secrets
Content-Type: application/json

{
  "secrets": {
    "DB_HOST": "prod-db.internal",
    "DB_PORT": "5432",
    "STRIPE_KEY": "sk_live_..."
  }
}

→ 200 OK
{
  "project": "payments-api",
  "environment": "prod",
  "set": ["DB_HOST", "DB_PORT", "STRIPE_KEY"],
  "version": 12
}
```

```
GET /v1/projects/{slug}/envs/{env}/secrets

→ 200 OK
{
  "project": "payments-api",
  "environment": "prod",
  "secrets": {
    "DB_HOST": "prod-db.internal",
    "DB_PORT": "5432",
    "STRIPE_KEY": "sk_live_..."
  },
  "version": 12,
  "pulled_at": "2026-04-05T10:30:00Z"
}
```

```
GET /v1/projects/{slug}/envs/{env}/diff/{otherEnv}

→ 200 OK
{
  "project": "payments-api",
  "left": "dev",
  "right": "staging",
  "only_left": ["DEBUG_MODE"],
  "only_right": ["SENTRY_DSN"],
  "different_values": ["DB_HOST", "DB_PORT"],
  "identical": ["APP_NAME"]
}
```

### Error Shape

All errors use a consistent JSON envelope:

```json
{
  "error": {
    "code": "not_found",
    "message": "Project 'payments-api' not found",
    "status": 404
  }
}
```

Standard codes: `bad_request`, `unauthorized`, `forbidden`, `not_found`, `conflict`, `rate_limited`, `internal`.

---

## 6. Domain Layer

The domain layer sits between HTTP handlers and the store. Handlers parse requests and call services. Services contain business logic, encryption, and audit logging. Stores execute SQL.

```
Handler (HTTP) → Service (business logic) → Store (SQL)
                       ↓
                 Crypto (encrypt/decrypt)
                       ↓
                 Audit (log access)
```

### Service Example

```go
// internal/service/secrets.go

type SecretsService struct {
    secrets *store.SecretsStore
    envs    *store.EnvironmentsStore
    crypto  *crypto.Engine
    audit   *AuditService
}

func (s *SecretsService) Pull(ctx context.Context, orgID, projectSlug, envSlug string) (map[string]string, error) {
    env, err := s.envs.GetByProjectAndSlug(ctx, orgID, projectSlug, envSlug)
    if err != nil {
        return nil, err
    }

    rows, err := s.secrets.GetByEnvironment(ctx, env.ID)
    if err != nil {
        return nil, err
    }

    result := make(map[string]string, len(rows))
    for _, row := range rows {
        plaintext, err := s.crypto.Decrypt(row.EncryptedValue, row.Nonce)
        if err != nil {
            return nil, fmt.Errorf("decrypt %s: %w", row.Key, err)
        }
        result[row.Key] = string(plaintext)
    }

    // Audit log (fire-and-forget, don't block the response)
    s.audit.LogAsync(ctx, AuditEntry{
        Action:       "secrets.pull",
        ProjectSlug:  projectSlug,
        EnvSlug:      envSlug,
    })

    return result, nil
}

func (s *SecretsService) Set(ctx context.Context, orgID, projectSlug, envSlug string, secrets map[string]string, userID string) error {
    env, err := s.envs.GetByProjectAndSlug(ctx, orgID, projectSlug, envSlug)
    if err != nil {
        return err
    }

    for key, value := range secrets {
        ciphertext, nonce, err := s.crypto.Encrypt([]byte(value))
        if err != nil {
            return fmt.Errorf("encrypt %s: %w", key, err)
        }
        if err := s.secrets.Upsert(ctx, env.ID, key, ciphertext, nonce, userID); err != nil {
            return fmt.Errorf("upsert %s: %w", key, err)
        }
    }

    s.audit.LogAsync(ctx, AuditEntry{
        Action:       "secrets.set",
        ProjectSlug:  projectSlug,
        EnvSlug:      envSlug,
    })

    return nil
}
```

### Handler Example

```go
// internal/handler/secrets.go

type SecretsHandler struct {
    service *service.SecretsService
}

func (h *SecretsHandler) Pull(w http.ResponseWriter, r *http.Request) {
    slug := chi.URLParam(r, "slug")
    env := chi.URLParam(r, "env")
    auth := AuthFromContext(r.Context())  // populated by auth middleware

    secrets, err := h.service.Pull(r.Context(), auth.OrgID, slug, env)
    if err != nil {
        WriteError(w, err)
        return
    }

    WriteJSON(w, http.StatusOK, PullResponse{
        Project:     slug,
        Environment: env,
        Secrets:     secrets,
        PulledAt:    time.Now().UTC(),
    })
}
```

---

## 7. Encryption Design

```
┌─────────────────────────────────────────────────┐
│ Encryption Hierarchy                            │
│                                                 │
│   MASTER KEY (env var or file)                  │
│       │                                         │
│       └── Used directly for AES-256-GCM         │
│           - Each secret value gets unique nonce  │
│           - Nonce stored alongside ciphertext   │
│           - Master key NEVER in the database    │
│                                                 │
│   Future: per-org keys encrypted by master key  │
└─────────────────────────────────────────────────┘
```

### Crypto Engine

```go
// internal/crypto/engine.go

type Engine struct {
    aead cipher.AEAD
}

func NewEngine(masterKeyBase64 string) (*Engine, error) {
    key, err := base64.StdEncoding.DecodeString(masterKeyBase64)
    if err != nil {
        return nil, fmt.Errorf("decode master key: %w", err)
    }
    if len(key) != 32 {
        return nil, fmt.Errorf("master key must be 32 bytes, got %d", len(key))
    }

    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, err
    }
    aead, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }

    return &Engine{aead: aead}, nil
}

func (e *Engine) Encrypt(plaintext []byte) (ciphertext, nonce []byte, err error) {
    nonce = make([]byte, e.aead.NonceSize()) // 12 bytes
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return nil, nil, err
    }
    ciphertext = e.aead.Seal(nil, nonce, plaintext, nil)
    return ciphertext, nonce, nil
}

func (e *Engine) Decrypt(ciphertext, nonce []byte) ([]byte, error) {
    return e.aead.Open(nil, nonce, ciphertext, nil)
}
```

### Master Key Configuration

```bash
# Option 1: Environment variable
export FOOSTASH_MASTER_KEY="base64-encoded-32-byte-key"

# Option 2: File (for Docker secrets, k8s secrets)
export FOOSTASH_MASTER_KEY_FILE="/run/secrets/foostash-master-key"

# Generate a key
openssl rand -base64 32
```

### Key Rotation (Future)

Re-encrypt all secrets with a new master key via a CLI command:

```
$ foostash admin rotate-key --new-key-file /path/to/new.key
```

This reads every secret, decrypts with old key, re-encrypts with new key, and updates in a transaction.

---

## 8. Auth Design

### API Key Flow (CLI + SDKs)

```
1. User logs in via dashboard or `foostash login`
2. User generates API key via dashboard or `foostash auth create-key --name "CI"`
3. Server generates random 32-byte key, returns raw key once: fst_a1b2c3d4e5f6...
4. Server stores SHA-256(key) in api_keys.key_hash
5. Client stores raw key in ~/.foostash/config.yaml
6. On each request: client sends key in Authorization header
7. Server hashes incoming key, looks up in api_keys, resolves user/org/scopes
8. Redis caches hash → auth context for 60s
```

### JWT Flow (Dashboard)

```
1. User POSTs email + password to /v1/auth/login
2. Server verifies bcrypt hash
3. Server returns JWT (signed with HMAC-SHA256, 24h expiry)
4. Dashboard sends JWT in Authorization header
5. Auth middleware verifies signature + expiry, extracts user/org from claims
```

### Auth Middleware

```go
// internal/middleware/auth.go

func AuthMiddleware(keys *store.APIKeysStore, cache *redis.Client, jwtSecret []byte) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            token := extractBearerToken(r)
            if token == "" {
                WriteError(w, ErrUnauthorized)
                return
            }

            var auth AuthContext
            var err error

            if strings.HasPrefix(token, "fst_") {
                auth, err = resolveAPIKey(r.Context(), token, keys, cache)
            } else {
                auth, err = resolveJWT(token, jwtSecret)
            }

            if err != nil {
                WriteError(w, ErrUnauthorized)
                return
            }

            ctx := context.WithValue(r.Context(), authContextKey, auth)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

### Scope Enforcement

Scopes are checked in handlers, not middleware. Middleware resolves identity; handlers decide authorization.

```go
func (h *SecretsHandler) Set(w http.ResponseWriter, r *http.Request) {
    auth := AuthFromContext(r.Context())
    if !auth.HasScope("secrets:write") {
        WriteError(w, ErrForbidden)
        return
    }
    // ...
}
```

---

## 9. Project Structure (Monorepo)

```
foostash/
├── cmd/
│   ├── server/
│   │   └── main.go              # API server entrypoint
│   └── cli/
│       └── main.go              # CLI entrypoint
│
├── internal/
│   ├── config/
│   │   └── config.go            # Env-based config loading
│   │
│   ├── crypto/
│   │   ├── engine.go            # AES-256-GCM encrypt/decrypt
│   │   └── engine_test.go
│   │
│   ├── store/
│   │   ├── db.go                # *sql.DB setup, connection pooling
│   │   ├── organizations.go     # Org queries
│   │   ├── users.go             # User queries
│   │   ├── projects.go          # Project queries
│   │   ├── environments.go      # Environment queries
│   │   ├── secrets.go           # Secret queries (upsert, get, versions)
│   │   ├── apikeys.go           # API key queries
│   │   └── audit.go             # Audit log queries
│   │
│   ├── service/
│   │   ├── auth.go              # Register, login, API key lifecycle
│   │   ├── secrets.go           # Pull, set, diff — calls store + crypto
│   │   ├── projects.go          # Project + environment CRUD
│   │   ├── audit.go             # Async audit logging
│   │   └── teams.go             # Invite, role management (paid)
│   │
│   ├── handler/
│   │   ├── auth.go              # HTTP handlers for auth endpoints
│   │   ├── secrets.go           # HTTP handlers for secrets endpoints
│   │   ├── projects.go          # HTTP handlers for project endpoints
│   │   ├── audit.go             # HTTP handlers for audit endpoints
│   │   ├── teams.go             # HTTP handlers for team endpoints
│   │   ├── health.go            # /healthz
│   │   └── helpers.go           # WriteJSON, WriteError, parse helpers
│   │
│   ├── middleware/
│   │   ├── auth.go              # API key + JWT resolution
│   │   ├── audit.go             # Per-request audit logging
│   │   └── ratelimit.go         # Redis-based rate limiting
│   │
│   └── router/
│       └── router.go            # Chi router wiring
│
├── cli/                          # CLI command implementations
│   ├── root.go                  # Cobra root command
│   ├── login.go
│   ├── init.go
│   ├── set.go
│   ├── pull.go
│   ├── run.go
│   ├── diff.go
│   ├── logs.go
│   ├── migrate.go
│   └── client/
│       └── http.go              # HTTP client for talking to Foostash server
│
├── sdk/
│   └── go/                      # Public Go SDK (separate go module)
│       ├── go.mod               # module github.com/foostash/foostash-go
│       ├── foostash.go          # Client struct, New(), Get(), GetAll()
│       ├── config.go            # Config from env vars or explicit
│       └── cache.go             # Optional in-memory TTL cache
│
├── web/                          # React dashboard (separate package)
│   ├── package.json
│   ├── src/
│   └── ...
│
├── migrations/
│   ├── 001_initial_schema.up.sql
│   └── 001_initial_schema.down.sql
│
├── docker-compose.yml            # Server + Postgres + Redis
├── Dockerfile                    # Multi-stage build for server binary
├── Makefile
├── go.mod                        # module github.com/foostash/foostash
├── go.sum
├── ARCHITECTURE.md
├── README.md
└── LICENSE
```

### Module Boundaries

The monorepo has two Go modules:

```
github.com/foostash/foostash       # server + CLI (root go.mod)
github.com/foostash/foostash-go    # public SDK (sdk/go/go.mod)
```

The server and CLI share `internal/` packages directly (same binary or same module). The SDK is a standalone module that talks to the server over HTTP — it has zero dependency on `internal/`.

---

## 10. CLI Design

### Config Files

```yaml
# ~/.foostash/config.yaml (global, created by `foostash login`)
server: "https://api.foostash.dev"  # or self-hosted URL
token: "fst_a1b2c3d4..."

# .foostash.yaml (per project, committed to git)
project: payments-api
default_env: dev
```

### CLI ↔ Server Relationship

The CLI is a **remote client** in normal use — it talks to the server over HTTP, same as the SDK. However, the CLI binary also embeds the `migrate` subcommand which operates directly on the database (for self-hosted setup).

```
foostash login       → hits server /v1/auth/login
foostash pull        → hits server /v1/projects/{slug}/envs/{env}/secrets
foostash set         → hits server /v1/projects/{slug}/envs/{env}/secrets
foostash run         → pulls secrets, then exec's the child process
foostash diff        → hits server /v1/projects/{slug}/envs/{env}/diff/{other}
foostash migrate     → connects directly to Postgres (self-hosted admin only)
```

### CLI HTTP Client

```go
// cli/client/http.go

type Client struct {
    base   string // e.g. "https://api.foostash.dev"
    token  string
    http   *http.Client
}

func (c *Client) Pull(projectSlug, env string) (map[string]string, error) {
    url := fmt.Sprintf("%s/v1/projects/%s/envs/%s/secrets", c.base, projectSlug, env)
    req, _ := http.NewRequest("GET", url, nil)
    req.Header.Set("Authorization", "Bearer "+c.token)

    resp, err := c.http.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        return nil, parseError(resp)
    }

    var result PullResponse
    json.NewDecoder(resp.Body).Decode(&result)
    return result.Secrets, nil
}
```

---

## 11. Dashboard (Paid Tier)

**Stack:** React + Tailwind + shadcn/ui

### Pages

```
/login                → Auth (email + password)
/projects             → List projects, create new
/projects/:slug       → Project overview, list environments
/projects/:slug/:env  → Secret manager (key-value editor table)
/projects/:slug/audit → Audit log with filters (paid)
/settings/team        → Invite members, manage roles, API keys (paid)
/settings/billing     → Plan management via Lemon Squeezy (paid)
/usage                → Charts: pulls/day, SDK breakdown (paid)
```

### Paid Differentiators

1. **Audit Log Viewer** — filterable table: who pulled what, when, from where
2. **Team Management** — invite by email, assign roles per project/env
3. **Usage Analytics** — daily pull counts, SDK breakdown, environment activity
4. **Rotation Reminders** — set expiry on secrets, email/webhook alerts
5. **Secret Comparison** — side-by-side diff of two environments in the UI

---

## 12. Feature Tiers

| Feature | Free (OSS) | Team ($4/mo) | Cloud ($10/mo) |
|---|---|---|---|
| Projects | 3 | Unlimited | Unlimited |
| Environments per project | 3 | Unlimited | Unlimited |
| Secrets per environment | 50 | Unlimited | Unlimited |
| CLI | ✓ | ✓ | ✓ |
| SDKs (Go, Java) | ✓ | ✓ | ✓ |
| Self-host | ✓ | ✓ | ✓ |
| API access | ✓ | ✓ | ✓ |
| Version history | Last 5 | Unlimited | Unlimited |
| Dashboard | Basic (CRUD only) | Full | Full |
| Audit logs | — | 30 days | 90 days |
| Team members | 1 | 10 | 25 |
| RBAC | — | ✓ | ✓ |
| Usage analytics | — | ✓ | ✓ |
| Rotation reminders | — | ✓ | ✓ |
| Managed hosting | — | — | ✓ |
| SLA | — | — | 99.9% |

### Tier Enforcement

Free tier limits are enforced at the service layer, not the database. Before creating a project, the service checks the org's plan and current count:

```go
func (s *ProjectsService) Create(ctx context.Context, orgID, slug, name string) error {
    org, _ := s.orgs.GetByID(ctx, orgID)
    count, _ := s.projects.CountByOrg(ctx, orgID)

    limit := planLimits[org.Plan].MaxProjects // free=3, team=unlimited
    if limit > 0 && count >= limit {
        return ErrPlanLimitReached
    }
    // ...
}
```

---

## 13. Docker & Self-Hosting

### docker-compose.yml

```yaml
version: "3.8"

services:
  foostash:
    image: ghcr.io/foostash/foostash:latest
    ports:
      - "8400:8400"
    environment:
      FOOSTASH_DB_URL: "postgres://foostash:foostash@postgres:5432/foostash?sslmode=disable"
      FOOSTASH_REDIS_URL: "redis://redis:6379"
      FOOSTASH_MASTER_KEY: "${FOOSTASH_MASTER_KEY}"
      FOOSTASH_JWT_SECRET: "${FOOSTASH_JWT_SECRET}"
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_started

  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: foostash
      POSTGRES_PASSWORD: foostash
      POSTGRES_DB: foostash
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U foostash"]
      interval: 5s
      timeout: 3s
      retries: 5

  redis:
    image: redis:7-alpine
    volumes:
      - redisdata:/data

volumes:
  pgdata:
  redisdata:
```

### Dockerfile

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o foostash ./cmd/server

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/foostash /usr/local/bin/foostash
COPY --from=builder /app/migrations /migrations
EXPOSE 8400
ENTRYPOINT ["foostash"]
CMD ["serve"]
```

### Server Config

All config via environment variables. No config files for the server.

```go
// internal/config/config.go

type Config struct {
    Port          int    `env:"FOOSTASH_PORT" default:"8400"`
    DatabaseURL   string `env:"FOOSTASH_DB_URL" required:"true"`
    RedisURL      string `env:"FOOSTASH_REDIS_URL" required:"true"`
    MasterKey     string `env:"FOOSTASH_MASTER_KEY"`
    MasterKeyFile string `env:"FOOSTASH_MASTER_KEY_FILE"`
    JWTSecret     string `env:"FOOSTASH_JWT_SECRET" required:"true"`
    MigrationsDir string `env:"FOOSTASH_MIGRATIONS_DIR" default:"/migrations"`
}
```

---

## 14. Implementation Roadmap

### Phase 1: Core (Weeks 1–3) — Open Source Launch

- [ ] Go module setup, project structure
- [ ] Config loading from env vars
- [ ] Postgres connection, raw SQL migrations runner
- [ ] Initial schema migration
- [ ] Crypto engine (AES-256-GCM)
- [ ] Store layer: orgs, users, projects, environments, secrets
- [ ] Service layer: auth (register/login/API keys), secrets (pull/set)
- [ ] Handler layer: Chi router, auth middleware, secrets endpoints
- [ ] CLI: login, init, set, pull, run
- [ ] Docker Compose + Dockerfile
- [ ] README with quickstart

### Phase 2: Polish + SDK (Weeks 4–5)

- [ ] Secret versioning (store + endpoint)
- [ ] `foostash diff` command + endpoint
- [ ] Go SDK (`github.com/foostash/foostash-go`) with TTL cache
- [ ] Java/Spring Boot SDK starter
- [ ] Rate limiting middleware (Redis)
- [ ] Basic dashboard: login, project/env/secret CRUD
- [ ] Error handling improvements, input validation

### Phase 3: Paid Features (Weeks 6–8)

- [ ] Audit logging middleware + store + endpoint
- [ ] Team invites + RBAC enforcement
- [ ] Dashboard: audit log viewer, usage analytics charts
- [ ] Tier enforcement in service layer
- [ ] Lemon Squeezy integration for payments
- [ ] License key validation for self-hosted paid features

### Phase 4: Growth (Ongoing)

- [ ] Node SDK, Python SDK
- [ ] GitHub Actions integration (pull secrets in CI)
- [ ] Webhook on secret change
- [ ] `foostash import .env` command
- [ ] Secret rotation automation
- [ ] Per-org encryption keys
- [ ] E2E encryption option
- [ ] Docs site (Docusaurus or Mintlify)

---

## 15. Tech Stack Summary

| Component | Technology | Reason |
|---|---|---|
| API Server | Go 1.22 + Chi | Fast, small binary, matches learning goals |
| Database | PostgreSQL 16 | Reliable, known well |
| DB Access | `database/sql` + `pgx` | No ORM, raw SQL, full control |
| Migrations | Plain SQL files | No framework dependency |
| Cache | Redis 7 | Rate limiting, API key cache |
| Encryption | AES-256-GCM (stdlib) | Industry standard, zero dependencies |
| CLI | Go + Cobra | Standard for Go CLIs |
| Dashboard | React + Tailwind | Fast to build |
| Payments | Lemon Squeezy | Simple for indie SaaS, handles tax |
| Hosting | Hetzner + Kubernetes | Existing infra |
| CI/CD | GitHub Actions | Free for open source |
| Container | Docker + multi-stage | Small production image |

---

## 16. Competitive Positioning

| Product | Gap Foostash Fills |
|---|---|
| HashiCorp Vault | Overkill for small teams, complex setup |
| Doppler | Closed-source, $20/seat, no self-host |
| Infisical | Good but heavier, more enterprise-focused |
| .env files | No encryption, no audit, no sharing, no versioning |
| AWS Secrets Manager | Vendor lock-in, IAM complexity |
| 1Password Secrets | Tied to 1Password ecosystem |

**Foostash niche:** Developer who wants Doppler's DX, Infisical's open-source ethos, but in a leaner package they can self-host with one `docker compose up`.

---

## 17. Open Questions

1. **E2E encryption** — Should secrets be decryptable only by the client, not the server? Strong differentiator but adds key management complexity to every client. Start without, add as paid premium feature later.

2. **Secret references / templating** — `DB_URL = postgres://{{DB_USER}}:{{DB_PASS}}@{{DB_HOST}}`. Nice DX but adds parser complexity. V2 candidate.

3. **Webhook on change** — Notify services to hot-reload when a secret changes. Useful but not MVP.

4. **Multi-region** — For cloud tier, replicate secrets across regions. Way post-MVP.

5. **CLI plugin system** — Let users extend with custom commands. Overkill for now.

6. **Import/export** — `foostash import .env` and `foostash export --format json`. Low effort, high value. Should be Phase 2.