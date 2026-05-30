ALTER TABLE songs
    ADD COLUMN updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP AFTER is_deleted;

ALTER TABLE charts
    ADD COLUMN updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP AFTER notes_designer,
    DROP INDEX idx_charts_song_id;

ALTER TABLE worldsend_charts
    ADD COLUMN updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP AFTER notes_designer,
    DROP INDEX idx_worldsend_charts_song_id;

ALTER TABLE sessions
    DROP INDEX idx_sessions_user_id;

CREATE INDEX idx_player_worldsend_records_player_updated_at
    ON player_worldsend_records(player_id, updated_at DESC);

CREATE INDEX idx_goals_user_created_id
    ON goals(user_id, created_at, id);
