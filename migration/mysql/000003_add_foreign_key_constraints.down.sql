-- 外部キー制約を削除（追加時と逆順）
ALTER TABLE users DROP FOREIGN KEY fk_users_player_id;
ALTER TABLE players DROP FOREIGN KEY fk_players_user_id;
