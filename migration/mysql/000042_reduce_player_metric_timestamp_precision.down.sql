ALTER TABLE players
    MODIFY COLUMN data_collected_at TIMESTAMP(6) NULL;

ALTER TABLE player_metric_histories
    MODIFY COLUMN data_collected_at TIMESTAMP(6) NOT NULL;
