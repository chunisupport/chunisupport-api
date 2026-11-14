package mysql

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSongCatalogRevisionMigration は楽曲カタログの変更世代を1行だけで保持することを確認します。
func TestSongCatalogRevisionMigration(t *testing.T) {
	// Given
	up := readNormalizedMigrationSQL(t, "000064_create_song_catalog_revision.up.sql")
	down := readNormalizedMigrationSQL(t, "000064_create_song_catalog_revision.down.sql")
	schema := readNormalizedMigrationSQL(t, "../schema_mysql.sql")

	// When / Then
	assert.Contains(t, up, "CREATE TABLE song_catalog_revision")
	assert.Contains(t, up, "revision BIGINT UNSIGNED NOT NULL")
	assert.Contains(t, up, "CHECK (id = 1)")
	assert.Contains(t, up, "INSERT INTO song_catalog_revision (id, revision) VALUES (1, 0);")
	assert.NotContains(t, up, "AUTO_INCREMENT")
	assert.Contains(t, schema, "CREATE TABLE `song_catalog_revision`")
	assert.Contains(t, down, "DROP TABLE IF EXISTS song_catalog_revision")
}
