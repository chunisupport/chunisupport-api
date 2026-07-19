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

func TestAddGoalGroupsUp_グループとグループ内順序を追加する(t *testing.T) {
	// When
	upSQL := readNormalizedMigrationSQL(t, "000037_add_goal_groups.up.sql")

	// Then
	assert.Contains(t, upSQL, "CREATE TABLE goal_groups")
	assert.Contains(t, upSQL, "UNIQUE KEY uq_goal_groups_user_name (user_id, name)")
	assert.Contains(t, upSQL, "ADD COLUMN group_id INT UNSIGNED NULL AFTER user_id")
	assert.Contains(t, upSQL, "FOREIGN KEY (user_id, group_id) REFERENCES goal_groups (user_id, id) ON DELETE RESTRICT")
	assert.Contains(t, upSQL, "CREATE INDEX idx_goals_user_group_sort_order_id ON goals(user_id, group_id, sort_order, id)")
	assert.Less(t,
		strings.Index(upSQL, "CREATE INDEX idx_goals_user_group_sort_order_id"),
		strings.Index(upSQL, "ADD CONSTRAINT fk_goals_group_user"),
	)
	assert.Less(t,
		strings.Index(upSQL, "ADD CONSTRAINT fk_goals_group_user"),
		strings.Index(upSQL, "DROP INDEX idx_goals_user_sort_order_id"),
	)
}

func TestAddGoalGroupsDown_従来の目標順へ戻す(t *testing.T) {
	// When
	downSQL := readNormalizedMigrationSQL(t, "000037_add_goal_groups.down.sql")

	// Then
	assert.Contains(t, downSQL, "CREATE INDEX idx_goals_user_sort_order_id ON goals(user_id, sort_order, id)")
	assert.Contains(t, downSQL, "ROW_NUMBER() OVER ( PARTITION BY current_order.user_id")
	assert.Contains(t, downSQL, "(current_order.group_id IS NULL) ASC")
	assert.Contains(t, downSQL, "current_order.group_sort_order ASC")
	assert.Contains(t, downSQL, "current_order.goal_sort_order ASC")
	assert.Contains(t, downSQL, "SET g.sort_order = reordered.new_sort_order")
	assert.Contains(t, downSQL, "DROP FOREIGN KEY fk_goals_group_user")
	assert.Contains(t, downSQL, "DROP COLUMN group_id")
	assert.Contains(t, downSQL, "DROP TABLE goal_groups")
	assert.Less(t,
		strings.Index(downSQL, "SET g.sort_order = reordered.new_sort_order"),
		strings.Index(downSQL, "DROP FOREIGN KEY fk_goals_group_user"),
	)
	assert.Less(t,
		strings.Index(downSQL, "DROP FOREIGN KEY fk_goals_group_user"),
		strings.Index(downSQL, "DROP INDEX idx_goals_user_group_sort_order_id"),
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

func TestCreateChartStatisticsUp_MySQLへ統計テーブルを作成する(t *testing.T) {
	// Given
	upSQL := readNormalizedMigrationSQL(t, "000036_create_chart_statistics.up.sql")

	// Then
	assert.Contains(t, upSQL, "CREATE TABLE rating_bands")
	assert.Contains(t, upSQL, "CREATE TABLE chart_stats_by_rating_band")
	assert.Contains(t, upSQL, "CREATE TABLE worldsend_chart_stats_by_rating_band")
	assert.Contains(t, upSQL, "CREATE TABLE chart_best_slot_stats_by_rating_band")
	assert.Contains(t, upSQL, "FOREIGN KEY (chart_id) REFERENCES charts(id) ON DELETE CASCADE")
	assert.Contains(t, upSQL, "FOREIGN KEY (worldsend_chart_id) REFERENCES worldsend_charts(id) ON DELETE CASCADE")
	assert.Contains(t, upSQL, "(0, 'ALL', NULL, NULL, 0)")
}

func TestCreateChartStatisticsDown_依存順に統計テーブルを削除する(t *testing.T) {
	// Given
	downSQL := readNormalizedMigrationSQL(t, "000036_create_chart_statistics.down.sql")

	// Then
	bestSlotIndex := strings.Index(downSQL, "DROP TABLE chart_best_slot_stats_by_rating_band")
	chartIndex := strings.Index(downSQL, "DROP TABLE chart_stats_by_rating_band")
	worldsendIndex := strings.Index(downSQL, "DROP TABLE worldsend_chart_stats_by_rating_band")
	ratingBandIndex := strings.Index(downSQL, "DROP TABLE rating_bands")
	require.NotEqual(t, -1, bestSlotIndex)
	require.NotEqual(t, -1, chartIndex)
	require.NotEqual(t, -1, worldsendIndex)
	require.NotEqual(t, -1, ratingBandIndex)
	assert.Less(t, bestSlotIndex, ratingBandIndex)
	assert.Less(t, chartIndex, ratingBandIndex)
	assert.Less(t, worldsendIndex, ratingBandIndex)
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

func TestRemoveCreatedAtFromCourses_コースマスタの作成日時を削除する(t *testing.T) {
	// Given
	upSQL := readNormalizedMigrationSQL(t, "000031_remove_created_at_from_courses.up.sql")
	downSQL := readNormalizedMigrationSQL(t, "000031_remove_created_at_from_courses.down.sql")

	// Then
	assert.Contains(t, upSQL, "ALTER TABLE courses DROP COLUMN created_at")
	assert.Contains(t, downSQL, "ALTER TABLE courses ADD COLUMN created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP AFTER is_deleted")
}

func TestIdentifySPHonorsByImageURLUp_手動で変更した名称を優先して重複を統合する(t *testing.T) {
	// Given
	upSQL := readNormalizedMigrationSQL(t, "000032_identify_sp_honors_by_image_url.up.sql")

	// Then
	assert.Contains(t, upSQL, "SET image_url = TRIM(image_url) WHERE image_url IS NOT NULL")
	assert.Contains(t, upSQL, "PARTITION BY image_url")
	assert.Contains(t, upSQL, ") AS honor_rank")
	assert.Contains(t, upSQL, "WHERE ranked.honor_rank = 1")
	assert.NotContains(t, upSQL, "AS row_number")
	assert.Contains(t, upSQL, "CASE WHEN name = SUBSTRING_INDEX(SUBSTRING_INDEX(SUBSTRING_INDEX(image_url, '#', 1), '?', 1), '/', -1) THEN 1 ELSE 0 END")
	assert.Contains(t, upSQL, "UPDATE player_honors AS ph")
	assert.Contains(t, upSQL, "SET ph.honor_id = keepers.keep_id")
	assert.Contains(t, upSQL, "DELETE duplicated FROM honors AS duplicated")
	assert.Contains(t, upSQL, "ADD UNIQUE KEY unique_honor_image_url (image_url)")
	assert.Less(
		t,
		strings.Index(upSQL, "SET image_url = TRIM(image_url)"),
		strings.Index(upSQL, "PARTITION BY image_url"),
	)
	assert.Less(
		t,
		strings.Index(upSQL, "DELETE duplicated FROM honors AS duplicated"),
		strings.Index(upSQL, "ADD UNIQUE KEY unique_honor_image_url (image_url)"),
	)
}

func TestIdentifySPHonorsByImageURLDown_imageURLの一意制約を削除する(t *testing.T) {
	// Given
	downSQL := readNormalizedMigrationSQL(t, "000032_identify_sp_honors_by_image_url.down.sql")

	// Then
	assert.Contains(t, downSQL, "DROP INDEX unique_honor_image_url")
}

func TestAddDisplayIDToCoursesUp_既存コースへ一意な表示用IDを採番する(t *testing.T) {
	// Given
	upSQL := readNormalizedMigrationSQL(t, "000033_add_display_id_to_courses.up.sql")

	// Then
	assert.Contains(t, upSQL, "ADD COLUMN display_id CHAR(16) NULL AFTER id")
	assert.Contains(t, upSQL, "LOWER(HEX(RANDOM_BYTES(8)))")
	assert.Contains(t, upSQL, "MODIFY COLUMN display_id CHAR(16) NOT NULL")
	assert.Contains(t, upSQL, "ADD UNIQUE KEY uq_courses_display_id (display_id)")
	assert.Less(t, strings.Index(upSQL, "LOWER(HEX(RANDOM_BYTES(8)))"), strings.Index(upSQL, "MODIFY COLUMN display_id"))
}

func TestAddDisplayIDToCoursesDown_表示用IDを削除する(t *testing.T) {
	// Given
	downSQL := readNormalizedMigrationSQL(t, "000033_add_display_id_to_courses.down.sql")

	// Then
	assert.Contains(t, downSQL, "DROP INDEX uq_courses_display_id")
	assert.Contains(t, downSQL, "DROP COLUMN display_id")
}

func TestCreateRecordFiltersUp_保存済みフィルタテーブルを作成する(t *testing.T) {
	// Given
	upSQL := readNormalizedMigrationSQL(t, "000034_create_record_filters.up.sql")

	// Then
	assert.Contains(t, upSQL, "CREATE TABLE record_filters")
	assert.Contains(t, upSQL, "id BINARY(16) NOT NULL")
	assert.Contains(t, upSQL, "user_id INT UNSIGNED NOT NULL")
	assert.Contains(t, upSQL, "name VARCHAR(30) NOT NULL")
	assert.Contains(t, upSQL, "filter_value_gzip BLOB NOT NULL")
	assert.Contains(t, upSQL, "is_worldsend BOOLEAN NOT NULL DEFAULT FALSE")
	assert.Contains(t, upSQL, "created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP")
	assert.Contains(t, upSQL, "updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP")
	assert.Contains(t, upSQL, "PRIMARY KEY (id)")
	assert.Contains(t, upSQL, "FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE")
	assert.Contains(t, upSQL, "INDEX idx_record_filters_user_updated_id (user_id, updated_at DESC, id ASC)")
	assert.NotContains(t, upSQL, "UNIQUE KEY")
}

func TestCreateRecordFiltersDown_保存済みフィルタテーブルを削除する(t *testing.T) {
	// Given
	downSQL := readNormalizedMigrationSQL(t, "000034_create_record_filters.down.sql")

	// Then
	assert.Contains(t, downSQL, "DROP TABLE IF EXISTS record_filters")
}

func TestSchemaMySQL_保存済みフィルタテーブルを含む(t *testing.T) {
	// Given
	schemaSQL := readNormalizedMigrationSQL(t, "../schema_mysql.sql")

	// Then
	assert.Contains(t, schemaSQL, "CREATE TABLE `record_filters`")
	assert.Contains(t, schemaSQL, "`id` binary(16) NOT NULL")
	assert.Contains(t, schemaSQL, "KEY `idx_record_filters_user_updated_id` (`user_id`,`updated_at` DESC,`id`)")
	assert.Contains(t, schemaSQL, "CONSTRAINT `fk_record_filters_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE")
}

func TestCreatePlayerLatestUpdatesUp_プレイヤーごとの最新登録結果テーブルを作成する(t *testing.T) {
	// Given
	upSQL := readNormalizedMigrationSQL(t, "000035_create_player_latest_updates.up.sql")

	// Then
	assert.Contains(t, upSQL, "CREATE TABLE player_latest_updates")
	assert.Contains(t, upSQL, "player_id MEDIUMINT UNSIGNED NOT NULL")
	assert.Contains(t, upSQL, "schema_version INT UNSIGNED NOT NULL")
	assert.Contains(t, upSQL, "result_gzip MEDIUMBLOB NOT NULL")
	assert.Contains(t, upSQL, "source_updated_at DATETIME(6) NOT NULL")
	assert.Contains(t, upSQL, "imported_at DATETIME(6) NOT NULL")
	assert.Contains(t, upSQL, "body_hash CHAR(64) NOT NULL")
	assert.Contains(t, upSQL, "PRIMARY KEY (player_id)")
	assert.Contains(t, upSQL, "FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE")
}

func TestCreatePlayerLatestUpdatesDown_最新登録結果テーブルを削除する(t *testing.T) {
	// Given
	downSQL := readNormalizedMigrationSQL(t, "000035_create_player_latest_updates.down.sql")

	// Then
	assert.Contains(t, downSQL, "DROP TABLE IF EXISTS player_latest_updates")
}

func TestSchemaMySQL_プレイヤー最新登録結果テーブルを含む(t *testing.T) {
	// Given
	schemaSQL := readNormalizedMigrationSQL(t, "../schema_mysql.sql")

	// Then
	assert.Contains(t, schemaSQL, "CREATE TABLE `player_latest_updates`")
	assert.Contains(t, schemaSQL, "`result_gzip` mediumblob NOT NULL")
	assert.Contains(t, schemaSQL, "PRIMARY KEY (`player_id`)")
	assert.Contains(t, schemaSQL, "CONSTRAINT `fk_player_latest_updates_player` FOREIGN KEY (`player_id`) REFERENCES `players` (`id`) ON DELETE CASCADE")
}
