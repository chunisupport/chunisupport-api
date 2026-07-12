CREATE TABLE course_classes (
    id TINYINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(16) NOT NULL,
    sort_order TINYINT UNSIGNED NOT NULL,
    UNIQUE KEY uq_course_classes_name (name),
    UNIQUE KEY uq_course_classes_sort_order (sort_order)
);

INSERT INTO course_classes (name, sort_order) VALUES
    ('1', 0), ('2', 1), ('3', 2), ('4', 3), ('5', 4), ('inf', 5), ('extra', 6);

CREATE TABLE courses (
    id MEDIUMINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    official_idx VARCHAR(32) NOT NULL,
    name VARCHAR(255) NOT NULL,
    course_class_id TINYINT UNSIGNED NOT NULL,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uq_courses_official_idx (official_idx),
    KEY idx_courses_class_deleted_idx (course_class_id, is_deleted, official_idx),
    CONSTRAINT fk_courses_course_class FOREIGN KEY (course_class_id) REFERENCES course_classes(id)
);

CREATE TABLE player_course_records (
    player_id MEDIUMINT UNSIGNED NOT NULL,
    course_id MEDIUMINT UNSIGNED NOT NULL,
    score MEDIUMINT UNSIGNED NOT NULL,
    is_clear BOOLEAN NOT NULL,
    combo_lamp_id TINYINT UNSIGNED NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (player_id, course_id),
    KEY idx_player_course_records_course_id (course_id),
    KEY idx_player_course_records_player_updated_at (player_id, updated_at DESC),
    CONSTRAINT fk_player_course_records_player FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE,
    CONSTRAINT fk_player_course_records_course FOREIGN KEY (course_id) REFERENCES courses(id),
    CONSTRAINT fk_player_course_records_combo_lamp FOREIGN KEY (combo_lamp_id) REFERENCES combo_lamp_types(id),
    CONSTRAINT chk_player_course_records_score CHECK (score BETWEEN 0 AND 3030000)
);
