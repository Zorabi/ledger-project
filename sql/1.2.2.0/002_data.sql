BEGIN;

INSERT INTO schema_migrations(version, description)
VALUES ('1.2.2.0', 'enforce dictionary custom permissions and value uniqueness')
ON CONFLICT (version) DO NOTHING;

COMMIT;
