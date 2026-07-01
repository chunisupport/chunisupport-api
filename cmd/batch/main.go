package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/chunisupport/chunisupport-api/internal/config"
	"github.com/chunisupport/chunisupport-api/internal/infra/db"
	"github.com/chunisupport/chunisupport-api/internal/infra/logger"
	"github.com/chunisupport/chunisupport-api/internal/infra/masterdata"
)

func main() {
	os.Exit(run())
}

func run() int {
	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		return 1
	}

	loggerHandler, err := logger.NewHandler(cfg.Logging)
	if err != nil {
		slog.Error("Failed to create logger", "error", err)
		return 1
	}
	slog.SetDefault(slog.New(loggerHandler))
	defer func() {
		if err := loggerHandler.Close(); err != nil {
			slog.Error("Failed to close logger", "error", err)
		}
	}()

	database, err := db.ConnectWithRetry(context.Background(), cfg.Database.DbConfig)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		return 1
	}
	defer func() {
		if err := database.Close(); err != nil {
			slog.Error("Failed to close database", "error", err)
		}
	}()

	staticDatabase, err := db.ConnectStatic(cfg.StaticDBPath)
	if err != nil {
		slog.Error("Failed to connect to static database", "error", err)
		return 1
	}
	defer func() {
		if err := staticDatabase.Close(); err != nil {
			slog.Error("Failed to close static database", "error", err)
		}
	}()

	_, err = masterdata.Preload(context.Background(), database)
	if err != nil {
		slog.Error("Failed to preload master data", "error", err)
		return 1
	}

	// TODO: バッチジョブの実装
	// 例: usecase.NewSomeUsecase(...).Execute(ctx, params)
	slog.Info("Batch job completed successfully")
	return 0
}
