ALTER TABLE songs
    DROP COLUMN updated_at;

ALTER TABLE charts
    ADD INDEX idx_charts_song_id (song_id),
    DROP COLUMN updated_at;

ALTER TABLE worldsend_charts
    ADD INDEX idx_worldsend_charts_song_id (song_id),
    DROP COLUMN updated_at;

ALTER TABLE sessions
    ADD INDEX idx_sessions_user_id (user_id);

DROP INDEX idx_player_worldsend_records_player_updated_at
    ON player_worldsend_records;

DROP INDEX idx_goals_user_created_id
    ON goals;
