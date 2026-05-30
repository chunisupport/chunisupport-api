package db

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/config"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

// ConnectWithRetry は起動時にMySQLが利用可能になるまで接続を再試行します。
func ConnectWithRetry(ctx context.Context, dbConfig config.DbConfig) (*sqlx.DB, error) {
	return connectWithRetry(ctx, dbConfig, ConnectContext)
}

func connectWithRetry(ctx context.Context, dbConfig config.DbConfig, connect func(context.Context, config.DbConfig) (*sqlx.DB, error)) (*sqlx.DB, error) {
	maxWait := time.Duration(dbConfig.StartupMaxWaitSec) * time.Second
	interval := time.Duration(dbConfig.StartupIntervalSec) * time.Second
	deadline := time.Now().Add(maxWait)

	for attempt := 1; ; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("database startup wait canceled: %w", err)
		}

		attemptCtx := ctx
		cancel := func() {}
		if maxWait > 0 {
			remaining := time.Until(deadline)
			if remaining <= 0 {
				return nil, fmt.Errorf("database did not become ready within %s: %w", maxWait, context.DeadlineExceeded)
			}
			attemptCtx, cancel = context.WithTimeout(ctx, remaining)
		}

		db, err := connect(attemptCtx, dbConfig)
		cancel()
		if err == nil {
			return db, nil
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, fmt.Errorf("database startup wait canceled: %w", ctxErr)
		}

		remaining := time.Until(deadline)
		if maxWait == 0 || remaining <= 0 {
			return nil, fmt.Errorf("database did not become ready within %s: %w", maxWait, err)
		}

		wait := min(interval, remaining)
		slog.Warn("Database is not ready yet; waiting before retry", "attempt", attempt, "interval", wait, "error", err)

		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, fmt.Errorf("database startup wait canceled: %w", ctx.Err())
		case <-timer.C:
		}
	}
}

// Connect はデータベースへの接続を確立し、*sqlx.DBを返します。
func Connect(dbConfig config.DbConfig) (*sqlx.DB, error) {
	return ConnectContext(context.Background(), dbConfig)
}

// ConnectContext はデータベースへの接続を確立し、*sqlx.DBを返します。
func ConnectContext(ctx context.Context, dbConfig config.DbConfig) (*sqlx.DB, error) {
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

	if err := db.PingContext(ctx); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			slog.Error("Failed to close database connection after ping failure", "error", closeErr)
		}
		return nil, fmt.Errorf("Failed to ping MySQL database: %w", err)
	}

	return db, nil
}
