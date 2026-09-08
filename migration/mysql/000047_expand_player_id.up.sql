-- players.id と全参照カラムは外部キーの型を一致させたまま拡張する。
ALTER TABLE users DROP FOREIGN KEY fk_users_player_id;
ALTER TABLE player_course_records DROP FOREIGN KEY fk_player_course_records_player;
ALTER TABLE player_favorite_songs DROP FOREIGN KEY fk_player_favorite_songs_player_id;
ALTER TABLE player_honors DROP FOREIGN KEY player_honors_ibfk_1;
ALTER TABLE player_latest_updates DROP FOREIGN KEY fk_player_latest_updates_player;
ALTER TABLE player_locked_songs DROP FOREIGN KEY fk_player_locked_songs_player_id;
ALTER TABLE player_metric_histories DROP FOREIGN KEY fk_player_metric_histories_player;
ALTER TABLE player_record_histories DROP FOREIGN KEY fk_player_record_histories_player;
ALTER TABLE player_records DROP FOREIGN KEY player_records_ibfk_1;
ALTER TABLE player_worldsend_record_histories DROP FOREIGN KEY fk_player_worldsend_record_histories_player;
ALTER TABLE player_worldsend_records DROP FOREIGN KEY player_worldsend_records_ibfk_1;

ALTER TABLE players MODIFY COLUMN id INT UNSIGNED NOT NULL AUTO_INCREMENT;
ALTER TABLE users MODIFY COLUMN player_id INT UNSIGNED NULL;
ALTER TABLE player_course_records MODIFY COLUMN player_id INT UNSIGNED NOT NULL;
ALTER TABLE player_favorite_songs MODIFY COLUMN player_id INT UNSIGNED NOT NULL;
ALTER TABLE player_honors MODIFY COLUMN player_id INT UNSIGNED NOT NULL;
ALTER TABLE player_latest_updates MODIFY COLUMN player_id INT UNSIGNED NOT NULL;
ALTER TABLE player_locked_songs MODIFY COLUMN player_id INT UNSIGNED NOT NULL;
ALTER TABLE player_metric_histories MODIFY COLUMN player_id INT UNSIGNED NOT NULL;
ALTER TABLE player_record_histories MODIFY COLUMN player_id INT UNSIGNED NOT NULL;
ALTER TABLE player_records MODIFY COLUMN player_id INT UNSIGNED NOT NULL;
ALTER TABLE player_worldsend_record_histories MODIFY COLUMN player_id INT UNSIGNED NOT NULL;
ALTER TABLE player_worldsend_records MODIFY COLUMN player_id INT UNSIGNED NOT NULL;

ALTER TABLE users ADD CONSTRAINT fk_users_player_id FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE SET NULL;
ALTER TABLE player_course_records ADD CONSTRAINT fk_player_course_records_player FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE;
ALTER TABLE player_favorite_songs ADD CONSTRAINT fk_player_favorite_songs_player_id FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE;
ALTER TABLE player_honors ADD CONSTRAINT player_honors_ibfk_1 FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE;
ALTER TABLE player_latest_updates ADD CONSTRAINT fk_player_latest_updates_player FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE;
ALTER TABLE player_locked_songs ADD CONSTRAINT fk_player_locked_songs_player_id FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE;
ALTER TABLE player_metric_histories ADD CONSTRAINT fk_player_metric_histories_player FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE;
ALTER TABLE player_record_histories ADD CONSTRAINT fk_player_record_histories_player FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE;
ALTER TABLE player_records ADD CONSTRAINT player_records_ibfk_1 FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE;
ALTER TABLE player_worldsend_record_histories ADD CONSTRAINT fk_player_worldsend_record_histories_player FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE;
ALTER TABLE player_worldsend_records ADD CONSTRAINT player_worldsend_records_ibfk_1 FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE;
