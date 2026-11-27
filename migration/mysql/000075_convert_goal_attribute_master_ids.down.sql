-- 目標の絞り込み属性 diff / genre / ver を公開識別子からDBの数値IDへ戻す。
-- 複数指定は数値ID昇順に並べ、マスタに存在しない値は取り除く。

UPDATE goals g
SET g.attributes = JSON_SET(g.attributes, '$.diff', (
    SELECT CAST(CONCAT('[', GROUP_CONCAT(t.id ORDER BY t.id SEPARATOR ','), ']') AS JSON)
    FROM JSON_TABLE(g.attributes, '$.diff[*]' COLUMNS (master_value VARCHAR(15) PATH '$')) AS jt
    INNER JOIN difficulties t ON t.name = jt.master_value COLLATE utf8mb4_unicode_ci
))
WHERE JSON_TYPE(JSON_EXTRACT(g.attributes, '$.diff')) = 'ARRAY';

UPDATE goals
SET attributes = JSON_SET(attributes, '$.diff', JSON_EXTRACT(attributes, '$.diff[0]'))
WHERE JSON_TYPE(JSON_EXTRACT(attributes, '$.diff')) = 'ARRAY'
  AND JSON_LENGTH(attributes, '$.diff') = 1;

UPDATE goals g
SET g.attributes = JSON_SET(g.attributes, '$.diff', (SELECT t.id FROM difficulties t WHERE t.name = CAST(JSON_UNQUOTE(JSON_EXTRACT(g.attributes, '$.diff')) AS CHAR CHARACTER SET utf8mb4) COLLATE utf8mb4_unicode_ci))
WHERE JSON_TYPE(JSON_EXTRACT(g.attributes, '$.diff')) IN ('STRING');

UPDATE goals
SET attributes = JSON_REMOVE(attributes, '$.diff')
WHERE JSON_TYPE(JSON_EXTRACT(attributes, '$.diff')) = 'NULL';

UPDATE goals g
SET g.attributes = JSON_SET(g.attributes, '$.genre', (
    SELECT CAST(CONCAT('[', GROUP_CONCAT(t.id ORDER BY t.id SEPARATOR ','), ']') AS JSON)
    FROM JSON_TABLE(g.attributes, '$.genre[*]' COLUMNS (master_value VARCHAR(30) PATH '$')) AS jt
    INNER JOIN genres t ON t.name = jt.master_value COLLATE utf8mb4_unicode_ci
))
WHERE JSON_TYPE(JSON_EXTRACT(g.attributes, '$.genre')) = 'ARRAY';

UPDATE goals
SET attributes = JSON_SET(attributes, '$.genre', JSON_EXTRACT(attributes, '$.genre[0]'))
WHERE JSON_TYPE(JSON_EXTRACT(attributes, '$.genre')) = 'ARRAY'
  AND JSON_LENGTH(attributes, '$.genre') = 1;

UPDATE goals g
SET g.attributes = JSON_SET(g.attributes, '$.genre', (SELECT t.id FROM genres t WHERE t.name = CAST(JSON_UNQUOTE(JSON_EXTRACT(g.attributes, '$.genre')) AS CHAR CHARACTER SET utf8mb4) COLLATE utf8mb4_unicode_ci))
WHERE JSON_TYPE(JSON_EXTRACT(g.attributes, '$.genre')) IN ('STRING');

UPDATE goals
SET attributes = JSON_REMOVE(attributes, '$.genre')
WHERE JSON_TYPE(JSON_EXTRACT(attributes, '$.genre')) = 'NULL';

UPDATE goals g
SET g.attributes = JSON_SET(g.attributes, '$.ver', (
    SELECT CAST(CONCAT('[', GROUP_CONCAT(t.id ORDER BY t.id SEPARATOR ','), ']') AS JSON)
    FROM JSON_TABLE(g.attributes, '$.ver[*]' COLUMNS (master_value CHAR(16) PATH '$')) AS jt
    INNER JOIN versions t ON t.display_id = jt.master_value COLLATE utf8mb4_unicode_ci
))
WHERE JSON_TYPE(JSON_EXTRACT(g.attributes, '$.ver')) = 'ARRAY';

UPDATE goals
SET attributes = JSON_SET(attributes, '$.ver', JSON_EXTRACT(attributes, '$.ver[0]'))
WHERE JSON_TYPE(JSON_EXTRACT(attributes, '$.ver')) = 'ARRAY'
  AND JSON_LENGTH(attributes, '$.ver') = 1;

UPDATE goals g
SET g.attributes = JSON_SET(g.attributes, '$.ver', (SELECT t.id FROM versions t WHERE t.display_id = CAST(JSON_UNQUOTE(JSON_EXTRACT(g.attributes, '$.ver')) AS CHAR CHARACTER SET utf8mb4) COLLATE utf8mb4_unicode_ci))
WHERE JSON_TYPE(JSON_EXTRACT(g.attributes, '$.ver')) IN ('STRING');

UPDATE goals
SET attributes = JSON_REMOVE(attributes, '$.ver')
WHERE JSON_TYPE(JSON_EXTRACT(attributes, '$.ver')) = 'NULL';
