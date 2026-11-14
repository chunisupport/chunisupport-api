-- 楽曲・譜面（songs / charts / worldsend_charts）の変更世代を1行で保持します。
-- APIとsong-batchは楽曲データを変更するトランザクション内でこの値を進め、
-- APIプロセスは値の変化だけを見て楽曲カタログキャッシュを再構築します。
-- updated_at は秒精度で同秒の変更や削除を区別できないため、変更検知には使いません。
CREATE TABLE song_catalog_revision (
    id TINYINT UNSIGNED NOT NULL,
    revision BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY (id),
    CONSTRAINT chk_song_catalog_revision_singleton
        CHECK (id = 1)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO song_catalog_revision (id, revision) VALUES (1, 0);
