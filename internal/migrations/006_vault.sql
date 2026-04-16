CREATE TABLE IF NOT EXISTS vault_secrets (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    key         VARCHAR(255) NOT NULL,
    ciphertext  BYTEA NOT NULL,
    nonce       BYTEA NOT NULL,
    version     INT NOT NULL DEFAULT 1,
    updated_by  UUID REFERENCES users(id),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_vault_secrets_org_key_live
    ON vault_secrets(org_id, key)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_vault_secrets_org ON vault_secrets(org_id);

CREATE TABLE IF NOT EXISTS vault_secret_history (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    key         VARCHAR(255) NOT NULL,
    version     INT NOT NULL,
    ciphertext  BYTEA NOT NULL,
    nonce       BYTEA NOT NULL,
    set_by      UUID REFERENCES users(id),
    set_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(org_id, key, version)
);

CREATE INDEX IF NOT EXISTS idx_vault_secret_history_org_key
    ON vault_secret_history(org_id, key, version DESC);
