CREATE TABLE player_latest_updates (
    player_id MEDIUMINT UNSIGNED NOT NULL,
    schema_version INT UNSIGNED NOT NULL,
    result_gzip MEDIUMBLOB NOT NULL,
    source_updated_at DATETIME(6) NOT NULL,
    imported_at DATETIME(6) NOT NULL,
    body_hash CHAR(64) NOT NULL,

    PRIMARY KEY (player_id),
    CONSTRAINT fk_player_latest_updates_player
        FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE
);
