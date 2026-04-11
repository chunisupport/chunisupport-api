PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS record_filters (
    id BLOB PRIMARY KEY, -- UUIDをBLOB形式で保存
    name TEXT NOT NULL,
    filter_value TEXT NOT NULL,
    is_worldsend BOOLEAN NOT NULL DEFAULT 0,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);