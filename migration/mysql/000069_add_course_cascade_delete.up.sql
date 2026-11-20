-- コース物理削除時に、紐づくコーススコアとスコア履歴も連鎖削除できるようにする。
ALTER TABLE player_course_records
    DROP FOREIGN KEY fk_player_course_records_course;

ALTER TABLE player_course_records
    ADD CONSTRAINT fk_player_course_records_course
        FOREIGN KEY (course_id) REFERENCES courses (id) ON DELETE CASCADE;

ALTER TABLE player_course_record_histories
    DROP FOREIGN KEY fk_player_course_record_histories_course;

ALTER TABLE player_course_record_histories
    ADD CONSTRAINT fk_player_course_record_histories_course
        FOREIGN KEY (course_id) REFERENCES courses (id) ON DELETE CASCADE;
