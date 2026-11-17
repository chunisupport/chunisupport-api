CREATE TABLE course_stats_by_rating_band (
    course_id MEDIUMINT UNSIGNED NOT NULL,
    rating_band_id TINYINT UNSIGNED NOT NULL,
    combo_none INT UNSIGNED NOT NULL DEFAULT 0,
    combo_fc INT UNSIGNED NOT NULL DEFAULT 0,
    combo_aj INT UNSIGNED NOT NULL DEFAULT 0,
    clear_failed INT UNSIGNED NOT NULL DEFAULT 0,
    clear_clear INT UNSIGNED NOT NULL DEFAULT 0,
    average_score DOUBLE NULL,
    median_score DOUBLE NULL,
    player_count INT UNSIGNED NOT NULL DEFAULT 0,
    PRIMARY KEY (course_id, rating_band_id),
    CONSTRAINT fk_course_stats_by_rating_band_course
        FOREIGN KEY (course_id) REFERENCES courses (id) ON DELETE CASCADE,
    CONSTRAINT fk_course_stats_by_rating_band_rating_band
        FOREIGN KEY (rating_band_id) REFERENCES rating_bands (id)
) ENGINE = InnoDB;
