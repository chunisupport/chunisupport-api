DROP INDEX idx_player_records_player_updated_at
    ON player_records;

CREATE INDEX idx_player_records_updated_at
    ON player_records(updated_at);

CREATE INDEX idx_player_worldsend_records_updated_at
    ON player_worldsend_records(updated_at);

CREATE INDEX idx_goals_user_id
    ON goals(user_id);

CREATE INDEX idx_songs_title
    ON songs(title);
