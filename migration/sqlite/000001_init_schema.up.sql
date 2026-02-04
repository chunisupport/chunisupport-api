CREATE TABLE IF NOT EXISTS rating_band (
    id INTEGER PRIMARY KEY,
    label TEXT NOT NULL,
    min_inclusive REAL,
    max_exclusive REAL,
    sort_order INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS chart_stats_by_rating_band (
    chart_id INTEGER NOT NULL,
    rating_band_id INTEGER NOT NULL,
    rank_aaal INTEGER NOT NULL,
    rank_s INTEGER NOT NULL,
    rank_sp INTEGER NOT NULL,
    rank_ss INTEGER NOT NULL,
    rank_ssp INTEGER NOT NULL,
    rank_sss INTEGER NOT NULL,
    rank_sssp INTEGER NOT NULL,
    rank_max INTEGER NOT NULL,
    combo_none INTEGER NOT NULL,
    combo_fc INTEGER NOT NULL,
    combo_aj INTEGER NOT NULL,
    clear_failed INTEGER NOT NULL,
    clear_clear INTEGER NOT NULL,
    clear_hard INTEGER NOT NULL,
    clear_brave INTEGER NOT NULL,
    clear_absolute INTEGER NOT NULL,
    clear_catastrophy INTEGER NOT NULL,
    PRIMARY KEY (chart_id, rating_band_id),
    FOREIGN KEY (rating_band_id) REFERENCES rating_band (id)
);
