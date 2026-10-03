ALTER TABLE charts
    DROP CHECK charts_chk_1,
    ADD CONSTRAINT charts_chk_1 CHECK (const BETWEEN 1.0 AND 16.0);
