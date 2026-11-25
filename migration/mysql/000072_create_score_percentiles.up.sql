-- 譜面統計バッチの変更世代を1行で保持します。
-- chart-stats-batch は統計テーブルを入れ替えるトランザクション内でこの値を進め、
-- APIプロセスは値の変化だけを見て上位%スナップショットキャッシュを再構築します。
CREATE TABLE chart_stats_revision (
    id TINYINT UNSIGNED NOT NULL,
    revision BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY (id),
    CONSTRAINT chk_chart_stats_revision_singleton
        CHECK (id = 1)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO chart_stats_revision (id, revision) VALUES (1, 0);

-- 通常譜面のスコア→上位%境界データ（ALL帯、全プレイ記録対象）
CREATE TABLE chart_score_percentiles (
    chart_id MEDIUMINT UNSIGNED NOT NULL,
    thresholds BLOB NOT NULL,
    PRIMARY KEY (chart_id),
    CONSTRAINT fk_chart_score_percentiles_chart
        FOREIGN KEY (chart_id) REFERENCES charts (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- WORLD'S END譜面のスコア→上位%境界データ（ALL帯、全プレイ記録対象）
CREATE TABLE worldsend_chart_score_percentiles (
    worldsend_chart_id MEDIUMINT UNSIGNED NOT NULL,
    thresholds BLOB NOT NULL,
    PRIMARY KEY (worldsend_chart_id),
    CONSTRAINT fk_worldsend_chart_score_percentiles_chart
        FOREIGN KEY (worldsend_chart_id) REFERENCES worldsend_charts (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- コースのスコア→上位%境界データ（ALL帯、全プレイ記録対象）
CREATE TABLE course_score_percentiles (
    course_id MEDIUMINT UNSIGNED NOT NULL,
    thresholds BLOB NOT NULL,
    PRIMARY KEY (course_id),
    CONSTRAINT fk_course_score_percentiles_course
        FOREIGN KEY (course_id) REFERENCES courses (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
