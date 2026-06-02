BEGIN;

INSERT INTO schema_migrations(version, description)
VALUES ('1.3.0.0', 'add contact owner member dimension')
ON CONFLICT (version) DO NOTHING;

COMMIT;
