ALTER TABLE api_tokens
    ADD COLUMN name VARCHAR(15) NOT NULL DEFAULT 'API Key' AFTER user_id,
    ADD INDEX idx_api_tokens_user_id (user_id),
    DROP INDEX uq_api_tokens_user_id;
