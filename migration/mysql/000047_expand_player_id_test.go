package mysql

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExpandPlayerIDDown_MEDIUMINT範囲を外部キー削除前に検証する(t *testing.T) {
	downSQL := readNormalizedMigrationSQL(t, "000047_expand_player_id.down.sql")
	createCheckTable := "CREATE TEMPORARY TABLE migration_require_player_id_range"
	rangeCheck := "INSERT INTO migration_require_player_id_range (valid) SELECT COALESCE(MAX(id), 0) <= 16777215 FROM players;"
	cleanupAfterCheck := "DROP TEMPORARY TABLE migration_require_player_id_range;"
	firstForeignKeyDrop := "ALTER TABLE users DROP FOREIGN KEY fk_users_player_id;"

	assert.Contains(t, downSQL, "CONSTRAINT chk_migration_require_player_id_range CHECK (valid = TRUE)")
	assert.Contains(t, downSQL, rangeCheck)
	assert.Less(t, strings.Index(downSQL, createCheckTable), strings.Index(downSQL, rangeCheck))
	assert.Less(t, strings.Index(downSQL, rangeCheck), strings.Index(downSQL, cleanupAfterCheck))
	assert.Less(t, strings.Index(downSQL, rangeCheck), strings.Index(downSQL, firstForeignKeyDrop))
}
