CREATE TABLE record_filters (
    id BINARY(16) NOT NULL,
    user_id INT UNSIGNED NOT NULL,
    name VARCHAR(30) NOT NULL,
    filter_value_gzip BLOB NOT NULL,
    is_worldsend BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    PRIMARY KEY (id),
    CONSTRAINT fk_record_filters_user
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_record_filters_user_updated_id (user_id, updated_at DESC, id ASC)
);
