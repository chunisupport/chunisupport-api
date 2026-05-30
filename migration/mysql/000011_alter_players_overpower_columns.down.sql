-- overpower_value, overpower_percentage の精度を 000001_init_schema の定義に戻す
ALTER TABLE players
    MODIFY COLUMN overpower_value DECIMAL(8, 2) NULL,
    MODIFY COLUMN overpower_percentage DECIMAL(5, 2) NULL;
