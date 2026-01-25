-- 称号種類マスタから expert と master を削除
-- 外部キー制約を考慮して、まず参照している honors テーブルのレコードを削除
DELETE FROM honors 
WHERE honor_type_id IN (
    SELECT id FROM honor_types WHERE name IN ('expert', 'master')
);

-- その後、honor_types から削除
DELETE FROM honor_types WHERE name IN ('expert', 'master');
