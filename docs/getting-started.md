# Getting started

## Register the first admin

The first user to call `/v1/auth/register` against a fresh server becomes the org's admin. All subsequent users join via invite.

```bash
ssh-keygen -t ed25519 -f ~/.ssh/id_ed25519 -C "you@example.com"

foostash register \
  --server http://localhost:8400 \
  --email you@example.com \
  --org "Acme"
# registered org=Acme email=you@example.com role=admin
```

What happened:

1. The CLI signed the request with your SSH private key.
2. The server created the org, your user (role `admin`), and stored your public key.
3. `~/.foostash/config.yaml` now carries the server URL and your key fingerprint — future commands don't need `--server`.

A master key is generated on first encrypt at `~/.foostash/master.key` (mode `0600`). **Back this up.** Losing it means losing local-encrypted secrets.

## Invite teammates

Admins generate tokens; invitees consume them.

```bash
# admin
foostash admin invite --email alice@acme.com --role developer
# invite created (expires 2026-04-21T12:00:00Z)
#
# Share this with alice@acme.com:
#   foostash join <token>
```

```bash
# invitee — must have ~/.ssh/id_ed25519 present
foostash join <token> --server http://localhost:8400
# joined org=Acme email=alice@acme.com role=developer
```

Roles:

- **admin** — can create projects/envs, invite users, revoke users, read the audit log.
- **developer** — can read project/env metadata and manage secrets on their own machine.

Next: [daily workflow](daily-workflow.md).
