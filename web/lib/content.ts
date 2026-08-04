/**
 * Page copy, in one place so it can be edited without touching layout.
 *
 * Every claim here is backed by code in this repo — `internal/sshauth` for
 * SSH auth, `internal/targets` for the deploy targets, `cli/serve.go` for the
 * self-hosted server. Keep it that way: no invented metrics, no aspirational
 * features.
 */

export const REPO = "https://github.com/Omotolani98/foostash";
export const VERSION = "v0.4.13";
export const INSTALL_CMD = "brew install Omotolani98/foostash/foostash";

export const NAV_LINKS = [
  { href: "#features", label: "features" },
  { href: "#architecture", label: "architecture" },
  { href: "#security", label: "security" },
  { href: "#selfhost", label: "self-host" },
  { href: "/docs", label: "docs" },
] as const;

/** Terminal replay script. `cmd` lines type out; the rest print whole. */
export type TermLine = { kind: "cmd" | "ok" | "out"; text: string };

export const TERM_SCRIPT: TermLine[] = [
  { kind: "cmd", text: "foostash init --project myapp" },
  { kind: "ok", text: "created .foostash.yaml  (project: myapp, env: dev)" },
  { kind: "cmd", text: "foostash set DB_HOST=localhost DB_PASSWORD=s3cr3t --env dev" },
  { kind: "ok", text: "2 keys encrypted (AES-256-GCM) — plaintext never left this machine" },
  { kind: "cmd", text: "foostash run --env dev -- go run ./cmd/server" },
  { kind: "out", text: "injecting 2 secrets into environment" },
  { kind: "out", text: "listening on :8080" },
  { kind: "cmd", text: "foostash history DB_PASSWORD" },
  { kind: "out", text: "v2  2026-07-10 14:02  you@acme  set" },
  { kind: "out", text: "v1  2026-07-09 09:41  you@acme  set" },
];

export const FEATURES = [
  {
    title: "zero-knowledge",
    body: "Secret values are encrypted locally with your master key. The server stores ciphertext and metadata — it can never read a value.",
  },
  {
    title: "ssh-key auth",
    body: "No passwords, no tokens to rotate. Your existing ~/.ssh/id_ed25519 signs every request; the server verifies your fingerprint.",
  },
  {
    title: "self-hostable",
    body: "One docker compose up -d gets you a running server and Postgres. No cloud dependency, ever.",
  },
  {
    title: "versioned + rollback",
    body: "Every write is a version. foostash history shows who changed what; foostash rollback restores any prior value.",
  },
  {
    title: "deploy targets",
    body: "Render secrets straight into Docker env files, Kubernetes Secrets, or GitHub Actions — no copy-paste step.",
  },
  {
    title: "team-ready",
    body: "Orgs, admin and developer roles, invite tokens, and a queryable audit log of every action.",
  },
];

export const ARCH_NODES = [
  {
    name: "foostash CLI",
    where: "your laptop / CI",
    note: ["master.key · AES-256-GCM", "plaintext lives only here"],
  },
  {
    name: "foostash serve",
    where: "self-hosted · one container",
    note: ["orgs · users · invites", "audit log · SSH pubkeys"],
  },
  {
    name: "Postgres",
    where: "yours, anywhere",
    note: ["no secret values", "no private keys"],
  },
];

export const ARCH_EDGES = [
  { over: "SSH-signed HTTPS", under: "ciphertext only" },
  { over: "pgx", under: "metadata" },
];

export const SECURITY_ROWS = [
  {
    term: "crypto",
    body: "AES-256-GCM for local secret files and export bundles. A random 12-byte nonce per write; authenticated ciphertext rejects tampering.",
  },
  {
    term: "master key",
    body: "32 bytes at ~/.foostash/master.key (mode 0600) or via FOOSTASH_MASTER_KEY. Losing it means losing local data — there is deliberately no recovery path.",
  },
  {
    term: "wire auth",
    body: "Every request is signed with your SSH private key and verified against your registered public key. Timestamp header gives ±5 min replay protection.",
  },
  {
    term: "server",
    body: "Holds org, user, project, env, invite, and audit metadata plus SSH public keys. Never secret values, never private keys.",
  },
  {
    term: "revocation",
    body: "admin users revoke flips revoked_at on the user row; every subsequent SSH-signed request from that key fails auth.",
  },
];

export const SELFHOST_STEPS = [
  {
    label: "run the server",
    lines: [
      { prompt: true, text: "docker compose up -d" },
      { comment: true, text: "# server + Postgres, done" },
    ],
  },
  {
    label: "register your org",
    lines: [
      { prompt: true, text: "foostash register \\" },
      { text: "  --server http://localhost:8400 \\" },
      { text: "  --email you@example.com \\" },
      { text: '  --org "Acme"' },
    ],
  },
  {
    label: "invite the team",
    lines: [
      { prompt: true, text: "foostash admin invite" },
      { comment: true, text: "# they run: foostash join <token>" },
    ],
  },
] as const;
