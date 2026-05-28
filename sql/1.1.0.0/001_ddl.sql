BEGIN;

CREATE TABLE IF NOT EXISTS auth_throttles (
    id bigserial PRIMARY KEY,
    scope varchar(30) NOT NULL,
    key varchar(160) NOT NULL,
    failed_count bigint NOT NULL DEFAULT 0,
    window_started_at timestamptz NOT NULL,
    last_failed_at timestamptz NOT NULL,
    locked_until timestamptz,
    created_at timestamptz,
    updated_at timestamptz
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_auth_throttle_scope_key ON auth_throttles(scope, key);
CREATE INDEX IF NOT EXISTS idx_auth_throttles_window_started_at ON auth_throttles(window_started_at);
CREATE INDEX IF NOT EXISTS idx_auth_throttles_last_failed_at ON auth_throttles(last_failed_at);
CREATE INDEX IF NOT EXISTS idx_auth_throttles_locked_until ON auth_throttles(locked_until);

COMMIT;
