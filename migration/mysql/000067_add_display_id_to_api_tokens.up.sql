ALTER TABLE api_tokens
    ADD COLUMN display_id CHAR(16) NULL AFTER id;

UPDATE api_tokens
SET display_id = LOWER(HEX(RANDOM_BYTES(8)))
WHERE display_id IS NULL;

ALTER TABLE api_tokens
    MODIFY COLUMN display_id CHAR(16) COLLATE utf8mb4_unicode_ci NOT NULL,
    ADD UNIQUE KEY uq_api_tokens_display_id (display_id);
