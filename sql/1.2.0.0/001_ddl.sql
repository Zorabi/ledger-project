BEGIN;

CREATE TABLE IF NOT EXISTS schema_migrations (
    version varchar(30) PRIMARY KEY,
    description varchar(255) NOT NULL,
    applied_at timestamptz NOT NULL DEFAULT now()
);

COMMIT;
