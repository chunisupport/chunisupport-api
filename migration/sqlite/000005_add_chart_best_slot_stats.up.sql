CREATE TABLE chart_best_slot_stats_by_rating_band (
    chart_id INTEGER NOT NULL,
    rating_band_id INTEGER NOT NULL,
    best_player_count INTEGER NOT NULL,
    eligible_player_count INTEGER NOT NULL,
    best_player_percentage REAL,
    PRIMARY KEY (chart_id, rating_band_id),
    FOREIGN KEY (rating_band_id) REFERENCES rating_bands(id),
    CHECK (best_player_count >= 0),
    CHECK (eligible_player_count >= 0),
    CHECK (best_player_count <= eligible_player_count),
    CHECK (best_player_percentage IS NULL OR
           (best_player_percentage >= 0 AND best_player_percentage <= 100)),
    CHECK ((eligible_player_count = 0 AND best_player_percentage IS NULL) OR
           (eligible_player_count > 0 AND best_player_percentage IS NOT NULL))
);

CREATE INDEX idx_chart_best_slot_stats_ranking
ON chart_best_slot_stats_by_rating_band (
    rating_band_id,
    best_player_percentage DESC,
    best_player_count DESC,
    chart_id
);
