ALTER TABLE chart_stats_by_rating_band
    ADD COLUMN combo_ajc INTEGER NOT NULL DEFAULT 0;

ALTER TABLE worldsend_chart_stats_by_rating_band
    ADD COLUMN combo_ajc INTEGER NOT NULL DEFAULT 0;
