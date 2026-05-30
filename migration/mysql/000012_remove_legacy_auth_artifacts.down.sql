-- up で削除した旧認証データ自体は復元できないため、password_hash は空文字でカラム定義のみ復元します。
ALTER TABLE users
    ADD COLUMN password_hash VARCHAR(255) NOT NULL DEFAULT '' AFTER firebase_uid;

ALTER TABLE users
    MODIFY COLUMN password_hash VARCHAR(255) NOT NULL AFTER firebase_uid;

CREATE TABLE IF NOT EXISTS sessions (
    id BINARY(16) NOT NULL,
    user_id INT UNSIGNED NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_sessions_expires_at (expires_at),
    KEY idx_sessions_user_expires (user_id, expires_at),
    CONSTRAINT sessions_ibfk_1 FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS user_recovery_codes (
    id INT UNSIGNED NOT NULL AUTO_INCREMENT,
    user_id INT UNSIGNED NOT NULL,
    code_hash BINARY(32) NOT NULL,
    created_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uq_user_recovery_codes_code_hash (code_hash),
    KEY idx_user_recovery_codes_user_id (user_id),
    CONSTRAINT user_recovery_codes_ibfk_1 FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE EVENT IF NOT EXISTS cleanup_expired_sessions
ON SCHEDULE EVERY 1 HOUR
STARTS CURRENT_TIMESTAMP
DO
DELETE FROM sessions WHERE expires_at < NOW();