CREATE TABLE IF NOT EXISTS user_recovery_codes (
    id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id INT UNSIGNED NOT NULL,
    code_hash BINARY(32) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_user_recovery_codes_user_id (user_id),
    UNIQUE KEY uq_user_recovery_codes_code_hash (code_hash),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
