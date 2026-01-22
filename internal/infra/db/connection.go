package db

import (
	"fmt"
	"log/slog"

	"github.com/Qman110101/chunisupport-api/internal/config"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

// Connect はデータベースへの接続を確立し、*sqlx.DBを返します。
func Connect(dbConfig config.DbConfig) (*sqlx.DB, error) {
	// DSN (Data Source Name) を構築
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&loc=Local",
		dbConfig.DbUser,
		dbConfig.DbPass,
		dbConfig.DbHost,
		dbConfig.DbPort,
		dbConfig.DbName,
	)

	db, err := sqlx.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("Failed to open MySQL database: %w", err)
	}

	if err := db.Ping(); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			slog.Error("Failed to close database connection after ping failure", "error", closeErr)
		}
		return nil, fmt.Errorf("Failed to ping MySQL database: %w", err)
	}

	return db, nil
}
