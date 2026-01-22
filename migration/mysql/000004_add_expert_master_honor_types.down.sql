-- 称号種類マスタから expert と master を削除
DELETE FROM honor_types WHERE name IN ('expert', 'master');
