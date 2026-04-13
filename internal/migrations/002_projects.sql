CREATE TABLE IF NOT EXISTS projects (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    slug        VARCHAR(100) NOT NULL,
    name        VARCHAR(255) NOT NULL,
    created_by  UUID REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(org_id, slug)
);

CREATE TABLE IF NOT EXISTS environments (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    slug        VARCHAR(50) NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(project_id, slug)
);

CREATE INDEX IF NOT EXISTS idx_projects_org      ON projects(org_id);
CREATE INDEX IF NOT EXISTS idx_environments_proj ON environments(project_id);
