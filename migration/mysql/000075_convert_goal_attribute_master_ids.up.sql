-- 目標の絞り込み属性 diff / genre / ver をDBの数値IDから公開識別子へ移行する。
-- diff と genre はマスタ名、ver はバージョンの display_id（API上の id）を保存する。
-- 複数指定は表示順（難易度・ジャンルは sort_order、バージョンは released_at）に並べ、
-- マスタに存在しないIDは取り除く。1件だけ残った配列は単一値に、1件も残らない属性は削除する。

UPDATE goals g
SET g.attributes = JSON_SET(g.attributes, '$.diff', (
    SELECT CAST(CONCAT('[', GROUP_CONCAT(JSON_QUOTE(t.name) ORDER BY t.sort_order, t.id SEPARATOR ','), ']') AS JSON)
    FROM JSON_TABLE(g.attributes, '$.diff[*]' COLUMNS (master_id INT UNSIGNED PATH '$')) AS jt
    INNER JOIN difficulties t ON t.id = jt.master_id
))
WHERE JSON_TYPE(JSON_EXTRACT(g.attributes, '$.diff')) = 'ARRAY';

UPDATE goals
SET attributes = JSON_SET(attributes, '$.diff', JSON_EXTRACT(attributes, '$.diff[0]'))
WHERE JSON_TYPE(JSON_EXTRACT(attributes, '$.diff')) = 'ARRAY'
  AND JSON_LENGTH(attributes, '$.diff') = 1;

UPDATE goals g
SET g.attributes = JSON_SET(g.attributes, '$.diff', (SELECT t.name FROM difficulties t WHERE t.id = JSON_VALUE(g.attributes, '$.diff' RETURNING UNSIGNED)))
WHERE JSON_TYPE(JSON_EXTRACT(g.attributes, '$.diff')) IN ('INTEGER', 'UNSIGNED INTEGER');

UPDATE goals
SET attributes = JSON_REMOVE(attributes, '$.diff')
WHERE JSON_TYPE(JSON_EXTRACT(attributes, '$.diff')) = 'NULL';

UPDATE goals g
SET g.attributes = JSON_SET(g.attributes, '$.genre', (
    SELECT CAST(CONCAT('[', GROUP_CONCAT(JSON_QUOTE(t.name) ORDER BY t.sort_order, t.id SEPARATOR ','), ']') AS JSON)
    FROM JSON_TABLE(g.attributes, '$.genre[*]' COLUMNS (master_id INT UNSIGNED PATH '$')) AS jt
    INNER JOIN genres t ON t.id = jt.master_id
))
WHERE JSON_TYPE(JSON_EXTRACT(g.attributes, '$.genre')) = 'ARRAY';

UPDATE goals
SET attributes = JSON_SET(attributes, '$.genre', JSON_EXTRACT(attributes, '$.genre[0]'))
WHERE JSON_TYPE(JSON_EXTRACT(attributes, '$.genre')) = 'ARRAY'
  AND JSON_LENGTH(attributes, '$.genre') = 1;

UPDATE goals g
SET g.attributes = JSON_SET(g.attributes, '$.genre', (SELECT t.name FROM genres t WHERE t.id = JSON_VALUE(g.attributes, '$.genre' RETURNING UNSIGNED)))
WHERE JSON_TYPE(JSON_EXTRACT(g.attributes, '$.genre')) IN ('INTEGER', 'UNSIGNED INTEGER');

UPDATE goals
SET attributes = JSON_REMOVE(attributes, '$.genre')
WHERE JSON_TYPE(JSON_EXTRACT(attributes, '$.genre')) = 'NULL';

UPDATE goals g
SET g.attributes = JSON_SET(g.attributes, '$.ver', (
    SELECT CAST(CONCAT('[', GROUP_CONCAT(JSON_QUOTE(t.display_id) ORDER BY t.released_at, t.id SEPARATOR ','), ']') AS JSON)
    FROM JSON_TABLE(g.attributes, '$.ver[*]' COLUMNS (master_id INT UNSIGNED PATH '$')) AS jt
    INNER JOIN versions t ON t.id = jt.master_id
))
WHERE JSON_TYPE(JSON_EXTRACT(g.attributes, '$.ver')) = 'ARRAY';

UPDATE goals
SET attributes = JSON_SET(attributes, '$.ver', JSON_EXTRACT(attributes, '$.ver[0]'))
WHERE JSON_TYPE(JSON_EXTRACT(attributes, '$.ver')) = 'ARRAY'
  AND JSON_LENGTH(attributes, '$.ver') = 1;

UPDATE goals g
SET g.attributes = JSON_SET(g.attributes, '$.ver', (SELECT t.display_id FROM versions t WHERE t.id = JSON_VALUE(g.attributes, '$.ver' RETURNING UNSIGNED)))
WHERE JSON_TYPE(JSON_EXTRACT(g.attributes, '$.ver')) IN ('INTEGER', 'UNSIGNED INTEGER');

UPDATE goals
SET attributes = JSON_REMOVE(attributes, '$.ver')
WHERE JSON_TYPE(JSON_EXTRACT(attributes, '$.ver')) = 'NULL';
