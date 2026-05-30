package db

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

// ConnectStatic は静的データ用SQLiteデータベースに接続します。
func ConnectStatic(dbPath string) (*sqlx.DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0750); err != nil {
		return nil, fmt.Errorf("failed to create static database directory: %w", err)
	}

	dsn := fmt.Sprintf("file:%s?_foreign_keys=on&_busy_timeout=5000", dbPath)
	db, err := sqlx.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open static database: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			slog.Error("Failed to close static database connection after ping failure", "error", closeErr)
		}
		return nil, fmt.Errorf("failed to ping static database: %w", err)
	}

	return db, nil
}
