ALTER TABLE space DROP CONSTRAINT space_name_key;
ALTER TABLE space ADD CONSTRAINT space_name_key UNIQUE (name);
ALTER TABLE space DROP tenant_id;
