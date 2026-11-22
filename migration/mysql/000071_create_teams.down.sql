-- プレイヤーの所属チーム参照を先に削除してからチーム本体を削除する。
ALTER TABLE players
    DROP FOREIGN KEY fk_players_team_id,
    DROP KEY idx_players_team_id,
    DROP COLUMN team_id;

DROP TABLE IF EXISTS teams;
