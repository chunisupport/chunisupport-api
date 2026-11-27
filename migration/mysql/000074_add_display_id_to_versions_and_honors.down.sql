ALTER TABLE honors
    DROP INDEX uq_honors_display_id,
    DROP COLUMN display_id;

ALTER TABLE versions
    DROP INDEX uq_versions_display_id,
    DROP COLUMN display_id;
