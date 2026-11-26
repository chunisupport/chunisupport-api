-- CHUNITHM-NET のプレイ履歴一覧を、プレイヤーごとに最新の最大50件保持する。
-- played_at はアプリケーションが UTC として解釈した時刻を保存する。
CREATE TABLE player_play_logs (
    player_id INT UNSIGNED NOT NULL,
    play_order TINYINT UNSIGNED NOT NULL,
    chart_id MEDIUMINT UNSIGNED NULL,
    worldsend_chart_id MEDIUMINT UNSIGNED NULL,
    jacket_img_id VARCHAR(20) NULL,
    difficulty_id TINYINT UNSIGNED NULL,
    score MEDIUMINT UNSIGNED NOT NULL,
    clear_lamp_id TINYINT UNSIGNED NOT NULL,
    combo_lamp_id TINYINT UNSIGNED NOT NULL,
    full_chain_id TINYINT UNSIGNED NOT NULL,
    played_at DATETIME NOT NULL,
    PRIMARY KEY (player_id, play_order),
    CONSTRAINT fk_player_play_logs_player
        FOREIGN KEY (player_id) REFERENCES players (id) ON DELETE CASCADE,
    CONSTRAINT fk_player_play_logs_chart
        FOREIGN KEY (chart_id) REFERENCES charts (id) ON DELETE CASCADE,
    CONSTRAINT fk_player_play_logs_worldsend_chart
        FOREIGN KEY (worldsend_chart_id) REFERENCES worldsend_charts (id) ON DELETE CASCADE,
    CONSTRAINT fk_player_play_logs_difficulty
        FOREIGN KEY (difficulty_id) REFERENCES difficulties (id),
    CONSTRAINT fk_player_play_logs_clear_lamp
        FOREIGN KEY (clear_lamp_id) REFERENCES clear_lamp_types (id),
    CONSTRAINT fk_player_play_logs_combo_lamp
        FOREIGN KEY (combo_lamp_id) REFERENCES combo_lamp_types (id),
    CONSTRAINT fk_player_play_logs_full_chain
        FOREIGN KEY (full_chain_id) REFERENCES full_chain_types (id),
    CONSTRAINT chk_player_play_logs_play_order CHECK (play_order BETWEEN 0 AND 49),
    CONSTRAINT chk_player_play_logs_score CHECK (score BETWEEN 0 AND 1010000),
    CONSTRAINT chk_player_play_logs_chart_type CHECK (
        (chart_id IS NULL OR difficulty_id IS NOT NULL) AND
        (worldsend_chart_id IS NULL OR difficulty_id IS NULL)
    )
) ENGINE = InnoDB;
