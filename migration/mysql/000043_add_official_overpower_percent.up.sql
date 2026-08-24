ALTER TABLE players
    ADD COLUMN official_overpower_percent DECIMAL(5, 2) NULL AFTER official_overpower;

ALTER TABLE player_metric_histories
    ADD COLUMN official_overpower_percent DECIMAL(5, 2) NULL AFTER official_overpower;
