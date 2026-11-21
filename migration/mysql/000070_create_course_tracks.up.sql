-- コース構成のトラックと、固定・プール指定された通常譜面・WORLD'S END譜面を保存する。
CREATE TABLE course_tracks (
    course_id MEDIUMINT UNSIGNED NOT NULL,
    track_no TINYINT UNSIGNED NOT NULL,
    selection_type VARCHAR(20) NOT NULL,
    filter JSON NULL,
    PRIMARY KEY (course_id, track_no),
    CONSTRAINT fk_course_tracks_course
        FOREIGN KEY (course_id) REFERENCES courses (id) ON DELETE CASCADE,
    CONSTRAINT chk_course_tracks_track_no
        CHECK (track_no BETWEEN 1 AND 3),
    CONSTRAINT chk_course_tracks_selection_type
        CHECK (selection_type IN ('FIXED', 'RANDOM_FILTER', 'RANDOM_POOL')),
    CONSTRAINT chk_course_tracks_filter
        CHECK (
            (selection_type = 'RANDOM_FILTER' AND filter IS NOT NULL)
            OR (selection_type <> 'RANDOM_FILTER' AND filter IS NULL)
        )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE course_track_charts (
    course_id MEDIUMINT UNSIGNED NOT NULL,
    track_no TINYINT UNSIGNED NOT NULL,
    chart_id MEDIUMINT UNSIGNED NOT NULL,
    PRIMARY KEY (course_id, track_no, chart_id),
    KEY idx_course_track_charts_chart_id (chart_id),
    CONSTRAINT fk_course_track_charts_track
        FOREIGN KEY (course_id, track_no) REFERENCES course_tracks (course_id, track_no) ON DELETE CASCADE,
    CONSTRAINT fk_course_track_charts_chart
        FOREIGN KEY (chart_id) REFERENCES charts (id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE course_track_worldsend_charts (
    course_id MEDIUMINT UNSIGNED NOT NULL,
    track_no TINYINT UNSIGNED NOT NULL,
    worldsend_chart_id MEDIUMINT UNSIGNED NOT NULL,
    PRIMARY KEY (course_id, track_no, worldsend_chart_id),
    KEY idx_course_track_worldsend_charts_chart_id (worldsend_chart_id),
    CONSTRAINT fk_course_track_worldsend_charts_track
        FOREIGN KEY (course_id, track_no) REFERENCES course_tracks (course_id, track_no) ON DELETE CASCADE,
    CONSTRAINT fk_course_track_worldsend_charts_chart
        FOREIGN KEY (worldsend_chart_id) REFERENCES worldsend_charts (id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
