-- 旧認証データを破棄します。down ではスキーマのみ復元され、セッションやパスワードハッシュは復元されません。
DROP EVENT IF EXISTS cleanup_expired_sessions;

DROP TABLE IF EXISTS user_recovery_codes;

DROP TABLE IF EXISTS sessions;

ALTER TABLE users
    DROP COLUMN password_hash;