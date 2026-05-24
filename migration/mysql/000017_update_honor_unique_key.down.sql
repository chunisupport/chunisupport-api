UPDATE honors h
INNER JOIN (
    SELECT h1.id
    FROM honors h1
    INNER JOIN (
        SELECT name, honor_type_id, MIN(id) AS keep_id
        FROM honors
        GROUP BY name, honor_type_id
        HAVING COUNT(*) > 1
    ) duplicated_group
        ON duplicated_group.honor_type_id = h1.honor_type_id
        AND duplicated_group.name = h1.name
        AND duplicated_group.keep_id <> h1.id
) duplicated ON duplicated.id = h.id
SET h.name = CONCAT(
    LEFT(CASE WHEN h.name = '' THEN 'honor' ELSE h.name END, 479),
    '#',
    h.id
);

ALTER TABLE honors
    DROP INDEX unique_honor_name_type_image_url,
    MODIFY COLUMN image_url VARCHAR(255) NULL,
    ADD UNIQUE KEY unique_honor_name_type (name, honor_type_id);
