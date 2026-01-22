-- 譜面統計テーブル
-- レーティング帯別（15.0-17.6、17.7+）にランク・ランプの人数を集計
-- 対象: 譜面定数10.0以上の譜面のみ
CREATE TABLE IF NOT EXISTS chart_statistics (
    -- 譜面ID (charts.id)
    chart_id MEDIUMINT UNSIGNED NOT NULL,
    -- レーティング帯（10倍した整数: 150-176, 177は17.7+を表す）
    rating_tier SMALLINT UNSIGNED NOT NULL,
    
    -- ランク別人数
    rank_s_count MEDIUMINT UNSIGNED NOT NULL DEFAULT 0,       -- Sランク人数
    rank_s_plus_count MEDIUMINT UNSIGNED NOT NULL DEFAULT 0,  -- S+ランク人数
    rank_ss_count MEDIUMINT UNSIGNED NOT NULL DEFAULT 0,      -- SSランク人数
    rank_ss_plus_count MEDIUMINT UNSIGNED NOT NULL DEFAULT 0, -- SS+ランク人数
    rank_sss_count MEDIUMINT UNSIGNED NOT NULL DEFAULT 0,     -- SSSランク人数
    rank_sss_plus_count MEDIUMINT UNSIGNED NOT NULL DEFAULT 0,-- SSS+ランク人数
    
    -- ランプ別人数
    lamp_aj_count MEDIUMINT UNSIGNED NOT NULL DEFAULT 0,      -- ALL JUSTICE人数
    lamp_fc_count MEDIUMINT UNSIGNED NOT NULL DEFAULT 0,      -- FULL COMBO人数
    lamp_other_count MEDIUMINT UNSIGNED NOT NULL DEFAULT 0,   -- その他ランプ人数
    
    -- メタデータ
    total_count MEDIUMINT UNSIGNED NOT NULL DEFAULT 0,        -- 合計人数（検算用）
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    PRIMARY KEY (chart_id, rating_tier),
    INDEX idx_chart_statistics_chart_id (chart_id),
    CONSTRAINT fk_chart_statistics_chart FOREIGN KEY (chart_id) REFERENCES charts(id) ON DELETE CASCADE
);
