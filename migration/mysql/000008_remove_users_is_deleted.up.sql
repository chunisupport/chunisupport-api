-- users テーブルの論理削除フラグを削除します。
-- 物理削除への移行が完了したため、is_deleted カラムおよび関連インデックスを削除します。

ALTER TABLE users DROP INDEX idx_users_deleted_private;
ALTER TABLE users ADD INDEX idx_users_private (is_private, player_id);
ALTER TABLE users DROP COLUMN is_deleted;
