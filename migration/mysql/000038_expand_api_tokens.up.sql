ALTER TABLE api_tokens
    ADD COLUMN name VARCHAR(50) NOT NULL DEFAULT '既存のトークン' AFTER user_id,
    ADD COLUMN token_prefix CHAR(5) NULL AFTER hashed_token,
    ADD COLUMN last_used_at DATETIME NULL AFTER token_prefix;

ALTER TABLE api_tokens
    ADD INDEX idx_api_tokens_user_created_id (user_id, created_at DESC, id DESC),
    DROP INDEX uq_api_tokens_user_id,
    ADD UNIQUE KEY uq_api_tokens_user_name (user_id, name);
