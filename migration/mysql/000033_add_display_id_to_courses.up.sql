ALTER TABLE courses
    ADD COLUMN display_id CHAR(16) NULL AFTER id;

UPDATE courses
SET display_id = LOWER(HEX(RANDOM_BYTES(8)))
WHERE display_id IS NULL;

ALTER TABLE courses
    MODIFY COLUMN display_id CHAR(16) NOT NULL,
    ADD UNIQUE KEY uq_courses_display_id (display_id);
