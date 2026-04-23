package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/app"
	"github.com/chunisupport/chunisupport-api/internal/config"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/infra/db"
	"github.com/chunisupport/chunisupport-api/internal/infra/logger"
	"github.com/chunisupport/chunisupport-api/internal/infra/masterdata"
)

func main() {
	os.Exit(run())
}

func run() int {
	slog.Info(info.Name + " v" + info.Version)

	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		return 1
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
		return 1
	}

	slog.Info("Connected to the database")

	staticDatabase, err := db.ConnectStatic(cfg.StaticDBPath)
	if err != nil {
		slog.Error("Failed to connect to static database", "error", err)
		return 1
	}

	slog.Info("Connected to the static database")

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
		return 1
	}

	slog.Info("Master data preloaded")

	staticMasterCache, err := masterdata.PreloadStatic(ctx, staticDatabase)
	if err != nil {
		slog.Error("Failed to preload static master data", "error", err)
		return 1
	}

	slog.Info("Static master data preloaded")

	firebaseTokenVerifier, firebaseUserDeleter, err := app.SetupFirebaseAuthServices(ctx, cfg)
	if err != nil {
		slog.Error("Failed to initialize firebase services", "error", err)
		return 1
	}

	// サーバーの作成と起動
	server := app.NewServer(database, staticDatabase, cfg, masterCache, staticMasterCache, firebaseTokenVerifier, firebaseUserDeleter)
	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- server.Start()
	}()

	select {
	case <-signalCtx.Done():
		slog.Info("Starting graceful shutdown")
	case err := <-serverErrCh:
		if err != nil {
			slog.Error("Server stopped with error", "error", err)
			if shutdownErr := shutdownServer(server, cfg.ShutdownTimeoutSeconds); shutdownErr != nil {
				slog.Error("Server shutdown after start failure failed", "error", shutdownErr)
			}
			return 1
		}
		slog.Info("Server stopped")
		if shutdownErr := shutdownServer(server, cfg.ShutdownTimeoutSeconds); shutdownErr != nil {
			slog.Error("Server shutdown after stop failed", "error", shutdownErr)
			return 1
		}
		return 0
	}

	if err := shutdownServer(server, cfg.ShutdownTimeoutSeconds); err != nil {
		slog.Error("Server shutdown failed", "error", err)
		return 1
	}

	slog.Info("Graceful shutdown completed")
	return 0
}

func shutdownServer(server *app.Server, shutdownTimeoutSeconds int) error {
	shutdownTimeout := time.Duration(shutdownTimeoutSeconds) * time.Second
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	return server.Shutdown(shutdownCtx)
}
