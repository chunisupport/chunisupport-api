-- overpower_value, overpower_percentage の精度を schema_mysql.sql の定義に合わせる
ALTER TABLE players
    MODIFY COLUMN overpower_value DECIMAL(9, 3) NULL,
    MODIFY COLUMN overpower_percentage DECIMAL(7, 4) NULL;
