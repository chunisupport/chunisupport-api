-- テーブルを逆順で削除(外部キー制約を考慮)
-- インデックスは自動的に削除されるため明示的な削除は不要

-- 外部キーチェックを一時無効化して循環依存を回避
SET FOREIGN_KEY_CHECKS = 0;

-- バージョンマスタを削除
DROP TABLE IF EXISTS versions;

DROP TABLE IF EXISTS user_recovery_codes;

DROP TABLE IF EXISTS player_worldsend_records;
DROP TABLE IF EXISTS player_records;
DROP TABLE IF EXISTS api_tokens;
DROP TABLE IF EXISTS sessions;

-- 中間テーブル
DROP TABLE IF EXISTS player_honors;

-- 親テーブル（外部キー制約を削除済みのため、順序は逆順でなくても大丈夫）
DROP TABLE IF EXISTS players;
DROP TABLE IF EXISTS users;

DROP TABLE IF EXISTS worldsend_charts;
DROP TABLE IF EXISTS charts;
DROP TABLE IF EXISTS songs;
DROP TABLE IF EXISTS honors;

-- マスターテーブル
DROP TABLE IF EXISTS slots;
DROP TABLE IF EXISTS account_types;
DROP TABLE IF EXISTS honor_types;
DROP TABLE IF EXISTS full_chain_types;
DROP TABLE IF EXISTS combo_lamp_types;
DROP TABLE IF EXISTS clear_lamp_types;
DROP TABLE IF EXISTS class_emblem_bases;
DROP TABLE IF EXISTS class_emblems;
DROP TABLE IF EXISTS difficulties;
DROP TABLE IF EXISTS genres;

SET FOREIGN_KEY_CHECKS = 1;
