package db

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Qman110101/chunisupport-api/internal/info"
	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

// ConnectStatic は静的データ用SQLiteデータベースに接続します。
func ConnectStatic() (*sqlx.DB, error) {
	executablePath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}

	dir := filepath.Dir(executablePath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create static database directory: %w", err)
	}

	dbPath := filepath.Join(dir, info.StaticDBFilename)
	dsn := fmt.Sprintf("file:%s?_foreign_keys=on&_busy_timeout=5000", dbPath)
	db, err := sqlx.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open static database: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	return db, nil
}
