CREATE TABLE goal_groups (
    id INT UNSIGNED NOT NULL AUTO_INCREMENT,
    user_id INT UNSIGNED NOT NULL,
    name VARCHAR(30) NOT NULL,
    sort_order SMALLINT UNSIGNED NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uq_goal_groups_user_name (user_id, name),
    UNIQUE KEY uq_goal_groups_user_id (user_id, id),
    KEY idx_goal_groups_user_sort_order_id (user_id, sort_order, id),
    CONSTRAINT fk_goal_groups_user_id FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

ALTER TABLE goals
    ADD COLUMN group_id INT UNSIGNED NULL AFTER user_id;

CREATE INDEX idx_goals_user_group_sort_order_id
    ON goals(user_id, group_id, sort_order, id);

ALTER TABLE goals
    ADD CONSTRAINT fk_goals_group_user
        FOREIGN KEY (user_id, group_id) REFERENCES goal_groups (user_id, id) ON DELETE RESTRICT;

DROP INDEX idx_goals_user_sort_order_id ON goals;
