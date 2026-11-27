package mysql

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAddDisplayIDToVersionsAndHonorsMigration は既存のバージョン・称号へ公開IDを採番してから一意制約を付けることを確認します。
func TestAddDisplayIDToVersionsAndHonorsMigration(t *testing.T) {
	// Given
	up := readNormalizedMigrationSQL(t, "000074_add_display_id_to_versions_and_honors.up.sql")
	down := readNormalizedMigrationSQL(t, "000074_add_display_id_to_versions_and_honors.down.sql")
	schema := readNormalizedMigrationSQL(t, "../schema_mysql.sql")

	// Then
	for _, table := range []string{"versions", "honors"} {
		assert.Contains(t, up, "ALTER TABLE "+table+" ADD COLUMN display_id CHAR(16) NULL AFTER id")
		assert.Contains(t, up, "ADD UNIQUE KEY uq_"+table+"_display_id (display_id)")
		assert.Contains(t, down, "DROP INDEX uq_"+table+"_display_id")
		assert.Contains(t, schema, "UNIQUE KEY `uq_"+table+"_display_id` (`display_id`)")
	}
	assert.Contains(t, up, "LOWER(HEX(RANDOM_BYTES(8)))")
}

// TestConvertGoalAttributeMasterIDsMigration は目標の diff / genre / ver を公開識別子へ移行し、down で数値IDへ戻せることを確認します。
func TestConvertGoalAttributeMasterIDsMigration(t *testing.T) {
	// Given: 000074 までを適用したDBに、数値IDで保存された目標がある
	db := newMigrationTestDatabase(t)
	applyMigrations(t, db, upMigrationsBefore(t, "000075")...)
	_, err := db.Exec(`INSERT INTO users (id, username) VALUES (1, 'goal_migration_user')`)
	require.NoError(t, err)

	var difficultyNames, genreNames, versionDisplayIDs []string
	require.NoError(t, db.Select(&difficultyNames, `SELECT name FROM difficulties ORDER BY sort_order, id`))
	require.NoError(t, db.Select(&genreNames, `SELECT name FROM genres WHERE id IN (1, 2) ORDER BY sort_order, id`))
	require.NoError(t, db.Select(&versionDisplayIDs, `SELECT display_id FROM versions WHERE id IN (1, 2) ORDER BY released_at, id`))
	var masterDifficultyID int
	require.NoError(t, db.Get(&masterDifficultyID, `SELECT id FROM difficulties WHERE name = 'MASTER'`))
	var firstGenreName string
	require.NoError(t, db.Get(&firstGenreName, `SELECT name FROM genres WHERE id = 1`))
	var allDifficultyIDs []int
	require.NoError(t, db.Select(&allDifficultyIDs, `SELECT id FROM difficulties ORDER BY id DESC`))
	allDifficultyIDsJSON, err := json.Marshal(allDifficultyIDs)
	require.NoError(t, err)

	tests := []struct {
		name       string
		attributes string
		wantUp     string
		wantDown   string
	}{
		{
			name:       "単一の難易度IDは難易度名になる",
			attributes: fmt.Sprintf(`{"diff": %d}`, masterDifficultyID),
			wantUp:     `{"diff": "MASTER"}`,
			wantDown:   fmt.Sprintf(`{"diff": %d}`, masterDifficultyID),
		},
		{
			name:       "複数の難易度IDは表示順の難易度名配列になる",
			attributes: fmt.Sprintf(`{"diff": %s}`, allDifficultyIDsJSON),
			wantUp:     mustJSON(t, map[string]any{"diff": difficultyNames}),
			wantDown:   `{"diff": [1, 2, 3, 4, 5]}`,
		},
		{
			name:       "存在しないIDを除いて1件だけ残ったジャンルは単一値になる",
			attributes: `{"genre": [1, 250]}`,
			wantUp:     mustJSON(t, map[string]any{"genre": firstGenreName}),
			wantDown:   `{"genre": 1}`,
		},
		{
			name:       "存在しないIDだけのバージョンは属性ごと削除される",
			attributes: `{"ver": 250, "name_folder": "A"}`,
			wantUp:     `{"name_folder": "A"}`,
			wantDown:   `{"name_folder": "A"}`,
		},
		{
			name:       "複数のバージョンIDは稼働日順の公開IDになり、他の属性は維持される",
			attributes: `{"ver": [2, 1], "genre": [2, 1], "const": {"min": 13.0, "max": 15.4}}`,
			wantUp:     mustJSON(t, map[string]any{"ver": versionDisplayIDs, "genre": genreNames, "const": map[string]float64{"min": 13.0, "max": 15.4}}),
			wantDown:   `{"ver": [1, 2], "genre": [1, 2], "const": {"min": 13.0, "max": 15.4}}`,
		},
		{
			name:       "絞り込み属性がない目標は変わらない",
			attributes: `{}`,
			wantUp:     `{}`,
			wantDown:   `{}`,
		},
	}
	for i, tt := range tests {
		_, err := db.Exec(`INSERT INTO goals (id, user_id, title, achievement_type_id, achievement_params, attributes, sort_order) VALUES (?, 1, ?, 1, '{}', ?, ?)`, i+1, fmt.Sprintf("goal%d", i+1), tt.attributes, i+1)
		require.NoError(t, err, tt.name)
	}

	// When: up を適用する
	applyMigrations(t, db, "000075_convert_goal_attribute_master_ids.up.sql")

	// Then
	for i, tt := range tests {
		assert.JSONEq(t, tt.wantUp, goalAttributes(t, db, i+1), tt.name)
	}

	// When: down を適用する
	applyMigrations(t, db, "000075_convert_goal_attribute_master_ids.down.sql")

	// Then
	for i, tt := range tests {
		assert.JSONEq(t, tt.wantDown, goalAttributes(t, db, i+1), tt.name)
	}
}

func goalAttributes(t *testing.T, db *sqlx.DB, id int) string {
	t.Helper()
	var attributes string
	require.NoError(t, db.Get(&attributes, `SELECT attributes FROM goals WHERE id = ?`, id))
	return attributes
}

func mustJSON(t *testing.T, value any) string {
	t.Helper()
	encoded, err := json.Marshal(value)
	require.NoError(t, err)
	return string(encoded)
}

// newMigrationTestDatabase はマイグレーション未適用の使い捨てDBを作成します。
// 特定のマイグレーションの前後でデータを検証するため、全upを適用する mysqltest は使いません。
func newMigrationTestDatabase(t *testing.T) *sqlx.DB {
	t.Helper()
	dsn := os.Getenv("TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("TEST_MYSQL_DSN が未設定です")
	}
	cfg, err := mysql.ParseDSN(dsn)
	require.NoError(t, err)
	cfg.DBName = ""
	cfg.ParseTime = true
	cfg.Loc = time.UTC
	cfg.MultiStatements = true
	admin, err := sqlx.Connect("mysql", cfg.FormatDSN())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, admin.Close()) })
	name := fmt.Sprintf("chunisupport_migration_test_%d", time.Now().UnixNano())
	_, err = admin.Exec("CREATE DATABASE " + name + " CHARACTER SET utf8mb4")
	require.NoError(t, err)
	t.Cleanup(func() { _, err := admin.Exec("DROP DATABASE " + name); require.NoError(t, err) })
	cfg.DBName = name
	db, err := sqlx.Connect("mysql", cfg.FormatDSN())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	return db
}

// applyMigrations は指定したマイグレーションファイルを順に適用します。
func applyMigrations(t *testing.T, db *sqlx.DB, files ...string) {
	t.Helper()
	for _, file := range files {
		data, err := os.ReadFile(file)
		require.NoError(t, err)
		_, err = db.Exec(string(data))
		require.NoError(t, err, file)
	}
}

// upMigrationsBefore は指定したファイル名より前の up マイグレーションをファイル名順に返します。
func upMigrationsBefore(t *testing.T, name string) []string {
	t.Helper()
	files, err := filepath.Glob("*.up.sql")
	require.NoError(t, err)
	before := make([]string, 0, len(files))
	for _, file := range files {
		if file < name {
			before = append(before, file)
		}
	}
	return before
}
