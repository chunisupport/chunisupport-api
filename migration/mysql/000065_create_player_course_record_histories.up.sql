CREATE TABLE player_course_record_histories (
    player_id INT UNSIGNED NOT NULL,
    course_id MEDIUMINT UNSIGNED NOT NULL,
    score MEDIUMINT UNSIGNED NOT NULL,
    is_clear BOOLEAN NOT NULL,
    combo_lamp_id TINYINT UNSIGNED NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    PRIMARY KEY (player_id, course_id, updated_at),
    CONSTRAINT fk_player_course_record_histories_player
        FOREIGN KEY (player_id) REFERENCES players (id) ON DELETE CASCADE,
    CONSTRAINT fk_player_course_record_histories_course
        FOREIGN KEY (course_id) REFERENCES courses (id),
    CONSTRAINT chk_player_course_record_histories_score
        CHECK (score BETWEEN 0 AND 3030000)
) ENGINE = InnoDB;
