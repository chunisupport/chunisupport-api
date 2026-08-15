CREATE TEMPORARY TABLE api_tokens_to_delete (
    id BIGINT UNSIGNED NOT NULL PRIMARY KEY
);

INSERT INTO api_tokens_to_delete (id)
SELECT id
FROM (
    SELECT
        id,
        ROW_NUMBER() OVER (
            PARTITION BY user_id
            ORDER BY created_at DESC, id DESC
        ) AS token_rank
    FROM api_tokens
) AS ranked
WHERE ranked.token_rank > 1;

DELETE api_tokens
FROM api_tokens
INNER JOIN api_tokens_to_delete
    ON api_tokens_to_delete.id = api_tokens.id;

DROP TEMPORARY TABLE api_tokens_to_delete;

ALTER TABLE api_tokens
    DROP INDEX uq_api_tokens_user_name,
    ADD UNIQUE KEY uq_api_tokens_user_id (user_id),
    DROP INDEX idx_api_tokens_user_created_id,
    DROP COLUMN last_used_at,
    DROP COLUMN token_prefix,
    DROP COLUMN name;
