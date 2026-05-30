-- players.user_id に外部キー制約を追加
ALTER TABLE players
ADD CONSTRAINT fk_players_user_id
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- users.player_id に外部キー制約を追加（削除時はNULLに設定）
ALTER TABLE users
ADD CONSTRAINT fk_users_player_id
FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE SET NULL;
