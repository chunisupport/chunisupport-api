DROP TABLE IF EXISTS player_metric_histories;

ALTER TABLE players
    MODIFY COLUMN official_player_rating DECIMAL(4, 2) NULL,
    MODIFY COLUMN data_collected_at TIMESTAMP NULL;
