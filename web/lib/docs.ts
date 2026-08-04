/**
 * Docs content. Mirrors the markdown in the repo's docs/ directory — when a
 * doc changes there, update the matching page here. Sample outputs (version
 * strings etc.) intentionally match the repo docs verbatim.
 */

export type DocBlock =
  | { t: "h2"; text: string }
  | { t: "p"; text: string }
  | { t: "code"; lang: string; code: string }
  | { t: "note"; text: string }
  | { t: "ul"; items: { term: string; text: string }[] }
  | { t: "table"; head: string[]; rows: string[][] };

export type DocPage = {
  slug: string;
  group: "Start" | "Use" | "Operate" | "Reference";
  title: string;
  source: string;
  lede: string;
  blocks: DocBlock[];
};

const REPO = "https://github.com/Omotolani98/foostash";

export const DOC_PAGES: DocPage[] = [
  {
    slug: "install",
    group: "Start",
    title: "Install",
    source: `${REPO}/blob/dev/docs/install.md`,
    lede: "Foostash is a single Go binary. Homebrew is the shortest path; source and release archives work everywhere else.",
    blocks: [
      { t: "h2", text: "Homebrew (macOS / Linux)" },
      { t: "code", lang: "bash", code: "brew install Omotolani98/foostash/foostash" },
      { t: "p", text: "Or tap first, then install:" },
      { t: "code", lang: "bash", code: "brew tap Omotolani98/foostash\nbrew install foostash" },
      { t: "p", text: "Upgrade with brew upgrade foostash." },
      { t: "h2", text: "From source" },
      {
        t: "code",
        lang: "bash",
        code: "go install github.com/Omotolani98/foostash/cmd/foostash@latest\nfoostash -v",
      },
      { t: "note", text: "Requires Go 1.25 or newer." },
      { t: "h2", text: "From a release binary" },
      {
        t: "p",
        text: "Download a prebuilt archive for your OS/arch from Releases, extract it, and place foostash on your PATH.",
      },
      {
        t: "code",
        lang: "bash",
        code: "# example: macOS arm64\ncurl -L https://github.com/Omotolani98/foostash/releases/latest/download/foostash_*_darwin_arm64.tar.gz | tar xz\nsudo mv foostash /usr/local/bin/",
      },
      { t: "h2", text: "Verify" },
      { t: "code", lang: "bash", code: "foostash -v\n# foostash v0.2.3 (commit …, built …)" },
    ],
  },
  {
    slug: "self-hosting",
    group: "Start",
    title: "Self-host the server",
    source: `${REPO}/blob/dev/docs/self-hosting.md`,
    lede: "Your server, your Postgres, your keys. The repo ships a compose file wiring the published GHCR image to Postgres.",
    blocks: [
      { t: "h2", text: "Quick start with Docker Compose" },
      {
        t: "code",
        lang: "bash",
        code: 'git clone https://github.com/Omotolani98/foostash.git\ncd foostash\n\ndocker compose pull\ndocker compose up -d\n\ncurl http://localhost:8400/v1/health\n# {"status":"ok","version":"v0.2.3","database":"ok"}',
      },
      { t: "h2", text: "Defaults" },
      {
        t: "table",
        head: ["Setting", "Value"],
        rows: [
          ["API port", "8400 (host and container)"],
          ["Postgres", "postgres:16-alpine · user/pass/db foostash"],
          ["Data volume", "foostash-pgdata (Docker-managed)"],
          ["Image", "ghcr.io/omotolani98/foostash:latest"],
          ["DB URL", "postgres://foostash:foostash@postgres:5432/foostash?sslmode=disable"],
        ],
      },
      { t: "h2", text: "Pin a release, or build locally" },
      {
        t: "code",
        lang: "bash",
        code: "FOOSTASH_IMAGE_TAG=v0.2.2 docker compose up -d\n\ndocker compose up -d --build",
      },
      { t: "h2", text: "Run without Docker" },
      {
        t: "code",
        lang: "bash",
        code: 'export FOOSTASH_PG_URL="postgres://foostash:foostash@localhost:5432/foostash?sslmode=disable"\nmake build\n./bin/foostash serve --addr :8400',
      },
      { t: "p", text: "Makefile helpers: make docker-build, make docker-up, make docker-down." },
      { t: "h2", text: "Operational notes" },
      {
        t: "ul",
        items: [
          {
            term: "Migrations run at startup. ",
            text: "Every foostash serve applies pending SQL migrations idempotently — no separate migrate step.",
          },
          {
            term: "Healthcheck. ",
            text: "GET /v1/health pings the DB with a 2s timeout. Compose uses it for container health; load balancers can too.",
          },
          {
            term: "Data lifecycle. ",
            text: "docker compose down keeps the foostash-pgdata volume. Use down -v to wipe it.",
          },
          {
            term: "TLS. ",
            text: "The server speaks plain HTTP. Terminate TLS in front (Caddy, nginx, Cloudflare, a managed LB) before exposing it to the internet.",
          },
        ],
      },
    ],
  },
  {
    slug: "getting-started",
    group: "Start",
    title: "Getting started",
    source: `${REPO}/blob/dev/docs/getting-started.md`,
    lede: "Register the first admin, then bring the rest of the team in with invite tokens.",
    blocks: [
      { t: "h2", text: "Register the first admin" },
      {
        t: "p",
        text: "The first user to call /v1/auth/register against a fresh server becomes the org admin. Everyone else joins via invite.",
      },
      {
        t: "code",
        lang: "bash",
        code: 'ssh-keygen -t ed25519 -f ~/.ssh/id_ed25519 -C "you@example.com"\n\nfoostash register \\\n  --server http://localhost:8400 \\\n  --email you@example.com \\\n  --org "Acme"\n# registered org=Acme email=you@example.com role=admin',
      },
      {
        t: "ul",
        items: [
          { term: "1. ", text: "The CLI signs the request with your SSH private key." },
          {
            term: "2. ",
            text: "The server creates the org, your user (role admin), and stores your public key.",
          },
          {
            term: "3. ",
            text: "~/.foostash/config.yaml now carries the server URL and key fingerprint — future commands do not need --server.",
          },
        ],
      },
      {
        t: "note",
        text: "A master key is generated on first encrypt at ~/.foostash/master.key (mode 0600). Back it up — losing it means losing local-encrypted secrets.",
      },
      { t: "h2", text: "Invite teammates" },
      {
        t: "code",
        lang: "bash",
        code: "# admin\nfoostash admin invite --email alice@acme.com --role developer\n# invite created (expires 2026-04-21T12:00:00Z)\n#\n# Share this with alice@acme.com:\n#   foostash join <token>",
      },
      {
        t: "code",
        lang: "bash",
        code: "# invitee — must have ~/.ssh/id_ed25519 present\nfoostash join <token> --server http://localhost:8400\n# joined org=Acme email=alice@acme.com role=developer",
      },
      { t: "h2", text: "Roles" },
      {
        t: "ul",
        items: [
          {
            term: "admin — ",
            text: "create projects and envs, invite users, revoke users, write the shared vault, read the audit log.",
          },
          {
            term: "developer — ",
            text: "read project/env metadata, manage secrets on their own machine, read the shared vault.",
          },
        ],
      },
    ],
  },
  {
    slug: "daily-workflow",
    group: "Use",
    title: "Daily workflow",
    source: `${REPO}/blob/dev/docs/daily-workflow.md`,
    lede: "The commands you will actually type every day: set, run, diff, history, rollback.",
    blocks: [
      {
        t: "code",
        lang: "bash",
        code: "# scaffold a project (creates .foostash.yaml, commit it)\nfoostash init --project myapp\n\n# add secrets to an environment\nfoostash set DB_HOST=localhost DB_PORT=5432 --env dev\nfoostash set STRIPE_KEY=sk_test_xxx --env prod --secret\n\n# read them back\nfoostash get DB_HOST --env dev\nfoostash keys --env dev\n\n# hydrate a process (no .env file written)\nfoostash run --env dev -- go run ./cmd/server\n\n# or dump to stdout as dotenv\nfoostash pull --env dev\n\n# compare environments\nfoostash diff dev prod\n\n# version history + rollback\nfoostash history DB_HOST --env prod\nfoostash rollback DB_HOST --env prod --version 2\n\n# environments\nfoostash envs\nfoostash envs create staging\nfoostash envs clone dev staging",
      },
      { t: "h2", text: "Import an existing .env" },
      { t: "code", lang: "bash", code: "foostash import .env --env dev" },
      { t: "h2", text: "Share an environment out-of-band" },
      {
        t: "code",
        lang: "bash",
        code: "# export a password-protected bundle\nfoostash export --env prod\n# Enter encryption password: ********\n# wrote myapp-prod.foostash\n\n# recipient:\nfoostash import myapp-prod.foostash --env prod\n# Enter decryption password: ********",
      },
      { t: "h2", text: "Sync secrets across machines" },
      {
        t: "p",
        text: "Add --remote to any of set, get, pull, delete, history, rollback to talk to the server instead of the local store. Values are encrypted on your machine before upload — the server only holds ciphertext and nonces.",
      },
      {
        t: "code",
        lang: "bash",
        code: "# machine A\nfoostash set STRIPE_KEY=sk_live_xxx --env prod --remote\n\n# machine B (same ~/.foostash/master.key — copy it over securely)\nfoostash pull --env prod --remote\nfoostash history STRIPE_KEY --env prod --remote\nfoostash rollback STRIPE_KEY --env prod --version 2 --remote",
      },
      {
        t: "note",
        text: "Any authenticated org member can read/write secrets for projects in their org. Without --remote, commands operate on local .enc files only.",
      },
    ],
  },
  {
    slug: "vault",
    group: "Use",
    title: "Shared vault",
    source: `${REPO}/blob/dev/cli/vault.go`,
    lede: "Org-wide secrets that are not tied to a project — CI tokens, registry credentials, shared API keys. All members read; admins write.",
    blocks: [
      { t: "h2", text: "Read the vault" },
      {
        t: "code",
        lang: "bash",
        code: "foostash vault list\nfoostash vault list --values\nfoostash vault get GHCR_TOKEN",
      },
      { t: "h2", text: "Write (admin-only)" },
      {
        t: "code",
        lang: "bash",
        code: "foostash vault set GHCR_TOKEN=ghp_xxx SENTRY_DSN=https://…\n\n# hidden input instead of shell history\nfoostash vault set GHCR_TOKEN --secret\n\nfoostash vault delete SENTRY_DSN",
      },
      { t: "h2", text: "History and rollback" },
      {
        t: "code",
        lang: "bash",
        code: "foostash vault history GHCR_TOKEN\n#   Version   Value                           Set At\n#   ──────────────────────────────────────────────────\n#   2         ghp_live_…                      2026-07-10 14:02:11\n#   1         ghp_old_…                       2026-06-02 09:18:44\n\nfoostash vault rollback GHCR_TOKEN --version 1",
      },
      {
        t: "ul",
        items: [
          {
            term: "Encryption. ",
            text: "Same as project secrets — values are encrypted locally with your master key before upload.",
          },
          {
            term: "Keys. ",
            text: "Every vault command accepts --ssh-key to point at a private key other than ~/.ssh/id_ed25519.",
          },
        ],
      },
    ],
  },
  {
    slug: "targets",
    group: "Use",
    title: "Deployment targets",
    source: `${REPO}/blob/dev/docs/commands.md`,
    lede: "Render an environment straight into the format your platform expects, instead of copy-pasting into a dashboard.",
    blocks: [
      {
        t: "code",
        lang: "bash",
        code: "# docker env file\nfoostash target docker render --env prod > .env.prod\n\n# kubernetes Secret\nfoostash target kubernetes render --env prod\nfoostash target kubernetes apply  --env prod\n\n# github actions repo secrets\nfoostash target github-actions render --env prod\nfoostash target github-actions push   --env prod",
      },
      {
        t: "note",
        text: "render prints to stdout so you can review or pipe it; apply and push talk to the platform directly.",
      },
    ],
  },
  {
    slug: "agent-setup",
    group: "Use",
    title: "Agent setup (MCP)",
    source: `${REPO}/blob/dev/docs/agent-setup.md`,
    lede: "Foostash ships a local MCP server so coding agents can set up Foostash without ever seeing a secret value.",
    blocks: [
      {
        t: "code",
        lang: "json",
        code: '{\n  "mcpServers": {\n    "foostash": {\n      "command": "foostash",\n      "args": ["mcp"]\n    }\n  }\n}',
      },
      {
        t: "p",
        text: "The MCP server runs on your machine over stdio and reuses the same local files as the CLI: ~/.foostash/config.yaml, the master key, and .foostash.yaml in the project directory.",
      },
      { t: "h2", text: "Tools" },
      {
        t: "ul",
        items: [
          {
            term: "inspect — ",
            text: "report local config, project config, master-key presence, and optional server health.",
          },
          {
            term: "server_setup_plan — ",
            text: "plan a local Docker Compose server install. No files are written.",
          },
          {
            term: "server_setup_apply — ",
            text: "apply a generated plan, start Docker Compose, and verify /v1/health.",
          },
          {
            term: "identity_setup — ",
            text: "register a new org or join with an invite using a local SSH key.",
          },
          {
            term: "project_setup — ",
            text: "create or reconnect a project, ensure environments, and write .foostash.yaml.",
          },
          {
            term: "environment_sync — ",
            text: "read a local dotenv file, encrypt changed values locally, and bulk-upload ciphertext.",
          },
        ],
      },
      { t: "h2", text: "Safety model" },
      {
        t: "ul",
        items: [
          {
            term: "",
            text: "The agent never receives private keys, master keys, invite tokens, or plaintext secret values.",
          },
          {
            term: "",
            text: "environment_sync accepts a file path, not raw values, and returns counts plus key names only.",
          },
          {
            term: "",
            text: "Server setup defaults to 127.0.0.1:8400. If you bind publicly, terminate TLS in front.",
          },
          {
            term: "",
            text: "Deletes, rollbacks, revocation, and role changes are intentionally not exposed in the first MCP release.",
          },
        ],
      },
      { t: "h2", text: "Example prompts" },
      {
        t: "code",
        lang: "text",
        code: "Use Foostash to inspect this project and tell me what's missing.\n\nPlan a local Foostash server setup, then ask before applying it.\n\nCreate a Foostash project named payments-api with dev, staging, and prod environments.\n\nSync .env.local into the dev Foostash environment. Do not print secret values.",
      },
    ],
  },
  {
    slug: "sdk-go",
    group: "Use",
    title: "Go SDK",
    source: `${REPO}/blob/dev/sdk/go/README.md`,
    lede: "Read-only SDK for loading and watching secrets at runtime, without shipping the CLI binary alongside your service.",
    blocks: [
      { t: "code", lang: "bash", code: "go get github.com/Omotolani98/foostash/sdk/go@latest" },
      {
        t: "p",
        text: "The SDK lives inside the main repo under sdk/go/ but is released with independent subdirectory tags (e.g. sdk/go/v0.1.0).",
      },
      { t: "h2", text: "Usage" },
      {
        t: "code",
        lang: "go",
        code: 'client, err := foostash.New(foostash.Config{\n    ServerURL:  "https://foostash.example.com",\n    Project:    "payments-api",\n    Env:        "prod",\n    SSHKeyPath: "/var/run/secrets/foostash-key",\n    MasterKey:  os.Getenv("FOOSTASH_MASTER_KEY"),\n})\nif err != nil {\n    log.Fatal(err)\n}\n\nctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)\ndefer cancel()\n\nsecrets, err := client.Pull(ctx)',
      },
      { t: "h2", text: "Watching for changes" },
      {
        t: "code",
        lang: "go",
        code: "ch, err := client.Watch(ctx, 30*time.Second)\nif err != nil {\n    log.Fatal(err)\n}\nfor snap := range ch {\n    reloadConfig(snap.Secrets)\n}",
      },
      {
        t: "p",
        text: "Watch polls on the given interval (clamped to a 5-second floor) and emits a Snapshot whenever a key version changes. The initial snapshot is emitted synchronously, so the first receive returns your current config. Transient network failures do not close the channel — attach a *slog.Logger with foostash.WithLogger(l) to observe them.",
      },
      { t: "h2", text: "Requirements" },
      {
        t: "ul",
        items: [
          {
            term: "An SSH key registered with the server. ",
            text: "Use the CLI once to provision a service-account key, then mount the unencrypted private key. Passphrase-protected keys are not supported — they are for humans, not long-running services.",
          },
          {
            term: "The org master key. ",
            text: "Base64, 32 bytes. Distribute via KMS, env var, or file mount. The SDK never reads ~/.foostash/.",
          },
        ],
      },
      { t: "h2", text: "Errors" },
      {
        t: "table",
        head: ["Sentinel", "Meaning"],
        rows: [
          ["foostash.ErrNotFound", "404 from the server"],
          ["foostash.ErrUnauthorized", "401"],
          ["foostash.ErrForbidden", "403"],
          ["foostash.ErrInvalidKeyLength", "master key did not decode to 32 bytes"],
        ],
      },
      {
        t: "note",
        text: "v1 is read-only: Get, Pull, Watch. Writes, history, and the vault namespace stay on the CLI for now.",
      },
    ],
  },
  {
    slug: "admin",
    group: "Operate",
    title: "Admin operations",
    source: `${REPO}/blob/dev/docs/admin.md`,
    lede: "All admin commands talk to the server and require an admin-role SSH key.",
    blocks: [
      {
        t: "code",
        lang: "bash",
        code: "# list all users in your org\nfoostash admin users list\n\n# promote or demote\nfoostash admin users role <user-id> admin\nfoostash admin users role <user-id> developer\n\n# revoke access (immediate; next request returns 401 user_revoked)\nfoostash admin users revoke <user-id>\n\n# query the audit log\nfoostash admin audit --since 24h\nfoostash admin audit --action project.create\nfoostash admin audit --user <user-id> --limit 50",
      },
      { t: "h2", text: "Audited actions" },
      {
        t: "code",
        lang: "text",
        code: "invite.create      user.change_role   user.revoke\nproject.create     project.delete\nenv.create         env.clone          env.delete",
      },
    ],
  },
  {
    slug: "configuration",
    group: "Operate",
    title: "Configuration",
    source: `${REPO}/blob/dev/docs/configuration.md`,
    lede: "Two YAML files and a handful of environment variables. That is the whole surface.",
    blocks: [
      { t: "h2", text: "Global config — ~/.foostash/config.yaml" },
      {
        t: "p",
        text: "Written by foostash register / foostash join. Edit only if you know what you are doing.",
      },
      {
        t: "code",
        lang: "yaml",
        code: "version: 1\ndefaults:\n  env: dev\n  format: dotenv\nserver: http://localhost:8400\nidentity:\n  email: you@example.com\n  ssh_key_path: /home/you/.ssh/id_ed25519\n  key_fingerprint: SHA256:AbCdEf...",
      },
      { t: "h2", text: "Project config — .foostash.yaml" },
      { t: "p", text: "Commit this one." },
      {
        t: "code",
        lang: "yaml",
        code: "project: myapp\ndefault_env: dev\nenvironments:\n  - dev\n  - staging\n  - prod",
      },
      { t: "h2", text: "Client environment variables" },
      {
        t: "table",
        head: ["Var", "Purpose"],
        rows: [
          [
            "FOOSTASH_MASTER_KEY",
            "Overrides the master key file. Base64-encoded 32 bytes. Do not mix with .enc files encrypted under a different key.",
          ],
        ],
      },
      { t: "h2", text: "Server environment variables" },
      {
        t: "table",
        head: ["Var", "Default", "Purpose"],
        rows: [
          [
            "FOOSTASH_PG_URL",
            "postgres://foostash:foostash@localhost:5432/foostash?sslmode=disable",
            "Postgres connection URL",
          ],
          ["FOOSTASH_LISTEN_ADDR", ":8400", "HTTP listen address"],
        ],
      },
    ],
  },
  {
    slug: "commands",
    group: "Reference",
    title: "Command reference",
    source: `${REPO}/blob/dev/docs/commands.md`,
    lede: "Every command in the binary. Run foostash <cmd> --help for flags.",
    blocks: [
      { t: "h2", text: "Auth & server" },
      {
        t: "table",
        head: ["Command", "Does"],
        rows: [
          ["foostash register", "Register the first admin for an org"],
          ["foostash login", "Re-verify and refresh local identity"],
          ["foostash join", "Consume an invite token"],
          ["foostash serve", "Run the HTTP API server (self-host)"],
          ["foostash mcp", "Run the local MCP server for coding agents"],
        ],
      },
      { t: "h2", text: "Projects & environments" },
      {
        t: "table",
        head: ["Command", "Does"],
        rows: [
          ["foostash init", "Scaffold .foostash.yaml"],
          ["foostash envs", "List / create / clone / delete environments"],
        ],
      },
      { t: "h2", text: "Secrets" },
      {
        t: "table",
        head: ["Command", "Does"],
        rows: [
          ["foostash set", "Set one or more keys"],
          ["foostash get", "Read one key"],
          ["foostash delete", "Remove a key"],
          ["foostash keys", "List keys in an env"],
          ["foostash pull", "Dump an env as dotenv to stdout"],
          ["foostash run", "Run a command with secrets injected"],
          ["foostash diff", "Compare two envs"],
          ["foostash history", "Version history for a key"],
          ["foostash rollback", "Restore a prior version"],
          ["foostash import", "Load from .env or a .foostash bundle"],
          ["foostash export", "Write a password-encrypted bundle"],
        ],
      },
      { t: "h2", text: "Shared vault" },
      {
        t: "table",
        head: ["Command", "Does"],
        rows: [
          ["foostash vault list", "List org-wide vault secrets"],
          ["foostash vault get", "Read a vault secret"],
          ["foostash vault set", "Set vault secrets (admin-only)"],
          ["foostash vault delete", "Delete a vault secret (admin-only)"],
          ["foostash vault history", "Version history for a vault key"],
          ["foostash vault rollback", "Roll a vault key back (admin-only)"],
        ],
      },
      { t: "h2", text: "Deployment targets" },
      {
        t: "table",
        head: ["Command", "Does"],
        rows: [
          ["foostash target docker render", "Render secrets as a docker env file"],
          ["foostash target kubernetes render|apply", "Render or apply a K8s Secret"],
          ["foostash target github-actions render|push", "Render or push GitHub Actions secrets"],
        ],
      },
      { t: "h2", text: "Admin" },
      {
        t: "table",
        head: ["Command", "Does"],
        rows: [
          ["foostash admin invite", "Create an invite token"],
          ["foostash admin users list", "List users in the org"],
          ["foostash admin users role", "Change a user's role"],
          ["foostash admin users revoke", "Revoke a user"],
          ["foostash admin audit", "Query the audit log"],
        ],
      },
    ],
  },
  {
    slug: "security",
    group: "Reference",
    title: "Security model",
    source: `${REPO}/blob/dev/docs/security.md`,
    lede: "Short enough to actually read, which is the point.",
    blocks: [
      {
        t: "ul",
        items: [
          {
            term: "Crypto. ",
            text: "AES-256-GCM for local secret files and password-protected export bundles. 12-byte random nonce per write; authenticated ciphertext rejects tampering.",
          },
          {
            term: "Master key. ",
            text: "32 bytes, base64-encoded, at ~/.foostash/master.key (mode 0600) or via FOOSTASH_MASTER_KEY. Losing it means losing local data — no recovery path.",
          },
          {
            term: "Wire auth. ",
            text: "Every request is signed with your SSH private key and verified against the registered public key for your fingerprint. Replay protection via a timestamp header (±5 min window).",
          },
          {
            term: "Server. ",
            text: "Holds org/user/project/env/invite/audit metadata and SSH public keys. Does not hold secret values or private keys.",
          },
          {
            term: "Revocation. ",
            text: "admin users revoke flips revoked_at on the user row; subsequent SSH-signed requests fail auth.",
          },
        ],
      },
      { t: "h2", text: "What the server can and cannot see" },
      {
        t: "table",
        head: ["Data", "On your machine", "On the server"],
        rows: [
          ["Secret values", "plaintext + ciphertext", "ciphertext only"],
          ["Master key", "yes (0600)", "never"],
          ["SSH private key", "yes", "never"],
          ["SSH public key", "yes", "yes"],
          ["Key names, versions, timestamps", "yes", "yes"],
          ["Org / user / audit metadata", "no", "yes"],
        ],
      },
    ],
  },
  {
    slug: "development",
    group: "Reference",
    title: "Development",
    source: `${REPO}/blob/dev/docs/development.md`,
    lede: "Build, test, and cut releases.",
    blocks: [
      {
        t: "code",
        lang: "bash",
        code: "make build         # bin/foostash with version ldflags\nmake test          # go test ./... -v\nmake vet           # go vet ./...\nmake run-server    # go run ./cmd/foostash-server\nmake sdk-test      # go test ./... inside sdk/go",
      },
      { t: "h2", text: "Repo layout" },
      {
        t: "code",
        lang: "text",
        code: "cmd/\n  foostash/          unified CLI + `foostash serve`\n  foostash-server/   server-only entrypoint\ncli/                 Cobra commands\ninternal/\n  crypto/            AES-GCM engine, master key, password-derived keys\n  sshauth/           SSH signing / verification\n  store/             local encrypted secret files\n  secrets/           local service for set/get/versioning\n  envs/              local environment manager\n  diff/              env comparison\n  runner/            process launcher with env injection\n  targets/           docker, k8s, gh-actions renderers\n  mcpserver/         local MCP server for coding agents\n  tui/               terminal UI (projects, secrets, vault, users, audit)\n  billing/           plan limits and enforcement\n  config/            YAML config load/save\n  apiclient/         HTTP client with SSH-signed requests\n  repo/              pgx repositories\n  service/           server-side domain logic\n  server/            Chi router, middleware, handlers\n  migrations/        embedded SQL, applied at startup\nsdk/go/              read-only Go SDK",
      },
      { t: "h2", text: "Release process" },
      {
        t: "p",
        text: "Releases are cut with GoReleaser via a GitHub Actions workflow on tag push. It builds cross-platform binaries, publishes the GitHub release, and updates the Homebrew tap formula.",
      },
      {
        t: "code",
        lang: "bash",
        code: "git tag v0.3.0\ngit push origin v0.3.0\n\n# SDK is tagged independently\ngit tag sdk/go/v0.1.0\ngit push origin sdk/go/v0.1.0",
      },
    ],
  },
];

export const DOC_GROUPS = ["Start", "Use", "Operate", "Reference"] as const;

export function anchorFor(text: string): string {
  return text
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-|-$/g, "");
}

/** Lowercased haystack per page, for the sidebar's client-side filter. */
export function searchIndex() {
  return DOC_PAGES.map((p) => ({
    slug: p.slug,
    title: p.title,
    group: p.group,
    haystack: `${p.title} ${p.lede} ${JSON.stringify(p.blocks)}`.toLowerCase(),
  }));
}
