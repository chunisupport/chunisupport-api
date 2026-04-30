DELETE t1 FROM api_tokens t1
INNER JOIN api_tokens t2
    ON t1.user_id = t2.user_id
    AND t1.id < t2.id;

ALTER TABLE api_tokens
    ADD UNIQUE KEY uq_api_tokens_user_id (user_id),
    DROP INDEX idx_api_tokens_user_id,
    DROP COLUMN name;
