ALTER TABLE players
    MODIFY COLUMN official_player_rating DECIMAL(4, 2) NOT NULL,
    MODIFY COLUMN data_collected_at TIMESTAMP(6) NULL;

CREATE TABLE player_metric_histories (
    player_id MEDIUMINT UNSIGNED NOT NULL,
    official_rating DECIMAL(4, 2) NOT NULL,
    official_overpower DECIMAL(8, 2) NOT NULL,
    data_collected_at TIMESTAMP(6) NOT NULL,
    PRIMARY KEY (player_id, data_collected_at),
    CONSTRAINT fk_player_metric_histories_player
        FOREIGN KEY (player_id) REFERENCES players (id) ON DELETE CASCADE
) ENGINE = InnoDB;
