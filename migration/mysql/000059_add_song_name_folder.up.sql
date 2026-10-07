ALTER TABLE songs
    ADD COLUMN name_folder_id TINYINT UNSIGNED NOT NULL DEFAULT 17 AFTER reading,
    ADD KEY idx_songs_name_folder_id (name_folder_id),
    ADD CONSTRAINT fk_songs_name_folder FOREIGN KEY (name_folder_id) REFERENCES name_folders (id);
