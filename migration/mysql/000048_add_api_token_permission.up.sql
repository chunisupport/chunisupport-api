ALTER TABLE api_tokens
    ADD COLUMN permission VARCHAR(10) NOT NULL DEFAULT 'read_write' AFTER name,
    ADD CONSTRAINT chk_api_tokens_permission CHECK (permission IN ('read', 'read_write'));
