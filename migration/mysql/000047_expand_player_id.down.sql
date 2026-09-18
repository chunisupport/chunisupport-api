-- CHECK制約でstrict SQL modeを確認し、範囲外IDの切り詰めが起こり得る接続では型縮小を開始しない。
DROP TEMPORARY TABLE IF EXISTS migration_require_strict_sql_mode;
CREATE TEMPORARY TABLE migration_require_strict_sql_mode (
    enabled BOOLEAN NOT NULL,
    CONSTRAINT chk_migration_require_strict_sql_mode CHECK (enabled = TRUE)
);
INSERT INTO migration_require_strict_sql_mode (enabled)
SELECT FIND_IN_SET('STRICT_ALL_TABLES', @@SESSION.sql_mode) > 0
    OR FIND_IN_SET('STRICT_TRANS_TABLES', @@SESSION.sql_mode) > 0;
DROP TEMPORARY TABLE migration_require_strict_sql_mode;

-- MEDIUMINT UNSIGNED の上限を超えるプレイヤーIDが存在する場合は、外部キーを外す前に中止する。
DROP TEMPORARY TABLE IF EXISTS migration_require_player_id_range;
CREATE TEMPORARY TABLE migration_require_player_id_range (
    valid BOOLEAN NOT NULL,
    CONSTRAINT chk_migration_require_player_id_range CHECK (valid = TRUE)
);
INSERT INTO migration_require_player_id_range (valid)
SELECT COALESCE(MAX(id), 0) <= 16777215
FROM players;
DROP TEMPORARY TABLE migration_require_player_id_range;

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

ALTER TABLE players MODIFY COLUMN id MEDIUMINT UNSIGNED NOT NULL AUTO_INCREMENT;
ALTER TABLE users MODIFY COLUMN player_id MEDIUMINT UNSIGNED NULL;
ALTER TABLE player_course_records MODIFY COLUMN player_id MEDIUMINT UNSIGNED NOT NULL;
ALTER TABLE player_favorite_songs MODIFY COLUMN player_id MEDIUMINT UNSIGNED NOT NULL;
ALTER TABLE player_honors MODIFY COLUMN player_id MEDIUMINT UNSIGNED NOT NULL;
ALTER TABLE player_latest_updates MODIFY COLUMN player_id MEDIUMINT UNSIGNED NOT NULL;
ALTER TABLE player_locked_songs MODIFY COLUMN player_id MEDIUMINT UNSIGNED NOT NULL;
ALTER TABLE player_metric_histories MODIFY COLUMN player_id MEDIUMINT UNSIGNED NOT NULL;
ALTER TABLE player_record_histories MODIFY COLUMN player_id MEDIUMINT UNSIGNED NOT NULL;
ALTER TABLE player_records MODIFY COLUMN player_id MEDIUMINT UNSIGNED NOT NULL;
ALTER TABLE player_worldsend_record_histories MODIFY COLUMN player_id MEDIUMINT UNSIGNED NOT NULL;
ALTER TABLE player_worldsend_records MODIFY COLUMN player_id MEDIUMINT UNSIGNED NOT NULL;

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
