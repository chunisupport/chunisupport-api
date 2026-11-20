-- コース削除時の連鎖削除を解除し、元の外部キー制約へ戻す。
ALTER TABLE player_course_record_histories
    DROP FOREIGN KEY fk_player_course_record_histories_course;

ALTER TABLE player_course_record_histories
    ADD CONSTRAINT fk_player_course_record_histories_course
        FOREIGN KEY (course_id) REFERENCES courses (id);

ALTER TABLE player_course_records
    DROP FOREIGN KEY fk_player_course_records_course;

ALTER TABLE player_course_records
    ADD CONSTRAINT fk_player_course_records_course
        FOREIGN KEY (course_id) REFERENCES courses (id);
