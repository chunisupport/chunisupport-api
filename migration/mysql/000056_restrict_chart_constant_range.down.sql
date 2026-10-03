ALTER TABLE charts
    DROP CHECK charts_chk_1,
    ADD CONSTRAINT charts_chk_1 CHECK (const >= 0);
