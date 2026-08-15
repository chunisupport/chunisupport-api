CREATE TABLE rating_bands (
    id TINYINT UNSIGNED PRIMARY KEY,
    label VARCHAR(16) NOT NULL,
    min_inclusive DECIMAL(6, 4) NULL,
    max_exclusive DECIMAL(6, 4) NULL,
    sort_order TINYINT UNSIGNED NOT NULL,
    UNIQUE KEY uq_rating_bands_label (label),
    UNIQUE KEY uq_rating_bands_sort_order (sort_order)
);

INSERT INTO rating_bands (id, label, min_inclusive, max_exclusive, sort_order) VALUES
(0, 'ALL', NULL, NULL, 0),
(1, '-14.9', NULL, 15.0, 1),
(2, '15.0', 15.0, 15.1, 2),
(3, '15.1', 15.1, 15.2, 3),
(4, '15.2', 15.2, 15.3, 4),
(5, '15.3', 15.3, 15.4, 5),
(6, '15.4', 15.4, 15.5, 6),
(7, '15.5', 15.5, 15.6, 7),
(8, '15.6', 15.6, 15.7, 8),
(9, '15.7', 15.7, 15.8, 9),
(10, '15.8', 15.8, 15.9, 10),
(11, '15.9', 15.9, 16.0, 11),
(12, '16.0', 16.0, 16.1, 12),
(13, '16.1', 16.1, 16.2, 13),
(14, '16.2', 16.2, 16.3, 14),
(15, '16.3', 16.3, 16.4, 15),
(16, '16.4', 16.4, 16.5, 16),
(17, '16.5', 16.5, 16.6, 17),
(18, '16.6', 16.6, 16.7, 18),
(19, '16.7', 16.7, 16.8, 19),
(20, '16.8', 16.8, 16.9, 20),
(21, '16.9', 16.9, 17.0, 21),
(22, '17.0', 17.0, 17.1, 22),
(23, '17.1', 17.1, 17.2, 23),
(24, '17.2', 17.2, 17.3, 24),
(25, '17.3', 17.3, 17.4, 25),
(26, '17.4', 17.4, 17.5, 26),
(27, '17.5', 17.5, 17.6, 27),
(28, '17.6+', 17.6, NULL, 28);

CREATE TABLE chart_stats_by_rating_band (
    chart_id MEDIUMINT UNSIGNED NOT NULL,
    rating_band_id TINYINT UNSIGNED NOT NULL,
    rank_aaal INT UNSIGNED NOT NULL DEFAULT 0,
    rank_s INT UNSIGNED NOT NULL DEFAULT 0,
    rank_sp INT UNSIGNED NOT NULL DEFAULT 0,
    rank_ss INT UNSIGNED NOT NULL DEFAULT 0,
    rank_ssp INT UNSIGNED NOT NULL DEFAULT 0,
    rank_sss INT UNSIGNED NOT NULL DEFAULT 0,
    rank_sssp INT UNSIGNED NOT NULL DEFAULT 0,
    rank_max INT UNSIGNED NOT NULL DEFAULT 0,
    combo_none INT UNSIGNED NOT NULL DEFAULT 0,
    combo_fc INT UNSIGNED NOT NULL DEFAULT 0,
    combo_aj INT UNSIGNED NOT NULL DEFAULT 0,
    combo_ajc INT UNSIGNED NOT NULL DEFAULT 0,
    clear_failed INT UNSIGNED NOT NULL DEFAULT 0,
    clear_clear INT UNSIGNED NOT NULL DEFAULT 0,
    clear_hard INT UNSIGNED NOT NULL DEFAULT 0,
    clear_brave INT UNSIGNED NOT NULL DEFAULT 0,
    clear_absolute INT UNSIGNED NOT NULL DEFAULT 0,
    clear_catastrophy INT UNSIGNED NOT NULL DEFAULT 0,
    average_score DOUBLE NULL,
    median_score DOUBLE NULL,
    player_count INT UNSIGNED NOT NULL DEFAULT 0,
    PRIMARY KEY (chart_id, rating_band_id),
    FOREIGN KEY (chart_id) REFERENCES charts(id) ON DELETE CASCADE,
    FOREIGN KEY (rating_band_id) REFERENCES rating_bands(id)
);

CREATE TABLE worldsend_chart_stats_by_rating_band (
    worldsend_chart_id MEDIUMINT UNSIGNED NOT NULL,
    rating_band_id TINYINT UNSIGNED NOT NULL,
    rank_aaal INT UNSIGNED NOT NULL DEFAULT 0,
    rank_s INT UNSIGNED NOT NULL DEFAULT 0,
    rank_sp INT UNSIGNED NOT NULL DEFAULT 0,
    rank_ss INT UNSIGNED NOT NULL DEFAULT 0,
    rank_ssp INT UNSIGNED NOT NULL DEFAULT 0,
    rank_sss INT UNSIGNED NOT NULL DEFAULT 0,
    rank_sssp INT UNSIGNED NOT NULL DEFAULT 0,
    rank_max INT UNSIGNED NOT NULL DEFAULT 0,
    combo_none INT UNSIGNED NOT NULL DEFAULT 0,
    combo_fc INT UNSIGNED NOT NULL DEFAULT 0,
    combo_aj INT UNSIGNED NOT NULL DEFAULT 0,
    combo_ajc INT UNSIGNED NOT NULL DEFAULT 0,
    clear_failed INT UNSIGNED NOT NULL DEFAULT 0,
    clear_clear INT UNSIGNED NOT NULL DEFAULT 0,
    clear_hard INT UNSIGNED NOT NULL DEFAULT 0,
    clear_brave INT UNSIGNED NOT NULL DEFAULT 0,
    clear_absolute INT UNSIGNED NOT NULL DEFAULT 0,
    clear_catastrophy INT UNSIGNED NOT NULL DEFAULT 0,
    average_score DOUBLE NULL,
    median_score DOUBLE NULL,
    player_count INT UNSIGNED NOT NULL DEFAULT 0,
    PRIMARY KEY (worldsend_chart_id, rating_band_id),
    FOREIGN KEY (worldsend_chart_id) REFERENCES worldsend_charts(id) ON DELETE CASCADE,
    FOREIGN KEY (rating_band_id) REFERENCES rating_bands(id)
);

CREATE TABLE chart_best_slot_stats_by_rating_band (
    chart_id MEDIUMINT UNSIGNED NOT NULL,
    rating_band_id TINYINT UNSIGNED NOT NULL,
    best_player_count INT UNSIGNED NOT NULL,
    eligible_player_count INT UNSIGNED NOT NULL,
    best_player_percentage DECIMAL(7, 4) NULL,
    PRIMARY KEY (chart_id, rating_band_id),
    FOREIGN KEY (chart_id) REFERENCES charts(id) ON DELETE CASCADE,
    FOREIGN KEY (rating_band_id) REFERENCES rating_bands(id),
    CHECK (best_player_count <= eligible_player_count),
    CHECK (best_player_percentage IS NULL OR best_player_percentage BETWEEN 0 AND 100),
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
