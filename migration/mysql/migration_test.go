package mysql

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func readNormalizedMigrationSQL(t *testing.T, filename string) string {
	t.Helper()

	sqlBytes, err := os.ReadFile(filename)
	require.NoError(t, err)

	lines := strings.Split(string(sqlBytes), "\n")
	statements := make([]string, 0, len(lines))
	for _, line := range lines {
		if !strings.HasPrefix(strings.TrimSpace(line), "--") {
			statements = append(statements, line)
		}
	}

	return strings.Join(strings.Fields(strings.Join(statements, "\n")), " ")
}

func TestAddExtDevRoleUp_固定IDのEXTDEVを追加する(t *testing.T) {
	// Given
	upSQL := readNormalizedMigrationSQL(t, "000027_add_extdev_role.up.sql")

	// Then
	assert.Contains(t, upSQL, "INSERT INTO account_types (id, name) VALUES (4, 'EXTDEV')")
	assert.NotContains(t, upSQL, "ALTER TABLE account_types")
	assert.NotContains(t, upSQL, "ON DUPLICATE KEY UPDATE")
}

func TestAddExtDevRoleDown_EXTDEVユーザーをPLAYERへ移行してからロールを削除する(t *testing.T) {
	// Given
	downSQL := readNormalizedMigrationSQL(t, "000027_add_extdev_role.down.sql")
	updateStatement := "UPDATE users SET account_type_id = 1 WHERE account_type_id = 4;"
	deleteStatement := "DELETE FROM account_types WHERE id = 4;"

	// When
	updateIndex := strings.Index(downSQL, updateStatement)
	deleteIndex := strings.Index(downSQL, deleteStatement)

	// Then
	assert.NotEqual(t, -1, updateIndex)
	assert.NotEqual(t, -1, deleteIndex)
	assert.Less(t, updateIndex, deleteIndex)
	assert.NotContains(t, downSQL, "ALTER TABLE account_types")
}
