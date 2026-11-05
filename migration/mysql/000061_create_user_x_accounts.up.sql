CREATE TABLE user_x_accounts (
    user_id INT UNSIGNED NOT NULL,
    x_user_id VARCHAR(20) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    x_username VARCHAR(50) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    is_public TINYINT(1) NOT NULL DEFAULT 0,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id),
    CONSTRAINT fk_user_x_accounts_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);
