ALTER TABLE versions
    DROP INDEX uq_versions_short_name,
    DROP COLUMN short_name;

ALTER TABLE genres
    DROP INDEX uq_genres_short_name,
    DROP COLUMN short_name;
