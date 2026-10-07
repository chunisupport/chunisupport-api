ALTER TABLE songs
    DROP FOREIGN KEY fk_songs_name_folder,
    DROP INDEX idx_songs_name_folder_id,
    DROP COLUMN name_folder_id;
