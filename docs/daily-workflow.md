# Daily workflow

```bash
# scaffold a project (creates .foostash.yaml, commit it)
foostash init --project myapp

# add secrets to an environment
foostash set DB_HOST=localhost DB_PORT=5432 --env dev
foostash set STRIPE_KEY=sk_test_xxx --env prod --secret

# read them back
foostash get DB_HOST --env dev
foostash keys --env dev

# hydrate a process with them (no .env file written)
foostash run --env dev -- go run ./cmd/server

# or dump to stdout as dotenv
foostash pull --env dev

# compare environments
foostash diff dev prod

# version history + rollback
foostash history DB_HOST --env prod
foostash rollback DB_HOST --env prod --version 2

# environments
foostash envs                       # list
foostash envs create staging        # new env
foostash envs clone dev staging     # copy
```

## Import an existing `.env`

```bash
foostash import .env --env dev
```

## Share an environment out-of-band

```bash
# export a password-protected bundle
foostash export --env prod
# Enter encryption password: ********
# wrote myapp-prod.foostash

# recipient:
foostash import myapp-prod.foostash --env prod
# Enter decryption password: ********
```

## Sync secrets across machines

Add `--remote` to any of `set`, `get`, `pull`, `delete`, `history`, `rollback` to talk to the server instead of (or in addition to) the local encrypted store. Values are encrypted **on your machine** before upload — the server only holds ciphertext and nonces.

```bash
# machine A
foostash set STRIPE_KEY=sk_live_xxx --env prod --remote

# machine B (same ~/.foostash/master.key — copy it over securely)
foostash pull --env prod --remote
foostash history STRIPE_KEY --env prod --remote
foostash rollback STRIPE_KEY --env prod --version 2 --remote
```

Any authenticated org member (admin or developer) can read/write secrets for projects in their org. Without `--remote`, commands operate on the local `.enc` files only.
