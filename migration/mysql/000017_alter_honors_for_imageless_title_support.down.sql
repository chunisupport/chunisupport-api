ALTER TABLE honors
    DROP INDEX unique_honor_type_name_image,
    ADD UNIQUE KEY unique_honor_name_type (name, honor_type_id);

UPDATE honors
SET name = ''
WHERE name IS NULL;

ALTER TABLE honors
    MODIFY COLUMN name VARCHAR(500) NOT NULL;
