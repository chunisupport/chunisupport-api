-- セッション自動クリーンアップイベントの作成
-- 1時間ごとに期限切れのセッションを削除します

-- イベントスケジューラの有効化が必要です
-- MySQL設定で event_scheduler=ON を設定するか、以下のコマンドを実行してください：
-- SET GLOBAL event_scheduler = ON;

-- 期限切れセッションを削除するイベント
CREATE EVENT IF NOT EXISTS cleanup_expired_sessions
ON SCHEDULE EVERY 1 HOUR
STARTS CURRENT_TIMESTAMP
DO
DELETE FROM sessions WHERE expires_at < NOW();
