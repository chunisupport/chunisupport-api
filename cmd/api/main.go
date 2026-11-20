package main

import (
	"context"
	"errors"
	"fmt"
	"io"
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
	"github.com/jmoiron/sqlx"
)

func main() {
	os.Exit(run())
}

func run() (exitCode int) {
	slog.Info(info.Name, "build_date", info.BuildDate, "revision", info.Revision)

	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		return 1
	}

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

	signalCtx, cancelSignalCtx := context.WithCancel(context.Background())
	defer cancelSignalCtx()
	terminationCh := make(chan os.Signal, 1)
	receivedTerminationCh := make(chan os.Signal, 1)
	signal.Notify(terminationCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(terminationCh)
	go func() {
		select {
		case sig := <-terminationCh:
			receivedTerminationCh <- sig
			cancelSignalCtx()
		case <-signalCtx.Done():
		}
	}()
	reloadCh := make(chan os.Signal, 1)
	app.NotifyLogReload(reloadCh)
	defer signal.Stop(reloadCh)

	database, err := db.ConnectWithRetry(signalCtx, cfg.Database.DbConfig)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			slog.Info("Startup canceled", "signal", pendingSignalName(receivedTerminationCh))
			return 0
		}
		slog.Error("Failed to connect to database", "error", err)
		return 1
	}
	// サーバーの起動準備が失敗した場合は、run がDB接続を閉じます。
	serverOwnsDatabase := false
	defer func() {
		if !serverOwnsDatabase {
			if err := database.Close(); err != nil {
				slog.Error("Failed to close database after startup failure or cancellation", "error", err)
				exitCode = 1
			}
		}
	}()
	slog.Info("Connected to the database")

	startupCtx, cancelStartup := context.WithTimeout(signalCtx, info.APIStartupInitializationTimeout)
	defer cancelStartup()
	server, err := initializeServer(startupCtx, signalCtx, database, cfg, accessLogWriter)
	if err != nil {
		if signalCtx.Err() != nil {
			slog.Info("Startup canceled", "signal", pendingSignalName(receivedTerminationCh))
			return 0
		}
		slog.Error("Failed to initialize server", "error", err)
		return 1
	}
	serverOwnsDatabase = true

	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- server.Start(startupCtx)
	}()

	for {
		select {
		case sig := <-receivedTerminationCh:
			slog.Info("Starting graceful shutdown", "signal", signalName(sig))
			if err := shutdownServer(server, cfg.ShutdownTimeoutSeconds); err != nil {
				slog.Error("Server shutdown failed", "error", err)
				return 1
			}
			slog.Info("Graceful shutdown completed", "signal", signalName(sig))
			return 0
		case <-reloadCh:
			if err := logManager.ReopenAll(); err != nil {
				slog.Error("Failed to reopen logs", "error", err)
			} else {
				slog.Info("Logs reopened")
			}
		case err := <-serverErrCh:
			if errors.Is(err, context.Canceled) && signalCtx.Err() != nil {
				slog.Info("Startup canceled", "signal", pendingSignalName(receivedTerminationCh))
				if err := shutdownServer(server, cfg.ShutdownTimeoutSeconds); err != nil {
					slog.Error("Server shutdown after startup cancellation failed", "error", err)
					return 1
				}
				return 0
			}
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

// initializeServer は待受開始前の初期化全体に共通の期限を適用します。
func initializeServer(startupCtx, signalCtx context.Context, database *sqlx.DB, cfg config.Config, accessLogWriter io.Writer) (*app.Server, error) {
	masterCache, err := masterdata.Preload(startupCtx, database)
	if err != nil {
		return nil, fmt.Errorf("failed to preload master data: %w", err)
	}
	slog.Info("Master data preloaded")

	staticMasterCache, err := masterdata.PreloadStatic(startupCtx, database)
	if err != nil {
		return nil, fmt.Errorf("failed to preload static master data: %w", err)
	}
	slog.Info("Static master data preloaded")

	firebaseTokenVerifier, firebaseUserDeleter, firebaseProviderFinder, err := app.SetupFirebaseAuthServices(startupCtx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize firebase services: %w", err)
	}

	return app.NewServer(startupCtx, signalCtx, database, cfg, masterCache, staticMasterCache, firebaseTokenVerifier, firebaseUserDeleter, firebaseProviderFinder, accessLogWriter)
}

func shutdownServer(server *app.Server, shutdownTimeoutSeconds int) error {
	shutdownTimeout := time.Duration(shutdownTimeoutSeconds) * time.Second
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	return server.Shutdown(shutdownCtx)
}

func pendingSignalName(ch <-chan os.Signal) string {
	select {
	case sig := <-ch:
		return signalName(sig)
	default:
		return ""
	}
}

func signalName(sig os.Signal) string {
	if sig == nil {
		return ""
	}
	if sig == os.Interrupt {
		return "SIGINT"
	}
	if sig == syscall.SIGTERM {
		return "SIGTERM"
	}
	return sig.String()
}
