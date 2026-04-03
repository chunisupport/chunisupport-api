ALTER TABLE users
    ADD COLUMN firebase_uid VARCHAR(128)
    CHARACTER SET ascii
    COLLATE ascii_bin
    NULL
    AFTER username;