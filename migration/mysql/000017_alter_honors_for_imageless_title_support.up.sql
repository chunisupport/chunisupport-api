ALTER TABLE honors
    MODIFY COLUMN name VARCHAR(500) NULL;

ALTER TABLE honors
    DROP INDEX unique_honor_name_type,
    ADD UNIQUE KEY unique_honor_type_name_image (honor_type_id, name, image_url);
