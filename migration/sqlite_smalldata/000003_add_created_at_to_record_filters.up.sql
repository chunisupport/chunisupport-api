PRAGMA foreign_keys = OFF;

CREATE TABLE record_filters_new (
    id BLOB NOT NULL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    filter_value_gzip BLOB NOT NULL,
    is_worldsend BOOLEAN NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO record_filters_new (id, user_id, name, filter_value_gzip, is_worldsend, created_at, updated_at)
SELECT id, user_id, name, filter_value_gzip, is_worldsend, COALESCE(updated_at, CURRENT_TIMESTAMP), updated_at
FROM record_filters;

DROP TABLE record_filters;
ALTER TABLE record_filters_new RENAME TO record_filters;
CREATE INDEX IF NOT EXISTS idx_record_filters_user_id ON record_filters (user_id);

PRAGMA foreign_keys = ON;
