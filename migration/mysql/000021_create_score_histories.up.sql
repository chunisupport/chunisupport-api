CREATE TABLE player_record_histories (
    player_id MEDIUMINT UNSIGNED NOT NULL,
    chart_id MEDIUMINT UNSIGNED NOT NULL,
    score MEDIUMINT UNSIGNED NOT NULL,
    clear_lamp_id TINYINT UNSIGNED NOT NULL,
    combo_lamp_id TINYINT UNSIGNED NOT NULL,
    full_chain_id TINYINT UNSIGNED NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    PRIMARY KEY (player_id, chart_id, updated_at),
    CONSTRAINT fk_player_record_histories_player
        FOREIGN KEY (player_id) REFERENCES players (id) ON DELETE CASCADE,
    CONSTRAINT fk_player_record_histories_chart
        FOREIGN KEY (chart_id) REFERENCES charts (id) ON DELETE CASCADE,
    CONSTRAINT chk_player_record_histories_score
        CHECK (score BETWEEN 0 AND 1010000)
) ENGINE = InnoDB;

CREATE TABLE player_worldsend_record_histories (
    player_id MEDIUMINT UNSIGNED NOT NULL,
    worldsend_chart_id MEDIUMINT UNSIGNED NOT NULL,
    score MEDIUMINT UNSIGNED NOT NULL,
    clear_lamp_id TINYINT UNSIGNED NOT NULL,
    combo_lamp_id TINYINT UNSIGNED NOT NULL,
    full_chain_id TINYINT UNSIGNED NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    PRIMARY KEY (player_id, worldsend_chart_id, updated_at),
    CONSTRAINT fk_player_worldsend_record_histories_player
        FOREIGN KEY (player_id) REFERENCES players (id) ON DELETE CASCADE,
    CONSTRAINT fk_player_worldsend_record_histories_chart
        FOREIGN KEY (worldsend_chart_id) REFERENCES worldsend_charts (id) ON DELETE CASCADE,
    CONSTRAINT chk_player_worldsend_record_histories_score
        CHECK (score BETWEEN 0 AND 1010000)
) ENGINE = InnoDB;
