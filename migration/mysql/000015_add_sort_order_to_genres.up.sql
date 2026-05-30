ALTER TABLE genres
    ADD COLUMN sort_order TINYINT UNSIGNED NULL AFTER name;

UPDATE genres SET sort_order = 0 WHERE name = 'POPS & ANIME';
UPDATE genres SET sort_order = 1 WHERE name = 'niconico';
UPDATE genres SET sort_order = 2 WHERE name = '東方Project';
UPDATE genres SET sort_order = 3 WHERE name = 'VARIETY';
UPDATE genres SET sort_order = 4 WHERE name = 'イロドリミドリ';
UPDATE genres SET sort_order = 5 WHERE name = 'ゲキマイ';
UPDATE genres SET sort_order = 6 WHERE name = 'ORIGINAL';
UPDATE genres SET sort_order = 255 WHERE sort_order IS NULL;

ALTER TABLE genres
    MODIFY COLUMN sort_order TINYINT UNSIGNED NOT NULL;
