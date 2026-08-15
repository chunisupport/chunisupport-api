ALTER TABLE goals
    CHANGE COLUMN invert invert_value BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN invert_percentage BOOLEAN NOT NULL DEFAULT FALSE AFTER invert_value;

UPDATE goals
SET invert_percentage = invert_value;
