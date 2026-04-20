# Command reference

```
Auth & server
  foostash register    Register the first admin for an org
  foostash login       Re-verify and refresh local identity
  foostash join        Consume an invite token
  foostash serve       Run the HTTP API server (self-host)

Projects & environments
  foostash init        Scaffold .foostash.yaml
  foostash envs        List / create / clone / delete environments

Secrets (local, per-project)
  foostash set         Set one or more keys
  foostash get         Read one key
  foostash delete      Remove a key
  foostash keys        List keys in an env
  foostash pull        Dump an env as dotenv to stdout
  foostash run         Run a command with secrets injected
  foostash diff        Compare two envs
  foostash history     Version history for a key
  foostash rollback    Restore a prior version
  foostash import      Load from .env or a .foostash bundle
  foostash export      Write a password-encrypted bundle

Deployment targets
  foostash target docker render             Render secrets as docker env file
  foostash target kubernetes render|apply   Render or apply a K8s Secret
  foostash target github-actions render|push  Render or push GitHub Actions secrets

Admin
  foostash admin invite       Create an invite token
  foostash admin users list   List users in the org
  foostash admin users role   Change a user's role
  foostash admin users revoke Revoke a user
  foostash admin audit        Query the audit log
```

Run `foostash <cmd> --help` for flags.
