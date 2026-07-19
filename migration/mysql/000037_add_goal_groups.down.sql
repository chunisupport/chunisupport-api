UPDATE goals g
INNER JOIN (
    SELECT
        current_order.id,
        ROW_NUMBER() OVER (
            PARTITION BY current_order.user_id
            ORDER BY
                (current_order.group_id IS NULL) ASC,
                current_order.group_sort_order ASC,
                current_order.goal_sort_order ASC,
                current_order.id ASC
        ) AS new_sort_order
    FROM (
        SELECT
            g.id,
            g.user_id,
            g.group_id,
            gg.sort_order AS group_sort_order,
            g.sort_order AS goal_sort_order
        FROM goals g
        LEFT JOIN goal_groups gg
            ON gg.user_id = g.user_id
            AND gg.id = g.group_id
    ) AS current_order
) AS reordered ON reordered.id = g.id
SET g.sort_order = reordered.new_sort_order;

CREATE INDEX idx_goals_user_sort_order_id
    ON goals(user_id, sort_order, id);

ALTER TABLE goals
    DROP FOREIGN KEY fk_goals_group_user;

DROP INDEX idx_goals_user_group_sort_order_id ON goals;

ALTER TABLE goals
    DROP COLUMN group_id;

DROP TABLE goal_groups;
