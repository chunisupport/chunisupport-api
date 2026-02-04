package db

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

// ConnectStatic は静的データ用SQLiteデータベースに接続します。
func ConnectStatic(path string) (*sqlx.DB, error) {
	if path == "" {
		return nil, fmt.Errorf("static database path is empty")
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create static database directory: %w", err)
	}

	dsn := fmt.Sprintf("file:%s?_foreign_keys=on&_busy_timeout=5000", path)
	db, err := sqlx.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open static database: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	return db, nil
}
