# Security model

- **Crypto.** AES-256-GCM for local secret files and password-protected export bundles. 12-byte random nonce per write; authenticated ciphertext rejects tampering.
- **Master key.** 32 bytes, base64-encoded, stored at `~/.foostash/master.key` (mode `0600`) or passed via `FOOSTASH_MASTER_KEY`. Losing it means losing local data — no recovery path.
- **Wire auth.** Every request is signed with your SSH private key. The server verifies against the public key it registered for your fingerprint. Replay protection via a timestamp header (±5 min window).
- **Server.** Holds org/user/project/env/invite/audit metadata and SSH public keys. Does not hold secret values or private keys.
- **Revocation.** `admin users revoke` flips `revoked_at` on the user row; subsequent SSH-signed requests fail auth.
