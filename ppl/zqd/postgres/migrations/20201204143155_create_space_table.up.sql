CREATE TABLE IF NOT EXISTS space(
    id text PRIMARY KEY,
    data_uri text,
    name text NOT NULL UNIQUE,
    parent_id text references space(id),
    storage json
);
