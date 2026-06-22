PRAGMA foreign_keys = OFF;

DROP TABLE IF EXISTS record_filters;

CREATE TABLE IF NOT EXISTS record_filters (
    id BLOB NOT NULL PRIMARY KEY, -- UUIDをBLOB形式で保存
    user_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    filter_value_gzip BLOB NOT NULL,
    is_worldsend BOOLEAN NOT NULL DEFAULT 0,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_record_filters_user_id ON record_filters (user_id);

PRAGMA foreign_keys = ON;
