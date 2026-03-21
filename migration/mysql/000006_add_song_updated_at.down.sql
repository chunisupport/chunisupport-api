ALTER TABLE worldsend_charts
  DROP COLUMN updated_at;

ALTER TABLE charts
  DROP COLUMN updated_at;

ALTER TABLE songs
  DROP COLUMN updated_at;