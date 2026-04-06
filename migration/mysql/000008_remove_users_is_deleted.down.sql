-- users テーブルの論理削除フラグを復元します。

ALTER TABLE users ADD COLUMN is_deleted TINYINT(1) NOT NULL DEFAULT 0;
ALTER TABLE users DROP INDEX idx_users_private;
ALTER TABLE users ADD INDEX idx_users_deleted_private (is_deleted, is_private, player_id);
