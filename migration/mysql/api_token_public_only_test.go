package mysql

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAddPublicOnlyToAPITokensMigration は既存トークンの挙動を変えないよう、既定値 FALSE で列を追加することを確認します。
func TestAddPublicOnlyToAPITokensMigration(t *testing.T) {
	// Given
	up := readNormalizedMigrationSQL(t, "000068_add_public_only_to_api_tokens.up.sql")
	down := readNormalizedMigrationSQL(t, "000068_add_public_only_to_api_tokens.down.sql")
	schema := readNormalizedMigrationSQL(t, "../schema_mysql.sql")

	// Then
	assert.Contains(t, up, "ADD COLUMN public_only BOOLEAN NOT NULL DEFAULT FALSE AFTER permission")
	assert.Contains(t, down, "DROP COLUMN public_only")
	assert.Contains(t, schema, "`public_only` tinyint(1) NOT NULL DEFAULT '0'")
}
