BEGIN;

ALTER TABLE contacts
    ADD COLUMN IF NOT EXISTS owner_user_id bigint;

CREATE INDEX IF NOT EXISTS idx_contacts_owner_user_id ON contacts(owner_user_id);

COMMIT;
