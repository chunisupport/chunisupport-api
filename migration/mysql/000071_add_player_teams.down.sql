ALTER TABLE players
    DROP CHECK chk_players_team,
    DROP CHECK chk_players_team_code,
    DROP KEY idx_players_team_code,
    DROP COLUMN team_color,
    DROP COLUMN team_name,
    DROP COLUMN team_code;
