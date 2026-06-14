package main

import (
	"context"
	"errors"
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
	slog.Info(info.Name, "build_date", info.BuildDate, "revision", info.Revision)

	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		return 1
	}

	// アプリのロガーを設定
	loggerHandler, err := logger.NewHandler(cfg.Logging)
	if err != nil {
		slog.Error("Failed to create app logger", "error", err)
		return 1
	}
	accessLogWriter, err := logger.NewAccessLogWriter(cfg.Logging)
	if err != nil {
		slog.Error("Failed to create access logger", "error", err)
		if closeErr := loggerHandler.Close(); closeErr != nil {
			slog.Error("Failed to close app logger", "error", closeErr)
		}
		return 1
	}
	logManager := &app.LogManager{
		AppHandler:   loggerHandler,
		AccessWriter: accessLogWriter,
	}
	slog.SetDefault(slog.New(loggerHandler))
	defer func() {
		if err := logManager.Close(); err != nil {
			slog.Error("Failed to close log manager", "error", err)
		}
	}()

	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	reloadCh := make(chan os.Signal, 1)
	app.NotifyLogReload(reloadCh)
	defer signal.Stop(reloadCh)

	database, err := db.ConnectWithRetry(signalCtx, cfg.Database.DbConfig)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			slog.Info("Startup canceled")
			return 0
		}
		slog.Error("Failed to connect to database", "error", err)
		return 1
	}
	if err := signalCtx.Err(); err != nil {
		if closeErr := database.Close(); closeErr != nil {
			slog.Error("Failed to close database after startup cancellation", "error", closeErr)
			return 1
		}
		slog.Info("Startup canceled")
		return 0
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
	if err := signalCtx.Err(); err != nil {
		if closeErr := database.Close(); closeErr != nil {
			slog.Error("Failed to close database after startup cancellation", "error", closeErr)
			return 1
		}
		if closeErr := staticDatabase.Close(); closeErr != nil {
			slog.Error("Failed to close static database after startup cancellation", "error", closeErr)
			return 1
		}
		slog.Info("Startup canceled")
		return 0
	}

	server := app.NewServer(database, staticDatabase, cfg, masterCache, staticMasterCache, firebaseTokenVerifier, firebaseUserDeleter, accessLogWriter)

	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- server.Start()
	}()

	for {
		select {
		case <-signalCtx.Done():
			slog.Info("Starting graceful shutdown")
			if err := shutdownServer(server, cfg.ShutdownTimeoutSeconds); err != nil {
				slog.Error("Server shutdown failed", "error", err)
				return 1
			}
			slog.Info("Graceful shutdown completed")
			return 0
		case <-reloadCh:
			if err := logManager.ReopenAll(); err != nil {
				slog.Error("Failed to reopen logs", "error", err)
			} else {
				slog.Info("Logs reopened")
			}
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
	}
}

func shutdownServer(server *app.Server, shutdownTimeoutSeconds int) error {
	shutdownTimeout := time.Duration(shutdownTimeoutSeconds) * time.Second
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	return server.Shutdown(shutdownCtx)
}
