-- 期間限定のお知らせ。登録はADMIN向けの管理APIで行う（SQLでの手順は docs/app_announcements.md）。
-- 種別と項目の組み合わせは、種別の追加に追従しやすいようアプリ側で検証する。
CREATE TABLE app_announcements (
    id CHAR(16) NOT NULL,
    kind VARCHAR(16) NOT NULL,
    title VARCHAR(200) NOT NULL,
    body TEXT NULL,
    target_url VARCHAR(2048) NULL,
    starts_at DATETIME NOT NULL,
    expires_at DATETIME NOT NULL,
    revision TINYINT UNSIGNED NOT NULL DEFAULT 1,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_app_announcements_expires_at (expires_at),
    CONSTRAINT chk_app_announcements_period CHECK (starts_at < expires_at),
    CONSTRAINT chk_app_announcements_revision CHECK (revision >= 1)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
