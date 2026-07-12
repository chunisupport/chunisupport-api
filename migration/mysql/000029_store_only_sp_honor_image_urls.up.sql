ALTER TABLE honors
    MODIFY COLUMN image_url VARCHAR(255) NULL;

UPDATE honors AS h
INNER JOIN honor_types AS ht ON ht.id = h.honor_type_id
SET h.image_url = NULL
WHERE ht.name <> 'sp';

UPDATE honors AS h
INNER JOIN honor_types AS ht ON ht.id = h.honor_type_id
SET h.name = SUBSTRING_INDEX(SUBSTRING_INDEX(SUBSTRING_INDEX(h.image_url, '#', 1), '?', 1), '/', -1)
WHERE ht.name = 'sp'
    AND h.image_url IS NOT NULL
    AND h.image_url <> '';

UPDATE player_honors AS ph
INNER JOIN honors AS duplicated ON duplicated.id = ph.honor_id
INNER JOIN (
    SELECT h.name, h.honor_type_id, MIN(h.id) AS keep_id
    FROM honors AS h
    GROUP BY h.name, h.honor_type_id
    HAVING COUNT(*) > 1
) AS duplicated_group
    ON duplicated.name = duplicated_group.name
    AND duplicated.honor_type_id = duplicated_group.honor_type_id
SET ph.honor_id = duplicated_group.keep_id
WHERE duplicated.id <> duplicated_group.keep_id;

DELETE duplicated
FROM honors AS duplicated
INNER JOIN (
    SELECT *
    FROM (
        SELECT h.name, h.honor_type_id, MIN(h.id) AS keep_id
        FROM honors AS h
        GROUP BY h.name, h.honor_type_id
        HAVING COUNT(*) > 1
    ) AS grouped_honors
) AS duplicated_group
    ON duplicated.name = duplicated_group.name
    AND duplicated.honor_type_id = duplicated_group.honor_type_id
    AND duplicated.id <> duplicated_group.keep_id;

ALTER TABLE honors
    DROP INDEX unique_honor_name_type_image_url,
    ADD UNIQUE KEY unique_honor_name_type (name, honor_type_id);
