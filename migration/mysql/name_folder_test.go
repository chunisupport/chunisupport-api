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
