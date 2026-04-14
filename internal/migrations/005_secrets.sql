CREATE TABLE IF NOT EXISTS secrets (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    env_id      UUID NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    key         VARCHAR(255) NOT NULL,
    ciphertext  BYTEA NOT NULL,
    nonce       BYTEA NOT NULL,
    version     INT NOT NULL DEFAULT 1,
    updated_by  UUID REFERENCES users(id),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_secrets_env_key_live
    ON secrets(env_id, key)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_secrets_env ON secrets(env_id);

CREATE TABLE IF NOT EXISTS secret_history (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    env_id      UUID NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    key         VARCHAR(255) NOT NULL,
    version     INT NOT NULL,
    ciphertext  BYTEA NOT NULL,
    nonce       BYTEA NOT NULL,
    set_by      UUID REFERENCES users(id),
    set_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(env_id, key, version)
);

CREATE INDEX IF NOT EXISTS idx_secret_history_env_key
    ON secret_history(env_id, key, version DESC);
