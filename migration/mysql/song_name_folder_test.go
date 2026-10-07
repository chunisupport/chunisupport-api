package mysql

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSongNameFolderMigration_数字を初期値にして必須の外部キーを追加する(t *testing.T) {
	master := readNormalizedMigrationSQL(t, "000058_create_name_folders.up.sql")
	assert.Contains(t, master, "(17, 'NUMBER', '数字', 17)")
	add := readNormalizedMigrationSQL(t, "000059_add_song_name_folder.up.sql")
	assert.Contains(t, add, "name_folder_id TINYINT UNSIGNED NOT NULL DEFAULT 17")
	assert.Contains(t, add, "FOREIGN KEY (name_folder_id) REFERENCES name_folders (id)")
	assert.Contains(t, add, "ADD KEY idx_songs_name_folder_id (name_folder_id)")
	assert.NotContains(t, add, "UPDATE songs")
	revert := readNormalizedMigrationSQL(t, "000059_add_song_name_folder.down.sql")
	assert.Contains(t, revert, "DROP FOREIGN KEY fk_songs_name_folder")
	assert.Contains(t, revert, "DROP INDEX idx_songs_name_folder_id")
	assert.Contains(t, revert, "DROP COLUMN name_folder_id")
}
