package db

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/config"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

// Connect はデータベースへの接続を確立し、*sqlx.DBを返します。
func Connect(dbConfig config.DbConfig) (*sqlx.DB, error) {
	// DSN (Data Source Name) を構築
	// clientFoundRows=true: UPDATE時に「変更された行数」ではなく「マッチした行数」を返すようにする。
	// これにより、値が変わらないUPDATEでもRowsAffected>=1となり、
	// Save/SaveSongのRowsAffected==0チェック（存在確認）が正しく動作する。
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&loc=Local&clientFoundRows=true",
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

	db.SetMaxOpenConns(dbConfig.MaxOpenConns)
	db.SetMaxIdleConns(dbConfig.MaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(dbConfig.ConnMaxLifetimeSec) * time.Second)
	db.SetConnMaxIdleTime(time.Duration(dbConfig.ConnMaxIdleTimeSec) * time.Second)

	if err := db.Ping(); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			slog.Error("Failed to close database connection after ping failure", "error", closeErr)
		}
		return nil, fmt.Errorf("Failed to ping MySQL database: %w", err)
	}

	return db, nil
}
