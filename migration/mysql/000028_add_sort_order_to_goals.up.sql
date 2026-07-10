ALTER TABLE goals
    ADD COLUMN sort_order SMALLINT UNSIGNED NULL AFTER invert;

UPDATE goals g
INNER JOIN (
    SELECT
        id,
        ROW_NUMBER() OVER (
            PARTITION BY user_id
            ORDER BY created_at ASC, id ASC
        ) AS sort_order
    FROM goals
) ranked ON ranked.id = g.id
SET g.sort_order = ranked.sort_order;

ALTER TABLE goals
    MODIFY COLUMN sort_order SMALLINT UNSIGNED NOT NULL;

CREATE INDEX idx_goals_user_sort_order_id
    ON goals(user_id, sort_order, id);

DROP INDEX idx_goals_user_created_id ON goals;
