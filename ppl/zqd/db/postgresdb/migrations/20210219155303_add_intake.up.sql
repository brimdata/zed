CREATE TABLE IF NOT EXISTS intake(
    id text PRIMARY KEY,
    name text NOT NULL,
    tenant_id text NOT NULL,
    shaper text,
    target_space_id text,
    UNIQUE(tenant_id, name)
);
