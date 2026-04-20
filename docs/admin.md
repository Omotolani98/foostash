# Admin operations

All admin commands talk to the server and require an admin-role SSH key.

```bash
# list all users in your org
foostash admin users list

# promote or demote
foostash admin users role <user-id> admin
foostash admin users role <user-id> developer

# revoke access (immediate; the user's next request returns 401 user_revoked)
foostash admin users revoke <user-id>

# query the audit log
foostash admin audit --since 24h
foostash admin audit --action project.create
foostash admin audit --user <user-id> --limit 50
```

## Audited actions

`invite.create`, `user.change_role`, `user.revoke`, `project.create`, `project.delete`, `env.create`, `env.clone`, `env.delete`.
