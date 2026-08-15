CREATE TABLE player_favorite_songs (
  player_id MEDIUMINT UNSIGNED NOT NULL,
  song_id INT UNSIGNED NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (player_id, song_id),
  KEY fk_player_favorite_songs_song_id (song_id),
  CONSTRAINT fk_player_favorite_songs_player_id
    FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE,
  CONSTRAINT fk_player_favorite_songs_song_id
    FOREIGN KEY (song_id) REFERENCES songs(id) ON DELETE CASCADE
);
