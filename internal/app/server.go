package app

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/chunisupport/chunisupport-api/internal/config"
	"github.com/chunisupport/chunisupport-api/internal/infra/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
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
}

// NewServer は新しいServerインスタンスを作成します
func NewServer(db *sqlx.DB, staticDB *sqlx.DB, cfg config.Config, masterCache *masterdata.Cache, staticMasterCache *masterdata.StaticCache, firebaseTokenVerifier usecase.TokenVerifier, firebaseUserDeleter usecase.FirebaseUserDeleter, echoLogWriter io.Writer) *Server {
	return &Server{
		echo:              NewRouter(db, staticDB, cfg, masterCache, staticMasterCache, firebaseTokenVerifier, firebaseUserDeleter, echoLogWriter),
		db:                db,
		staticDB:          staticDB,
		cfg:               cfg,
		masterCache:       masterCache,
		staticMasterCache: staticMasterCache,
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
