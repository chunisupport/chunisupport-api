ALTER TABLE goals
    DROP COLUMN invert_percentage,
    CHANGE COLUMN invert_value invert BOOLEAN NOT NULL DEFAULT FALSE;
