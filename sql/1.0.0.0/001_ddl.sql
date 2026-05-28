BEGIN;

CREATE TABLE IF NOT EXISTS families (
    id bigserial PRIMARY KEY,
    name varchar(120) NOT NULL,
    contact_name varchar(80),
    contact_email varchar(160),
    remark text,
    status varchar(20) NOT NULL DEFAULT 'enabled',
    admin_user_id bigint,
    created_at timestamptz,
    updated_at timestamptz,
    deleted_at timestamptz
);
CREATE INDEX IF NOT EXISTS idx_families_status ON families(status);
CREATE INDEX IF NOT EXISTS idx_families_admin_user_id ON families(admin_user_id);
CREATE INDEX IF NOT EXISTS idx_families_deleted_at ON families(deleted_at);

CREATE TABLE IF NOT EXISTS users (
    id bigserial PRIMARY KEY,
    family_id bigint,
    username varchar(80) NOT NULL,
    display_name varchar(80) NOT NULL,
    email varchar(160),
    password_hash varchar(255) NOT NULL,
    role varchar(30) NOT NULL,
    must_change_password boolean NOT NULL DEFAULT true,
    status varchar(20) NOT NULL DEFAULT 'enabled',
    last_login_at timestamptz,
    created_at timestamptz,
    updated_at timestamptz,
    deleted_at timestamptz
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_family_id ON users(family_id);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at);

CREATE TABLE IF NOT EXISTS user_sessions (
    id bigserial PRIMARY KEY,
    user_id bigint NOT NULL,
    refresh_token_hash varchar(128) NOT NULL,
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    ip varchar(80),
    user_agent varchar(255),
    created_at timestamptz
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_sessions_refresh_token_hash ON user_sessions(refresh_token_hash);
CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id ON user_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_sessions_expires_at ON user_sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_user_sessions_revoked_at ON user_sessions(revoked_at);

CREATE TABLE IF NOT EXISTS dictionary_types (
    id bigserial PRIMARY KEY,
    code varchar(60) NOT NULL,
    name varchar(80) NOT NULL,
    description varchar(255),
    allow_family_custom boolean NOT NULL DEFAULT true,
    sort_order bigint NOT NULL DEFAULT 0,
    enabled boolean NOT NULL DEFAULT true,
    created_at timestamptz,
    updated_at timestamptz
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_dictionary_types_code ON dictionary_types(code);
CREATE INDEX IF NOT EXISTS idx_dictionary_types_enabled ON dictionary_types(enabled);

CREATE TABLE IF NOT EXISTS dictionary_items (
    id bigserial PRIMARY KEY,
    type_id bigint NOT NULL,
    family_id bigint,
    label varchar(80) NOT NULL,
    value varchar(80) NOT NULL,
    sort_order bigint NOT NULL DEFAULT 0,
    enabled boolean NOT NULL DEFAULT true,
    is_system boolean NOT NULL DEFAULT false,
    created_at timestamptz,
    updated_at timestamptz,
    deleted_at timestamptz
);
CREATE INDEX IF NOT EXISTS idx_dictionary_items_type_id ON dictionary_items(type_id);
CREATE INDEX IF NOT EXISTS idx_dictionary_items_family_id ON dictionary_items(family_id);
CREATE INDEX IF NOT EXISTS idx_dictionary_items_enabled ON dictionary_items(enabled);
CREATE INDEX IF NOT EXISTS idx_dictionary_items_is_system ON dictionary_items(is_system);
CREATE INDEX IF NOT EXISTS idx_dictionary_items_deleted_at ON dictionary_items(deleted_at);

CREATE TABLE IF NOT EXISTS contacts (
    id bigserial PRIMARY KEY,
    family_id bigint NOT NULL,
    name varchar(120) NOT NULL,
    contact_type varchar(80) NOT NULL,
    relation_type varchar(80) NOT NULL,
    phone varchar(40),
    address varchar(255),
    remark text,
    status varchar(20) NOT NULL DEFAULT 'enabled',
    created_by bigint,
    created_at timestamptz,
    updated_at timestamptz,
    deleted_at timestamptz
);
CREATE INDEX IF NOT EXISTS idx_contacts_family_id ON contacts(family_id);
CREATE INDEX IF NOT EXISTS idx_contacts_name ON contacts(name);
CREATE INDEX IF NOT EXISTS idx_contacts_contact_type ON contacts(contact_type);
CREATE INDEX IF NOT EXISTS idx_contacts_relation_type ON contacts(relation_type);
CREATE INDEX IF NOT EXISTS idx_contacts_status ON contacts(status);
CREATE INDEX IF NOT EXISTS idx_contacts_deleted_at ON contacts(deleted_at);

CREATE TABLE IF NOT EXISTS gift_records (
    id bigserial PRIMARY KEY,
    family_id bigint NOT NULL,
    contact_id bigint NOT NULL,
    occasion_type varchar(80) NOT NULL,
    direction varchar(10) NOT NULL,
    amount_cents bigint NOT NULL DEFAULT 0,
    gift_description text,
    occurred_on date NOT NULL,
    handled_by varchar(80),
    reciprocity_status varchar(20) NOT NULL DEFAULT 'none',
    related_record_id bigint,
    remark text,
    created_by bigint,
    created_at timestamptz,
    updated_at timestamptz,
    deleted_at timestamptz
);
CREATE INDEX IF NOT EXISTS idx_gift_records_family_id ON gift_records(family_id);
CREATE INDEX IF NOT EXISTS idx_gift_records_contact_id ON gift_records(contact_id);
CREATE INDEX IF NOT EXISTS idx_gift_records_occasion_type ON gift_records(occasion_type);
CREATE INDEX IF NOT EXISTS idx_gift_records_direction ON gift_records(direction);
CREATE INDEX IF NOT EXISTS idx_gift_records_occurred_on ON gift_records(occurred_on);
CREATE INDEX IF NOT EXISTS idx_gift_records_reciprocity_status ON gift_records(reciprocity_status);
CREATE INDEX IF NOT EXISTS idx_gift_records_related_record_id ON gift_records(related_record_id);
CREATE INDEX IF NOT EXISTS idx_gift_records_deleted_at ON gift_records(deleted_at);

CREATE TABLE IF NOT EXISTS attachments (
    id bigserial PRIMARY KEY,
    family_id bigint NOT NULL,
    record_id bigint NOT NULL,
    file_name varchar(255) NOT NULL,
    storage_key varchar(255) NOT NULL,
    file_type varchar(80),
    file_size bigint,
    uploaded_by bigint,
    created_at timestamptz,
    deleted_at timestamptz
);
CREATE INDEX IF NOT EXISTS idx_attachments_family_id ON attachments(family_id);
CREATE INDEX IF NOT EXISTS idx_attachments_record_id ON attachments(record_id);
CREATE INDEX IF NOT EXISTS idx_attachments_deleted_at ON attachments(deleted_at);

CREATE TABLE IF NOT EXISTS backup_configs (
    id bigserial PRIMARY KEY,
    family_id bigint NOT NULL,
    enabled boolean NOT NULL DEFAULT false,
    frequency varchar(20) NOT NULL DEFAULT 'weekly',
    recipient_emails jsonb NOT NULL DEFAULT '[]',
    encrypt_enabled boolean NOT NULL DEFAULT false,
    zip_password_encrypted text,
    zip_password_hint varchar(255),
    last_sent_at timestamptz,
    last_status varchar(30),
    created_at timestamptz,
    updated_at timestamptz
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_backup_configs_family_id ON backup_configs(family_id);

CREATE TABLE IF NOT EXISTS backup_jobs (
    id bigserial PRIMARY KEY,
    family_id bigint NOT NULL,
    trigger_type varchar(30) NOT NULL,
    status varchar(30) NOT NULL,
    file_name varchar(255),
    file_size bigint,
    file_sha256 varchar(64),
    recipient_emails jsonb NOT NULL DEFAULT '[]',
    started_at timestamptz,
    finished_at timestamptz,
    error_message text,
    created_by bigint
);
CREATE INDEX IF NOT EXISTS idx_backup_jobs_family_id ON backup_jobs(family_id);
CREATE INDEX IF NOT EXISTS idx_backup_jobs_status ON backup_jobs(status);

CREATE TABLE IF NOT EXISTS import_jobs (
    id bigserial PRIMARY KEY,
    family_id bigint NOT NULL,
    operator_id bigint NOT NULL,
    status varchar(30) NOT NULL,
    dry_run boolean NOT NULL DEFAULT true,
    source_file_name varchar(255),
    source_payload jsonb NOT NULL DEFAULT '{}',
    created_count bigint,
    updated_count bigint,
    skipped_count bigint,
    error_count bigint,
    started_at timestamptz,
    finished_at timestamptz,
    error_message text
);
CREATE INDEX IF NOT EXISTS idx_import_jobs_family_id ON import_jobs(family_id);
CREATE INDEX IF NOT EXISTS idx_import_jobs_operator_id ON import_jobs(operator_id);
CREATE INDEX IF NOT EXISTS idx_import_jobs_status ON import_jobs(status);

CREATE TABLE IF NOT EXISTS import_job_errors (
    id bigserial PRIMARY KEY,
    import_job_id bigint NOT NULL,
    sheet_name varchar(80),
    row_number bigint,
    field_name varchar(80),
    error_message text,
    raw_data jsonb NOT NULL DEFAULT '{}'
);
CREATE INDEX IF NOT EXISTS idx_import_job_errors_import_job_id ON import_job_errors(import_job_id);

CREATE TABLE IF NOT EXISTS audit_logs (
    id bigserial PRIMARY KEY,
    family_id bigint,
    operator_id bigint,
    action varchar(80) NOT NULL,
    resource_type varchar(80) NOT NULL,
    resource_id varchar(80),
    success boolean NOT NULL DEFAULT true,
    ip varchar(80),
    user_agent varchar(255),
    detail jsonb NOT NULL DEFAULT '{}',
    created_at timestamptz
);
CREATE INDEX IF NOT EXISTS idx_audit_logs_family_id ON audit_logs(family_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_operator_id ON audit_logs(operator_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource_type ON audit_logs(resource_type);
CREATE INDEX IF NOT EXISTS idx_audit_logs_success ON audit_logs(success);

COMMIT;
