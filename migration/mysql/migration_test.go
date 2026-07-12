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

func TestAddSortOrderToGoalsUp_既存の作成順を連番へ移行する(t *testing.T) {
	// Given
	upSQL := readNormalizedMigrationSQL(t, "000028_add_sort_order_to_goals.up.sql")

	// Then
	assert.Contains(t, upSQL, "ADD COLUMN sort_order SMALLINT UNSIGNED NULL")
	assert.Contains(t, upSQL, "ROW_NUMBER() OVER ( PARTITION BY user_id ORDER BY created_at ASC, id ASC )")
	assert.Contains(t, upSQL, "MODIFY COLUMN sort_order SMALLINT UNSIGNED NOT NULL")
	assert.Contains(t, upSQL, "CREATE INDEX idx_goals_user_sort_order_id ON goals(user_id, sort_order, id)")
	assert.NotContains(t, upSQL, "UNIQUE")
	assert.Less(
		t,
		strings.Index(upSQL, "CREATE INDEX idx_goals_user_sort_order_id"),
		strings.Index(upSQL, "DROP INDEX idx_goals_user_created_id"),
	)
}

func TestAddSortOrderToGoalsDown_従来の一覧インデックスへ戻す(t *testing.T) {
	// Given
	downSQL := readNormalizedMigrationSQL(t, "000028_add_sort_order_to_goals.down.sql")

	// Then
	assert.Contains(t, downSQL, "DROP INDEX idx_goals_user_sort_order_id ON goals")
	assert.Contains(t, downSQL, "CREATE INDEX idx_goals_user_created_id ON goals(user_id, created_at, id)")
	assert.Contains(t, downSQL, "DROP COLUMN sort_order")
	assert.Less(
		t,
		strings.Index(downSQL, "CREATE INDEX idx_goals_user_created_id"),
		strings.Index(downSQL, "DROP INDEX idx_goals_user_sort_order_id"),
	)
}

func TestStoreOnlySPHonorImageURLsUp_通常称号をNULL化して重複を統合する(t *testing.T) {
	// Given
	upSQL := readNormalizedMigrationSQL(t, "000029_store_only_sp_honor_image_urls.up.sql")

	// Then
	assert.Contains(t, upSQL, "MODIFY COLUMN image_url VARCHAR(255) NULL")
	assert.Contains(t, upSQL, "SET h.image_url = NULL WHERE ht.name <> 'sp'")
	assert.Contains(t, upSQL, "SET h.name = SUBSTRING_INDEX(SUBSTRING_INDEX(SUBSTRING_INDEX(h.image_url, '#', 1), '?', 1), '/', -1)")
	assert.Contains(t, upSQL, "UPDATE player_honors AS ph")
	assert.Contains(t, upSQL, "SET ph.honor_id = duplicated_group.keep_id")
	assert.Contains(t, upSQL, "DELETE duplicated FROM honors AS duplicated")
	assert.Contains(t, upSQL, "ADD UNIQUE KEY unique_honor_name_type (name, honor_type_id)")
	assert.Less(t,
		strings.Index(upSQL, "MODIFY COLUMN image_url VARCHAR(255) NULL"),
		strings.Index(upSQL, "SET h.image_url = NULL"),
	)
}

func TestStoreOnlySPHonorImageURLsDown_NULLを空文字へ戻す(t *testing.T) {
	// Given
	downSQL := readNormalizedMigrationSQL(t, "000029_store_only_sp_honor_image_urls.down.sql")

	// Then
	assert.Contains(t, downSQL, "UPDATE honors SET image_url = '' WHERE image_url IS NULL")
	assert.Contains(t, downSQL, "DROP INDEX unique_honor_name_type")
	assert.Contains(t, downSQL, "MODIFY COLUMN image_url VARCHAR(255) NOT NULL DEFAULT ''")
}

func TestCreateCoursesUp_コースマスタとレコードを作成する(t *testing.T) {
	// Given
	upSQL := readNormalizedMigrationSQL(t, "000030_create_courses.up.sql")

	// Then
	assert.Contains(t, upSQL, "CREATE TABLE course_classes")
	assert.Contains(t, upSQL, "('extra', 6)")
	assert.Contains(t, upSQL, "CREATE TABLE courses")
	assert.Contains(t, upSQL, "name VARCHAR(255) NOT NULL")
	assert.Contains(t, upSQL, "is_deleted BOOLEAN NOT NULL DEFAULT FALSE")
	assert.Contains(t, upSQL, "CREATE TABLE player_course_records")
	assert.Contains(t, upSQL, "CHECK (score BETWEEN 0 AND 3030000)")
	assert.Contains(t, upSQL, "PRIMARY KEY (player_id, course_id)")
}

func TestCreateCoursesDown_参照順の逆順で削除する(t *testing.T) {
	// Given
	downSQL := readNormalizedMigrationSQL(t, "000030_create_courses.down.sql")

	// Then
	recordsIndex := strings.Index(downSQL, "DROP TABLE IF EXISTS player_course_records")
	coursesIndex := strings.Index(downSQL, "DROP TABLE IF EXISTS courses")
	classesIndex := strings.Index(downSQL, "DROP TABLE IF EXISTS course_classes")
	assert.Less(t, recordsIndex, coursesIndex)
	assert.Less(t, coursesIndex, classesIndex)
}
