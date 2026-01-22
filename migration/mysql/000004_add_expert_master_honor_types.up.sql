-- 称号種類マスタに expert と master を追加
INSERT INTO honor_types (name) VALUES
    ('expert'),
    ('master')
ON DUPLICATE KEY UPDATE name = VALUES(name);
