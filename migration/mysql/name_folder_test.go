package mysql

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNameFoldersMigration_選択肢と表示順(t *testing.T) {
	sql := readNormalizedMigrationSQL(t, "000058_create_name_folders.up.sql")
	rows := regexp.MustCompile(`\(\d+, '([^']+)', '([^']+)', (\d+)\)`).FindAllStringSubmatch(sql, -1)
	got := make([][]string, 0, len(rows))
	for _, row := range rows {
		got = append(got, row[1:])
	}
	assert.Equal(t, [][]string{
		{"ABCD", "ABCD", "1"}, {"EFGH", "EFGH", "2"}, {"IJKL", "IJKL", "3"},
		{"MNOP", "MNOP", "4"}, {"QRST", "QRST", "5"}, {"UVWXYZ", "UVWXYZ", "6"},
		{"A", "あ行", "7"}, {"KA", "か行", "8"}, {"SA", "さ行", "9"},
		{"TA", "た行", "10"}, {"NA", "な行", "11"}, {"HA", "は行", "12"},
		{"MA", "ま行", "13"}, {"YA", "や行", "14"}, {"RA", "ら行", "15"},
		{"WA", "わ行", "16"}, {"NUMBER", "数字", "17"},
	}, got)
	assert.Contains(t, sql, "UNIQUE KEY uq_name_folders_code (code)")
	assert.Contains(t, sql, "UNIQUE KEY uq_name_folders_name (name)")
	assert.Contains(t, sql, "UNIQUE KEY uq_name_folders_sort_order (sort_order)")
	assert.Equal(t, "DROP TABLE name_folders;", readNormalizedMigrationSQL(t, "000058_create_name_folders.down.sql"))
}

// parseNameFolderSortOrders は CASE 式から code と sort_order の組を記述順に取り出す。
func parseNameFolderSortOrders(sql string) [][]string {
	rows := regexp.MustCompile(`WHEN '([^']+)' THEN (\d+)`).FindAllStringSubmatch(sql, -1)
	got := make([][]string, 0, len(rows))
	for _, row := range rows {
		got = append(got, row[1:])
	}
	return got
}

func TestReorderNameFoldersMigration_筐体と同じカナ英字数字の順に並べ替える(t *testing.T) {
	// Given
	upSQL := readNormalizedMigrationSQL(t, "000060_reorder_name_folders.up.sql")
	downSQL := readNormalizedMigrationSQL(t, "000060_reorder_name_folders.down.sql")

	// Then: 一意制約に衝突しないよう、確定値の前に既存値を退避する
	for _, sql := range []string{upSQL, downSQL} {
		assert.Contains(t, sql, "UPDATE name_folders SET sort_order = sort_order + 100;")
	}
	assert.Equal(t, [][]string{
		{"A", "1"}, {"KA", "2"}, {"SA", "3"}, {"TA", "4"}, {"NA", "5"},
		{"HA", "6"}, {"MA", "7"}, {"YA", "8"}, {"RA", "9"}, {"WA", "10"},
		{"ABCD", "11"}, {"EFGH", "12"}, {"IJKL", "13"},
		{"MNOP", "14"}, {"QRST", "15"}, {"UVWXYZ", "16"}, {"NUMBER", "17"},
	}, parseNameFolderSortOrders(upSQL))
	assert.Equal(t, [][]string{
		{"ABCD", "1"}, {"EFGH", "2"}, {"IJKL", "3"},
		{"MNOP", "4"}, {"QRST", "5"}, {"UVWXYZ", "6"},
		{"A", "7"}, {"KA", "8"}, {"SA", "9"}, {"TA", "10"}, {"NA", "11"},
		{"HA", "12"}, {"MA", "13"}, {"YA", "14"}, {"RA", "15"}, {"WA", "16"},
		{"NUMBER", "17"},
	}, parseNameFolderSortOrders(downSQL))
}
