BEGIN;

INSERT INTO schema_migrations (version, description)
VALUES
    ('1.0.0.0', '首版业务表结构和预置字典'),
    ('1.1.0.0', '登录失败防护表'),
    ('1.2.0.0', '手动 SQL 版本目录和迁移记录表')
ON CONFLICT (version) DO UPDATE
SET description = EXCLUDED.description,
    applied_at = now();

COMMIT;
