DELETE older FROM player_latest_updates AS older
JOIN player_latest_updates AS newer
    ON older.player_id = newer.player_id
    AND older.source_updated_at < newer.source_updated_at;

ALTER TABLE player_latest_updates
    DROP PRIMARY KEY,
    ADD PRIMARY KEY (player_id);
