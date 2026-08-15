ALTER TABLE courses
    DROP INDEX uq_courses_display_id,
    DROP COLUMN display_id;
