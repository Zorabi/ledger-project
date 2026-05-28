BEGIN;

INSERT INTO dictionary_types (code, name, description, allow_family_custom, sort_order, enabled, created_at, updated_at)
VALUES
    ('relation_type', '关系类型', '', true, 10, true, now(), now()),
    ('contact_type', '对象类型', '', true, 20, true, now(), now()),
    ('occasion_type', '事项类型', '', true, 30, true, now(), now()),
    ('gift_type', '礼品类型', '', true, 40, true, now(), now()),
    ('payment_method', '支付方式', '', true, 50, true, now(), now())
ON CONFLICT (code) DO UPDATE
SET name = EXCLUDED.name,
    allow_family_custom = EXCLUDED.allow_family_custom,
    sort_order = EXCLUDED.sort_order,
    enabled = EXCLUDED.enabled,
    updated_at = now();

WITH defaults(type_code, label, value, sort_order) AS (
    VALUES
        ('relation_type', '亲戚', '亲戚', 1),
        ('relation_type', '朋友', '朋友', 2),
        ('relation_type', '同事', '同事', 3),
        ('relation_type', '同学', '同学', 4),
        ('relation_type', '邻居', '邻居', 5),
        ('contact_type', '个人', '个人', 1),
        ('contact_type', '家庭', '家庭', 2),
        ('contact_type', '单位', '单位', 3),
        ('occasion_type', '婚礼', '婚礼', 1),
        ('occasion_type', '满月', '满月', 2),
        ('occasion_type', '生日', '生日', 3),
        ('occasion_type', '升学', '升学', 4),
        ('occasion_type', '乔迁', '乔迁', 5),
        ('occasion_type', '探病', '探病', 6),
        ('gift_type', '礼金', '礼金', 1),
        ('gift_type', '礼品', '礼品', 2),
        ('payment_method', '现金', '现金', 1),
        ('payment_method', '微信', '微信', 2),
        ('payment_method', '支付宝', '支付宝', 3),
        ('payment_method', '银行转账', '银行转账', 4)
)
INSERT INTO dictionary_items (type_id, family_id, label, value, sort_order, enabled, is_system, created_at, updated_at)
SELECT dt.id, NULL, d.label, d.value, d.sort_order, true, true, now(), now()
FROM defaults d
JOIN dictionary_types dt ON dt.code = d.type_code
WHERE NOT EXISTS (
    SELECT 1
    FROM dictionary_items di
    WHERE di.type_id = dt.id
      AND di.family_id IS NULL
      AND di.value = d.value
      AND di.deleted_at IS NULL
);

COMMIT;
