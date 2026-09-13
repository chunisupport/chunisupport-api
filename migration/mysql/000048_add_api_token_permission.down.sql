ALTER TABLE api_tokens
    DROP CHECK chk_api_tokens_permission,
    DROP COLUMN permission;
