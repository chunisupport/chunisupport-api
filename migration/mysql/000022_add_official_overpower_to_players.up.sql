ALTER TABLE players
    ADD COLUMN official_overpower DECIMAL(8, 2) NOT NULL DEFAULT 0.00 AFTER overpower_value;
