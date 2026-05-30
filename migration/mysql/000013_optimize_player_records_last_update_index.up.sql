DROP INDEX idx_player_records_updated_at
    ON player_records;

DROP INDEX idx_player_worldsend_records_updated_at
    ON player_worldsend_records;

DROP INDEX idx_goals_user_id
    ON goals;

DROP INDEX idx_songs_title
    ON songs;

CREATE INDEX idx_player_records_player_updated_at
    ON player_records(player_id, updated_at DESC);
