package mysql

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAddDisplayIDToAPITokensMigration は既存トークンへ表示用IDを採番してから一意制約を付けることを確認します。
func TestAddDisplayIDToAPITokensMigration(t *testing.T) {
	// Given
	up := readNormalizedMigrationSQL(t, "000067_add_display_id_to_api_tokens.up.sql")
	down := readNormalizedMigrationSQL(t, "000067_add_display_id_to_api_tokens.down.sql")
	schema := readNormalizedMigrationSQL(t, "../schema_mysql.sql")

	// Then
	assert.Contains(t, up, "ADD COLUMN display_id CHAR(16) NULL AFTER id")
	assert.Contains(t, up, "LOWER(HEX(RANDOM_BYTES(8)))")
	assert.Contains(t, up, "ADD UNIQUE KEY uq_api_tokens_display_id (display_id)")
	assert.Less(t, strings.Index(up, "LOWER(HEX(RANDOM_BYTES(8)))"), strings.Index(up, "MODIFY COLUMN display_id"))
	assert.Contains(t, down, "DROP INDEX uq_api_tokens_display_id")
	assert.Contains(t, down, "DROP COLUMN display_id")
	assert.Contains(t, schema, "UNIQUE KEY `uq_api_tokens_display_id` (`display_id`)")
}
