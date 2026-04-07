ALTER TABLE charts
    ADD COLUMN notes_designer VARCHAR(100) NULL AFTER notes;

ALTER TABLE worldsend_charts
    ADD COLUMN notes_designer VARCHAR(100) NULL AFTER notes;
