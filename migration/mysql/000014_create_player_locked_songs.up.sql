CREATE TABLE player_locked_songs (
  player_id MEDIUMINT UNSIGNED NOT NULL,
  song_id INT UNSIGNED NOT NULL,
  is_ultima BOOLEAN NOT NULL,
  PRIMARY KEY (player_id, song_id, is_ultima),
  CONSTRAINT fk_player_locked_songs_player_id FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE,
  CONSTRAINT fk_player_locked_songs_song_id FOREIGN KEY (song_id) REFERENCES songs(id) ON DELETE CASCADE
);
