UPDATE honors
SET image_url = ''
WHERE image_url IS NULL;

ALTER TABLE honors
    MODIFY COLUMN image_url VARCHAR(255) NOT NULL DEFAULT '',
    DROP INDEX unique_honor_name_type,
    ADD UNIQUE KEY unique_honor_name_type_image_url (name, honor_type_id, image_url);
