-- プレイヤーデータで観測した所属チームをプレイヤーに保存する。
-- チームはChuniSupport上で独立して管理せず、同じ team_code を持つプレイヤーの集まりとして扱う。
-- そのため teams テーブルは作らず、メンバーがいなくなればチームも自然に消える。
ALTER TABLE players
    ADD COLUMN team_code CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NULL AFTER possession_id,
    ADD COLUMN team_name VARCHAR(64) NULL AFTER team_code,
    ADD COLUMN team_color VARCHAR(16) CHARACTER SET ascii COLLATE ascii_bin NULL AFTER team_name,
    ADD KEY idx_players_team_code (team_code),
    ADD CONSTRAINT chk_players_team_code
        CHECK (team_code IS NULL OR REGEXP_LIKE(team_code, '^[A-Z0-9]{6}$', 'c')),
    -- チーム名とカラーは所属チームがある場合だけ持ち、所属チームには必ずチーム名がある。
    ADD CONSTRAINT chk_players_team
        CHECK ((team_code IS NULL AND team_name IS NULL AND team_color IS NULL)
            OR (team_code IS NOT NULL AND team_name IS NOT NULL));
