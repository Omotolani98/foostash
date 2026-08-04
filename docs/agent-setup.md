# Agent setup

Foostash ships a local MCP server so coding agents can set up Foostash without seeing secret values.

```json
{
  "mcpServers": {
    "foostash": {
      "command": "foostash",
      "args": ["mcp"]
    }
  }
}
```

The MCP server runs on your machine over stdio. It reuses the same local files as the CLI:

- `~/.foostash/config.yaml` for server URL and SSH identity
- `~/.foostash/master.key` or `FOOSTASH_MASTER_KEY` for local encryption
- `.foostash.yaml` in the project directory for project/env selection

## Tools

- `inspect` — report local config, project config, master-key presence, and optional server health.
- `server_setup_plan` — plan a local Docker Compose Foostash server install. No files are written.
- `server_setup_apply` — apply a previously generated plan, start Docker Compose, and verify `/v1/health`.
- `identity_setup` — register a new org or join with an invite using a local SSH key.
- `project_setup` — create or reconnect a project, ensure environments, and write `.foostash.yaml`.
- `environment_sync` — read a local dotenv file, encrypt changed values locally, and bulk-upload ciphertext.

## Safety model

- The agent never receives private keys, master keys, invite tokens in tool output, or plaintext secret values.
- `environment_sync` accepts a file path, not raw secret values, and returns counts plus key names only.
- Server setup defaults to `127.0.0.1:8400`. If you bind publicly, terminate TLS in front of Foostash.
- Destructive operations such as deletes, rollbacks, user revocation, and role changes are intentionally not exposed in the first MCP release.

## Example prompts

```text
Use Foostash to inspect this project and tell me what's missing.
```

```text
Plan a local Foostash server setup, then ask before applying it.
```

```text
Create a Foostash project named payments-api with dev, staging, and prod environments.
```

```text
Sync .env.local into the dev Foostash environment. Do not print secret values.
```
