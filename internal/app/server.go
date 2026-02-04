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
	echo              *echo.Echo
	db                *sqlx.DB
	staticDB          *sqlx.DB
	cfg               config.Config
	masterCache       *masterdata.Cache
	staticMasterCache *masterdata.StaticCache
	echoLogWriter     io.WriteCloser
}

// NewServer は新しいServerインスタンスを作成します
func NewServer(db *sqlx.DB, staticDB *sqlx.DB, cfg config.Config, masterCache *masterdata.Cache, staticMasterCache *masterdata.StaticCache) *Server {
	// Echoのロガーを設定
	var echoLogWriter io.WriteCloser
	echoLogWriterResult, err := SetupEchoLogger(cfg)
	if err != nil {
		slog.Error("Failed to setup echo logger", "error", err)
	} else {
		echoLogWriter = echoLogWriterResult
	}

	return &Server{
		echo:              NewRouter(db, staticDB, cfg, masterCache, staticMasterCache, echoLogWriter),
		db:                db,
		staticDB:          staticDB,
		cfg:               cfg,
		masterCache:       masterCache,
		staticMasterCache: staticMasterCache,
		echoLogWriter:     echoLogWriter,
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

	if s.staticDB != nil {
		if err := s.staticDB.Close(); err != nil {
			slog.Error("Failed to close static database connection", "error", err)
			shutdownErrs = append(shutdownErrs, err)
		}
	}

	return errors.Join(shutdownErrs...)
}
