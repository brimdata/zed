ALTER TABLE space ADD tenant_id text;
ALTER TABLE space DROP CONSTRAINT space_name_key;
ALTER TABLE space ADD CONSTRAINT space_name_key UNIQUE (tenant_id, name);
