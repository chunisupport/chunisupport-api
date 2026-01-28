-- ジャンルマスタ
CREATE TABLE IF NOT EXISTS genres (
    id TINYINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(30) NOT NULL UNIQUE
);
INSERT INTO genres (name) VALUES
    ('POPS & ANIME'),
    ('niconico'),
    ('東方Project'),
    ('VARIETY'),
    ('イロドリミドリ'),
    ('ゲキマイ'),
    ('ORIGINAL')
ON DUPLICATE KEY UPDATE name = VALUES(name);

-- 譜面難易度マスタ
CREATE TABLE IF NOT EXISTS difficulties (
    id TINYINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(30) NOT NULL UNIQUE
);
INSERT INTO difficulties (name) VALUES
    ('BASIC'),
    ('ADVANCED'),
    ('EXPERT'),
    ('MASTER'),
    ('ULTIMA')
ON DUPLICATE KEY UPDATE name = VALUES(name);

-- クラスエンブレムマスタ
CREATE TABLE IF NOT EXISTS class_emblems (
    id TINYINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(10) NOT NULL UNIQUE
);
INSERT INTO class_emblems (name) VALUES
    ('1'),
    ('2'),
    ('3'),
    ('4'),
    ('5'),
    ('inf')
ON DUPLICATE KEY UPDATE name = VALUES(name);

-- クラスエンブレムベースマスタ
CREATE TABLE IF NOT EXISTS class_emblem_bases (
    id TINYINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(10) NOT NULL UNIQUE
);
INSERT INTO class_emblem_bases (name) VALUES
    ('1'),
    ('2'),
    ('3'),
    ('4'),
    ('5'),
    ('inf')
ON DUPLICATE KEY UPDATE name = VALUES(name);

-- クリアランプマスタ
CREATE TABLE IF NOT EXISTS clear_lamp_types (
    id TINYINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(20) NOT NULL UNIQUE
);
INSERT INTO clear_lamp_types (name) VALUES
    ('FAILED'),
    ('CLEAR'),
    ('HARD'),
    ('BRAVE'),
    ('ABSOLUTE'),
    ('CATASTROPHY')
ON DUPLICATE KEY UPDATE name = VALUES(name);

CREATE TABLE IF NOT EXISTS combo_lamp_types (
    id TINYINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(20) NOT NULL UNIQUE
);
INSERT INTO combo_lamp_types (name) VALUES
    ('NONE'),
    ('FULL COMBO'),
    ('ALL JUSTICE')
ON DUPLICATE KEY UPDATE name = VALUES(name);

-- 楽曲枠マスタ
CREATE TABLE IF NOT EXISTS slots (
    id TINYINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(30) NOT NULL UNIQUE
);
INSERT INTO slots (name) VALUES
    ('none'),
    ('best'),
    ('best_candidate'),
    ('new'),
    ('new_candidate')
ON DUPLICATE KEY UPDATE name = VALUES(name);

-- フルチェインランプマスタ
CREATE TABLE IF NOT EXISTS full_chain_types (
    id TINYINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(20) NOT NULL UNIQUE
);
INSERT INTO full_chain_types (name) VALUES
    ('NONE'),
    ('FULL CHAIN GOLD'),
    ('FULL CHAIN PLATINUM')
ON DUPLICATE KEY UPDATE name = VALUES(name);

-- 称号種類マスタ
CREATE TABLE IF NOT EXISTS honor_types (
    id TINYINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE
);
INSERT INTO honor_types (name) VALUES
    ('normal'),
    ('copper'),
    ('silver'),
    ('gold'),
    ('platina'),
    ('rainbow'),
    ('staff'),
    ('ongeki'),
    ('maimai'),
    ('expert'),
    ('master'),
    ('ultima'),
    ('sp'),
    ('phoenix_g'),
    ('phoenix_p'),
    ('phoenix_r')
ON DUPLICATE KEY UPDATE name = VALUES(name);

-- アカウントタイプマスタ
CREATE TABLE IF NOT EXISTS account_types (
    id TINYINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(20) NOT NULL UNIQUE
);
INSERT INTO account_types (name) VALUES
    ('PLAYER'),
    ('EDITOR'),
    ('ADMIN')
ON DUPLICATE KEY UPDATE name = VALUES(name);

-- 称号マスタ
CREATE TABLE IF NOT EXISTS honors (
    id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(500) NOT NULL,
    honor_type_id TINYINT UNSIGNED NOT NULL,
    image_url VARCHAR(255) NULL, -- 称号画像URL
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (honor_type_id) REFERENCES honor_types(id),
    UNIQUE KEY unique_honor_name_type (name, honor_type_id)
);

-- 曲テーブル
CREATE TABLE IF NOT EXISTS songs (
    id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    display_id CHAR(16) UNIQUE NOT NULL, -- 16進数16文字の表示用ID
    title VARCHAR(300) NOT NULL,
    artist VARCHAR(300) NOT NULL,
    genre_id TINYINT UNSIGNED NOT NULL,
    bpm INT,
    released_at DATE,
    official_idx VARCHAR(10) NOT NULL UNIQUE, -- 公式JSONから取得できるidxの値（重複防止のためUNIQUE制約を追加）
    jacket VARCHAR(20), -- ジャケット画像ファイル名
    is_worldsend BOOLEAN NOT NULL DEFAULT 0, -- WORLD'S END楽曲フラグ: 0=通常楽曲, 1=WORLD'S END
    is_deleted TINYINT(1) NOT NULL DEFAULT 0, -- 論理削除フラグ: 0=有効, 1=削除済み
    FOREIGN KEY (genre_id) REFERENCES genres(id),
    CHECK (bpm IS NULL OR bpm > 0)
);

-- 譜面テーブル
CREATE TABLE IF NOT EXISTS charts (
    id MEDIUMINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    song_id INT UNSIGNED NOT NULL,
    difficulty_id TINYINT UNSIGNED NOT NULL,
    const DECIMAL(3, 1) NOT NULL,
    is_const_unknown BOOLEAN NOT NULL DEFAULT 1,
    notes INT,
    FOREIGN KEY (song_id) REFERENCES songs(id) ON DELETE CASCADE,
    FOREIGN KEY (difficulty_id) REFERENCES difficulties(id),
    UNIQUE KEY unique_song_difficulty (song_id, difficulty_id),
    CHECK (const >= 0),
    CHECK (notes IS NULL OR notes >= 0)
);

-- WORLD'S END譜面テーブル
CREATE TABLE IF NOT EXISTS worldsend_charts (
    id MEDIUMINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    song_id INT UNSIGNED NOT NULL UNIQUE, -- WORLD'S ENDは1曲1譜面
    we_star TINYINT, -- 星の数（1～5）
    we_kanji CHAR(1), -- カテゴリ漢字（光、蔵、改、狂、etc.）
    notes INT,
    FOREIGN KEY (song_id) REFERENCES songs(id) ON DELETE CASCADE,
    CHECK (we_star IS NULL OR we_star BETWEEN 1 AND 5),
    CHECK (notes IS NULL OR notes >= 0)
);

-- ユーザテーブル（システムユーザー情報）
CREATE TABLE IF NOT EXISTS users (
    id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    account_type_id TINYINT UNSIGNED NOT NULL DEFAULT 1,
    player_id MEDIUMINT UNSIGNED DEFAULT NULL, -- CHUNITHMプレイヤー情報への参照（初回登録時などはNULL可能）
    is_deleted BOOLEAN NOT NULL DEFAULT 0, -- 論理削除フラグ
    is_private BOOLEAN NOT NULL DEFAULT 0, -- プライバシー設定: 0=公開, 1=非公開（他ユーザーから見えない）
    is_suspicious BOOLEAN NOT NULL DEFAULT 0, -- 不審アカウントフラグ: 0=正常, 1=不審

    UNIQUE KEY uq_users_player_id (player_id),
    FOREIGN KEY (account_type_id) REFERENCES account_types(id)
);

-- プレイヤーテーブル（CHUNITHMプレイヤー情報）
CREATE TABLE IF NOT EXISTS players (
    id MEDIUMINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id INT UNSIGNED NOT NULL,
    player_name VARCHAR(20) NOT NULL,
    player_level INT NOT NULL,
    official_player_rating DECIMAL(4,2) NULL, -- 公式データから取得したレーティング
    calculated_player_rating DECIMAL(6,4) NULL, -- スコア等から計算したレーティング
    new_average_rating DECIMAL(6,4) NULL, -- 新曲枠の平均レーティング
    best_average_rating DECIMAL(6,4) NULL, -- ベスト枠の平均レーティング
    class_emblem_id TINYINT UNSIGNED,
    class_emblem_base_id TINYINT UNSIGNED,
    last_played_at DATETIME NULL,
    overpower_value DECIMAL(8, 2) NULL,
    overpower_percentage DECIMAL(5, 2) NULL,
    team_name VARCHAR(50) NULL,
    team_color VARCHAR(20) NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (class_emblem_id) REFERENCES class_emblems(id),
    FOREIGN KEY (class_emblem_base_id) REFERENCES class_emblem_bases(id),
    UNIQUE KEY uq_players_user_id (user_id),
    CHECK (player_level >= 1)
);

-- プレイヤー称号中間テーブル（プレイヤーと称号のリレーション）
CREATE TABLE IF NOT EXISTS player_honors (
    player_id MEDIUMINT UNSIGNED NOT NULL,
    honor_id INT UNSIGNED NOT NULL,
    slot TINYINT NOT NULL, -- 称号スロット: 1=上段, 2=中段, 3=下段
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    PRIMARY KEY (player_id, slot),
    FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE,
    FOREIGN KEY (honor_id) REFERENCES honors(id),
    CHECK (slot BETWEEN 1 AND 3)
);

-- セッションテーブル
CREATE TABLE IF NOT EXISTS sessions (
    id VARCHAR(255) PRIMARY KEY, -- セッションID (例: UUID)
    user_id INT UNSIGNED NOT NULL, -- ユーザーID
    expires_at TIMESTAMP NOT NULL, -- セッションの有効期限
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, -- 作成日時
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- APIトークンテーブル
CREATE TABLE IF NOT EXISTS api_tokens (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id INT UNSIGNED NOT NULL,
    hashed_token CHAR(64) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE KEY uq_api_tokens_user_id (user_id),
    UNIQUE KEY uq_api_tokens_hashed_token (hashed_token)
);

-- プレイヤーレコードテーブル（プレイヤーの譜面ごとの記録情報）
CREATE TABLE IF NOT EXISTS player_records (
    player_id MEDIUMINT UNSIGNED NOT NULL,
    chart_id MEDIUMINT UNSIGNED NOT NULL,
    score MEDIUMINT UNSIGNED NOT NULL, -- スコア（0～1,010,000）
    clear_lamp_id TINYINT UNSIGNED NOT NULL DEFAULT 1, -- クリアランプ（デフォルト：FAILED）
    combo_lamp_id TINYINT UNSIGNED NOT NULL DEFAULT 1, -- コンボランプ（デフォルト：NONE）
    full_chain_id TINYINT UNSIGNED NOT NULL DEFAULT 1, -- フルチェイン（デフォルト：NONE）
    slot_id TINYINT UNSIGNED NOT NULL,
    slot_order TINYINT UNSIGNED NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE,
    FOREIGN KEY (chart_id) REFERENCES charts(id) ON DELETE CASCADE,
    FOREIGN KEY (clear_lamp_id) REFERENCES clear_lamp_types(id),
    FOREIGN KEY (combo_lamp_id) REFERENCES combo_lamp_types(id),
    FOREIGN KEY (full_chain_id) REFERENCES full_chain_types(id),
    FOREIGN KEY (slot_id) REFERENCES slots(id),
    PRIMARY KEY (player_id, chart_id), -- 同じプレイヤーが同じ譜面に対して複数のレコードを持てないようにする
    CHECK (score BETWEEN 0 AND 1010000),
    CHECK (slot_order IS NULL OR slot_order BETWEEN 1 AND 255)
);

-- プレイヤーWORLD'S ENDレコードテーブル（プレイヤーのWORLD'S END譜面ごとの記録情報）
CREATE TABLE IF NOT EXISTS player_worldsend_records (
    player_id MEDIUMINT UNSIGNED NOT NULL,
    worldsend_chart_id MEDIUMINT UNSIGNED NOT NULL,
    score MEDIUMINT UNSIGNED NOT NULL, -- スコア（0～1,010,000）
    clear_lamp_id TINYINT UNSIGNED NOT NULL DEFAULT 1, -- クリアランプ（デフォルト：FAILED）
    combo_lamp_id TINYINT UNSIGNED NOT NULL DEFAULT 1, -- コンボランプ（デフォルト：NONE）
    full_chain_id TINYINT UNSIGNED NOT NULL DEFAULT 1, -- フルチェイン（デフォルト：NONE）
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE,
    FOREIGN KEY (worldsend_chart_id) REFERENCES worldsend_charts(id) ON DELETE CASCADE,
    FOREIGN KEY (clear_lamp_id) REFERENCES clear_lamp_types(id),
    FOREIGN KEY (combo_lamp_id) REFERENCES combo_lamp_types(id),
    FOREIGN KEY (full_chain_id) REFERENCES full_chain_types(id),
    PRIMARY KEY (player_id, worldsend_chart_id), -- 同じプレイヤーが同じWORLD'S END譜面に対して複数のレコードを持てないようにする
    CHECK (score BETWEEN 0 AND 1010000)
);

CREATE TABLE IF NOT EXISTS user_recovery_codes (
    id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id INT UNSIGNED NOT NULL,
    code_hash BINARY(32) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_user_recovery_codes_user_id (user_id),
    UNIQUE KEY uq_user_recovery_codes_code_hash (code_hash),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- バージョンマスタ
CREATE TABLE IF NOT EXISTS versions (
    id TINYINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE,
    released_at DATE NOT NULL
);

INSERT INTO versions (name, released_at) VALUES
    ('CHUNITHM', '2015-07-16'),
    ('CHUNITHM PLUS', '2016-02-04'),
    ('CHUNITHM AIR', '2016-08-25'),
    ('CHUNITHM AIR PLUS', '2017-02-09'),
    ('CHUNITHM STAR', '2017-08-24'),
    ('CHUNITHM STAR PLUS', '2018-03-08'),
    ('CHUNITHM AMAZON', '2018-10-25'),
    ('CHUNITHM AMAZON PLUS', '2019-04-11'),
    ('CHUNITHM CRYSTAL', '2019-10-24'),
    ('CHUNITHM CRYSTAL PLUS', '2020-07-16'),
    ('CHUNITHM PARADISE', '2021-01-21'),
    ('CHUNITHM PARADISE LOST', '2021-05-13'),
    ('CHUNITHM NEW', '2021-11-04'),
    ('CHUNITHM NEW PLUS', '2022-04-14'),
    ('CHUNITHM SUN', '2022-10-13'),
    ('CHUNITHM SUN PLUS', '2023-05-11'),
    ('CHUNITHM LUMINOUS', '2023-12-14'),
    ('CHUNITHM LUMINOUS PLUS', '2024-06-20'),
    ('CHUNITHM VERSE', '2024-12-12'),
    ('CHUNITHM X-VERSE', '2025-07-16'),
    ('CHUNITHM X-VERSE-X', '2025-12-11')
ON DUPLICATE KEY UPDATE
    name = VALUES(name),
    released_at = VALUES(released_at);

-- インデックス（必要に応じて追加）
CREATE INDEX idx_songs_title ON songs(title); -- 楽曲タイトル検索を高速化
CREATE INDEX idx_songs_worldsend_deleted ON songs(is_worldsend, is_deleted); -- 楽曲一覧取得（WHERE is_worldsend = 0 AND is_deleted = 0）を高速化
CREATE INDEX idx_charts_song_id ON charts(song_id); -- 楽曲から紐づく譜面一覧を取得するJOINを高速化
CREATE INDEX idx_worldsend_charts_song_id ON worldsend_charts(song_id); -- WORLD'S END楽曲から譜面を取得するJOINを高速化
CREATE INDEX idx_users_deleted_private ON users(is_deleted, is_private, player_id); -- ユーザー一覧取得（WHERE is_deleted = FALSE AND is_private = FALSE AND player_id IS NOT NULL）を高速化
CREATE INDEX idx_players_player_name ON players(player_name); -- プレイヤー名検索やソートを高速化
CREATE INDEX idx_sessions_user_id ON sessions(user_id); -- ユーザー単位でのセッション失効処理を高速化
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at); -- 期限切れセッションの清掃ジョブ用
CREATE INDEX idx_sessions_user_expires ON sessions(user_id, expires_at); -- セッション数カウント（WHERE user_id = ? AND expires_at > NOW()）を高速化
CREATE INDEX idx_player_records_chart_id ON player_records(chart_id); -- 譜面別ランキングや自己ベスト比較を高速化
CREATE INDEX idx_player_records_updated_at ON player_records(updated_at); -- 最新更新順に並べる取得処理を高速化
CREATE INDEX idx_player_worldsend_records_worldsend_chart_id ON player_worldsend_records(worldsend_chart_id); -- WORLD'S END譜面別ランキングを高速化
CREATE INDEX idx_player_worldsend_records_updated_at ON player_worldsend_records(updated_at); -- WORLD'S END最新更新順に並べる取得処理を高速化
