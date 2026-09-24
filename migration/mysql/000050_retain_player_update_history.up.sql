ALTER TABLE player_latest_updates
    DROP PRIMARY KEY,
    ADD PRIMARY KEY (player_id, source_updated_at);
