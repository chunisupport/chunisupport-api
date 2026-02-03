package app

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/Qman110101/chunisupport-api/internal/config"
	"github.com/Qman110101/chunisupport-api/internal/infra/masterdata"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
)

// Server はアプリケーションサーバーを表します
type Server struct {
	echo           *echo.Echo
	db             *sqlx.DB
	cfg            config.Config
	masterCache    *masterdata.Cache
	echoLogWriter  io.WriteCloser
	shutdownCancel context.CancelFunc
}

// NewServer は新しいServerインスタンスを作成します
func NewServer(db *sqlx.DB, cfg config.Config, masterCache *masterdata.Cache) *Server {
	// Echoのロガーを設定
	var echoLogWriter io.WriteCloser
	echoLogWriterResult, err := SetupEchoLogger(cfg)
	if err != nil {
		slog.Error("Failed to setup echo logger", "error", err)
	} else {
		echoLogWriter = echoLogWriterResult
	}

	serverCtx, cancel := context.WithCancel(context.Background())

	return &Server{
		echo:           NewRouter(serverCtx, db, cfg, masterCache, echoLogWriter),
		db:             db,
		cfg:            cfg,
		masterCache:    masterCache,
		echoLogWriter:  echoLogWriter,
		shutdownCancel: cancel,
	}
}

// Start はサーバーを開始します
func (s *Server) Start() error {
	port := ":" + strconv.Itoa(s.cfg.AppPort)
	slog.Info("Starting server on port " + port)

	if err := s.echo.Start(port); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("Failed to start server", "error", err)
		return err
	}

	return nil
}

// Shutdown はサーバーを正常に終了します
func (s *Server) Shutdown(ctx context.Context) error {
	var shutdownErrs []error

	if s.shutdownCancel != nil {
		s.shutdownCancel()
	}

	if s.echo != nil {
		if err := s.echo.Shutdown(ctx); err != nil {
			slog.Error("Failed to shutdown echo server", "error", err)
			shutdownErrs = append(shutdownErrs, err)
		}
	}

	// Echoログファイルのクローズ
	if s.echoLogWriter != nil {
		if err := s.echoLogWriter.Close(); err != nil {
			slog.Error("Failed to close echo log file", "error", err)
			shutdownErrs = append(shutdownErrs, err)
		}
	}

	if s.db != nil {
		if err := s.db.Close(); err != nil {
			slog.Error("Failed to close database connection", "error", err)
			shutdownErrs = append(shutdownErrs, err)
		}
	}

	return errors.Join(shutdownErrs...)
}
