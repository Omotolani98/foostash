CREATE TABLE IF NOT EXISTS audit_log (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id         UUID        NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id        UUID        REFERENCES users(id) ON DELETE SET NULL,
    action         VARCHAR(64) NOT NULL,
    resource_type  VARCHAR(64),
    resource_id    VARCHAR(255),
    status         INTEGER     NOT NULL,
    request_ip     INET,
    metadata       JSONB,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_audit_org_created ON audit_log(org_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_user        ON audit_log(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_action      ON audit_log(action);
