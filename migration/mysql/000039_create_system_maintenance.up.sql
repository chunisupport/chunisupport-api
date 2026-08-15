CREATE TABLE system_maintenance (
    id TINYINT UNSIGNED NOT NULL,
    enabled BOOLEAN NOT NULL,
    comment VARCHAR(1000) NOT NULL DEFAULT '',
    updated_by_user_id INT UNSIGNED NULL,
    updated_at DATETIME(6) NOT NULL,
    PRIMARY KEY (id),
    CONSTRAINT fk_system_maintenance_updated_by_user
        FOREIGN KEY (updated_by_user_id)
        REFERENCES users (id)
        ON DELETE SET NULL,
    CONSTRAINT chk_system_maintenance_singleton
        CHECK (id = 1)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO system_maintenance (id, enabled, comment, updated_by_user_id, updated_at)
VALUES (1, FALSE, '', NULL, UTC_TIMESTAMP(6));
