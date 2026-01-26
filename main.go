package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Qman110101/chunisupport-api/internal/app"
	"github.com/Qman110101/chunisupport-api/internal/config"
	"github.com/Qman110101/chunisupport-api/internal/info"
	"github.com/Qman110101/chunisupport-api/internal/infra/db"
	"github.com/Qman110101/chunisupport-api/internal/infra/logger"
	"github.com/Qman110101/chunisupport-api/internal/infra/masterdata"
)

func main() {
	slog.Info(info.Name + " v" + info.Version)

	env := os.Getenv("APP_ENV")
	if env != "" {
		slog.Debug("Environment: " + env)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		return
	}

	// アプリのロガーを設定
	logLevel := logger.ParseLogLevel(cfg.LogLevel)
	var appLogger *slog.Logger
	var loggerHandler *logger.Handler
	if appLoggerWithFile, err := logger.NewHandlerWithFile(cfg.LogPaths.App, logLevel); err == nil {
		loggerHandler = appLoggerWithFile
		appLogger = slog.New(loggerHandler)
	} else {
		slog.Error("Failed to create app logger with file", "error", err)
		loggerHandler = logger.NewHandler(logLevel)
		appLogger = slog.New(loggerHandler)
	}
	slog.SetDefault(appLogger)
	defer func() {
		if err := loggerHandler.Close(); err != nil {
			slog.Error("Failed to close logger", "error", err)
		}
	}()

	database, err := db.Connect(cfg.Database.DbConfig)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		return
	}

	if err := database.Ping(); err != nil {
		slog.Error("Failed to ping database", "error", err)
		return
	}

	slog.Info("Connected to the database")

	// 必須データの存在チェック
	// if err := db.ValidateRequiredData(database); err != nil {
	// 	slog.Error("Required data validation failed", "error", err)
	// 	slog.Error("Application cannot start without required data in songs and charts tables")
	// 	slog.Info("Please ensure the database has been properly migrated and populated with song/chart data")
	// 	return
	// }

	ctx := context.Background()
	masterCache, err := masterdata.Preload(ctx, database)
	if err != nil {
		slog.Error("Failed to preload master data", "error", err)
		return
	}

	slog.Info("Master data preloaded")

	// サーバーの作成と起動
	server := app.NewServer(database, cfg, masterCache)
	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := server.Start(); err != nil {
			stop()
		}
	}()

	<-signalCtx.Done()
	slog.Info("Starting graceful shutdown")

	shutdownTimeout := time.Duration(cfg.ShutdownTimeoutSeconds) * time.Second
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("Server shutdown failed", "error", err)
		return
	}

	slog.Info("Graceful shutdown completed")
}
