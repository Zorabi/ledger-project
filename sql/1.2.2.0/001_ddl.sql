BEGIN;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM dictionary_items
        WHERE deleted_at IS NULL
          AND enabled = true
          AND family_id IS NULL
        GROUP BY type_id, value
        HAVING count(*) > 1
    ) THEN
        RAISE EXCEPTION 'dictionary_items has duplicate enabled system values; clean duplicates before applying 1.2.2.0 DDL';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM dictionary_items
        WHERE deleted_at IS NULL
          AND enabled = true
          AND family_id IS NOT NULL
        GROUP BY type_id, family_id, value
        HAVING count(*) > 1
    ) THEN
        RAISE EXCEPTION 'dictionary_items has duplicate enabled family values; clean duplicates before applying 1.2.2.0 DDL';
    END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS idx_dictionary_items_system_type_value_unique
    ON dictionary_items(type_id, value)
    WHERE family_id IS NULL AND deleted_at IS NULL AND enabled = true;

CREATE UNIQUE INDEX IF NOT EXISTS idx_dictionary_items_family_type_value_unique
    ON dictionary_items(type_id, family_id, value)
    WHERE family_id IS NOT NULL AND deleted_at IS NULL AND enabled = true;

COMMIT;
