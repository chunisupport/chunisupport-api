CREATE TABLE achievement_types (
  id   TINYINT UNSIGNED NOT NULL AUTO_INCREMENT,
  code VARCHAR(30)  NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_achievement_types_code (code)
);

INSERT INTO achievement_types (code) VALUES
  ('rank_count'),
  ('score_count'),
  ('avg_score'),
  ('hardlamp_count'),
  ('combolamp_count'),
  ('total_score'),
  ('overpower_value'),
  ('overpower_percent');

CREATE TABLE goals (
  id                   INT UNSIGNED     NOT NULL AUTO_INCREMENT,
  user_id              INT UNSIGNED     NOT NULL,
  title                VARCHAR(30)      NOT NULL,
  achievement_type_id  TINYINT UNSIGNED NOT NULL,
  achievement_params   JSON             NOT NULL,
  attributes           JSON             NOT NULL,
  invert               BOOLEAN          NOT NULL DEFAULT FALSE,
  created_at           DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_goals_user_id (user_id),
  CONSTRAINT fk_goals_user_id             FOREIGN KEY (user_id)             REFERENCES users             (id) ON DELETE CASCADE,
  CONSTRAINT fk_goals_achievement_type_id FOREIGN KEY (achievement_type_id) REFERENCES achievement_types (id) ON DELETE RESTRICT
);
