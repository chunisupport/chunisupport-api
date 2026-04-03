ALTER TABLE users
    ADD COLUMN firebase_uid VARCHAR(128)
    CHARACTER SET ascii
    COLLATE ascii_bin
    NULL
    AFTER username;

-- add unique index for firebase_uid
CREATE UNIQUE INDEX uk_users_firebase_uid ON users (firebase_uid);