ALTER TABLE api_tokens
    DROP INDEX uq_api_tokens_display_id,
    DROP COLUMN display_id;
