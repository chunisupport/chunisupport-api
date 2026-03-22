package db

import (
	"fmt"
	"log/slog"
	"slices"

	"github.com/jmoiron/sqlx"
)

var allowedValidationTables = []string{"songs", "charts", "genres", "song_difficulties", "users"}

// ValidateRequiredData はアプリケーションが正常に動作するために必要なデータが
// データベースに存在するかをチェックします。
// 必須テーブル（songs、charts）にデータが存在しない場合はエラーを返します。
func ValidateRequiredData(db *sqlx.DB) error {
	slog.Info("Starting database validation for required data")

	// songsテーブルのデータ存在チェック
	if err := checkTableHasData(db, "songs"); err != nil {
		return fmt.Errorf("songs table validation failed: %w", err)
	}

	// chartsテーブルのデータ存在チェック
	if err := checkTableHasData(db, "charts"); err != nil {
		return fmt.Errorf("charts table validation failed: %w", err)
	}

	slog.Info("Database validation completed successfully - all required data exists")
	return nil
}

// checkTableHasData は指定されたテーブルにデータが存在するかをチェックします。
func checkTableHasData(db *sqlx.DB, tableName string) error {
	if !slices.Contains(allowedValidationTables, tableName) {
		return fmt.Errorf("table %s is not allowed for validation", tableName)
	}

	var count int
	query := "SELECT COUNT(*) FROM " + tableName

	slog.Debug("Checking data existence", "table", tableName)

	if err := db.Get(&count, query); err != nil {
		return fmt.Errorf("failed to count records in table %s: %w", tableName, err)
	}

	if count == 0 {
		return fmt.Errorf("table %s has no data - application requires %s data to function properly", tableName, tableName)
	}

	slog.Info("Table validation passed", "table", tableName, "record_count", count)
	return nil
}

// GetTableStats は各テーブルのレコード数を取得して統計情報を返します（デバッグ用）。
func GetTableStats(db *sqlx.DB) (map[string]int, error) {
	tables := []string{"songs", "charts", "genres", "song_difficulties", "users"}
	stats := make(map[string]int)

	for _, table := range tables {
		if !slices.Contains(allowedValidationTables, table) {
			stats[table] = -1
			continue
		}

		var count int
		query := "SELECT COUNT(*) FROM " + table
		if err := db.Get(&count, query); err != nil {
			slog.Warn("Failed to get count for table", "table", table, "error", err)
			stats[table] = -1 // エラーを示すために -1 を設定
			continue
		}
		stats[table] = count
	}

	return stats, nil
}
