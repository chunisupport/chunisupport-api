CREATE TABLE possessions (
    id TINYINT UNSIGNED NOT NULL,
    name VARCHAR(10) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_possessions_name (name)
);

INSERT INTO possessions (id, name) VALUES
    (1, 'normal'),
    (2, 'silver'),
    (3, 'gold'),
    (4, 'platina'),
    (5, 'rainbow');

ALTER TABLE players
    ADD COLUMN possession_id TINYINT UNSIGNED NOT NULL DEFAULT 1 AFTER class_emblem_base_id,
    ADD KEY idx_players_possession_id (possession_id),
    ADD CONSTRAINT fk_players_possession_id FOREIGN KEY (possession_id) REFERENCES possessions(id);
