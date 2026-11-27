ALTER TABLE versions
    ADD COLUMN display_id CHAR(16) NULL AFTER id;

UPDATE versions
SET display_id = LOWER(HEX(RANDOM_BYTES(8)))
WHERE display_id IS NULL;

ALTER TABLE versions
    MODIFY COLUMN display_id CHAR(16) COLLATE utf8mb4_unicode_ci NOT NULL,
    ADD UNIQUE KEY uq_versions_display_id (display_id);

ALTER TABLE honors
    ADD COLUMN display_id CHAR(16) NULL AFTER id;

UPDATE honors
SET display_id = LOWER(HEX(RANDOM_BYTES(8)))
WHERE display_id IS NULL;

ALTER TABLE honors
    MODIFY COLUMN display_id CHAR(16) COLLATE utf8mb4_unicode_ci NOT NULL,
    ADD UNIQUE KEY uq_honors_display_id (display_id);
