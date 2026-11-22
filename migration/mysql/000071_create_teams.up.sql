-- プレイヤーデータから収集したCHUNITHMのチームを保存し、プレイヤーの所属チームを参照する。
-- チーム名は変更できるため、大文字英数字6桁のチームコードで一意に識別する。
CREATE TABLE teams (
    id INT UNSIGNED NOT NULL AUTO_INCREMENT,
    code CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    name VARCHAR(64) NOT NULL,
    color VARCHAR(16) CHARACTER SET ascii COLLATE ascii_bin NULL,
    data_collected_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uq_teams_code (code),
    CONSTRAINT chk_teams_code
        CHECK (REGEXP_LIKE(code, '^[A-Z0-9]{6}$', 'c'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- チームが削除されても所属プレイヤーは残し、未所属として扱う。
ALTER TABLE players
    ADD COLUMN team_id INT UNSIGNED NULL AFTER possession_id,
    ADD KEY idx_players_team_id (team_id),
    ADD CONSTRAINT fk_players_team_id
        FOREIGN KEY (team_id) REFERENCES teams (id) ON DELETE SET NULL;
