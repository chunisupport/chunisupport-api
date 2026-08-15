CREATE INDEX idx_goals_user_created_id
    ON goals(user_id, created_at, id);

DROP INDEX idx_goals_user_sort_order_id ON goals;

ALTER TABLE goals
    DROP COLUMN sort_order;
