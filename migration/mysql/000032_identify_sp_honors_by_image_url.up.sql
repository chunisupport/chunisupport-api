UPDATE honors
SET image_url = TRIM(image_url)
WHERE image_url IS NOT NULL;

UPDATE honors
SET image_url = NULL
WHERE image_url = '';

CREATE TEMPORARY TABLE honor_image_url_keepers AS
SELECT ranked.image_url, ranked.id AS keep_id
FROM (
    SELECT
        id,
        image_url,
        ROW_NUMBER() OVER (
            PARTITION BY image_url
            ORDER BY
                CASE
                    WHEN name = SUBSTRING_INDEX(SUBSTRING_INDEX(SUBSTRING_INDEX(image_url, '#', 1), '?', 1), '/', -1)
                        THEN 1
                    ELSE 0
                END,
                id
        ) AS honor_rank
    FROM honors
    WHERE image_url IS NOT NULL
) AS ranked
WHERE ranked.honor_rank = 1;

UPDATE player_honors AS ph
INNER JOIN honors AS duplicated ON duplicated.id = ph.honor_id
INNER JOIN honor_image_url_keepers AS keepers
    ON keepers.image_url = duplicated.image_url
SET ph.honor_id = keepers.keep_id
WHERE duplicated.id <> keepers.keep_id;

DELETE duplicated
FROM honors AS duplicated
INNER JOIN honor_image_url_keepers AS keepers
    ON keepers.image_url = duplicated.image_url
    AND duplicated.id <> keepers.keep_id;

DROP TEMPORARY TABLE honor_image_url_keepers;

ALTER TABLE honors
    ADD UNIQUE KEY unique_honor_image_url (image_url);
