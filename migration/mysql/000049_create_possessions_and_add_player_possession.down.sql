ALTER TABLE players
    DROP FOREIGN KEY fk_players_possession_id,
    DROP INDEX idx_players_possession_id,
    DROP COLUMN possession_id;

DROP TABLE possessions;
