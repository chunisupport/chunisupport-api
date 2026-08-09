UPDATE player_metric_histories
SET data_collected_at = TIMESTAMPADD(MICROSECOND, -MICROSECOND(data_collected_at), data_collected_at)
WHERE MICROSECOND(data_collected_at) <> 0;

ALTER TABLE player_metric_histories
    MODIFY COLUMN data_collected_at TIMESTAMP NOT NULL;

UPDATE players
SET data_collected_at = TIMESTAMPADD(MICROSECOND, -MICROSECOND(data_collected_at), data_collected_at)
WHERE data_collected_at IS NOT NULL
  AND MICROSECOND(data_collected_at) <> 0;

ALTER TABLE players
    MODIFY COLUMN data_collected_at TIMESTAMP NULL;
