package app

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/config"
	"github.com/chunisupport/chunisupport-api/internal/infra/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v5"
)

// Server はアプリケーションサーバーを表します
type Server struct {
	echo              *echo.Echo
	startCtx          context.Context
	cancelStart       context.CancelFunc
	startDone         chan struct{}
	db                *sqlx.DB
	staticDB          *sqlx.DB
	smallDataDB       *sqlx.DB
	cfg               config.Config
	masterCache       *masterdata.Cache
	staticMasterCache *masterdata.StaticCache
}

// NewServer は新しいServerインスタンスを作成します
func NewServer(db *sqlx.DB, staticDB *sqlx.DB, smallDataDB *sqlx.DB, cfg config.Config, masterCache *masterdata.Cache, staticMasterCache *masterdata.StaticCache, firebaseTokenVerifier usecase.TokenVerifier, firebaseUserDeleter usecase.FirebaseUserDeleter, echoLogWriter io.Writer) *Server {
	startCtx, cancelStart := context.WithCancel(context.Background())
	return &Server{
		echo:              NewRouter(db, staticDB, smallDataDB, cfg, masterCache, staticMasterCache, firebaseTokenVerifier, firebaseUserDeleter, echoLogWriter),
		startCtx:          startCtx,
		cancelStart:       cancelStart,
		startDone:         make(chan struct{}),
		db:                db,
		staticDB:          staticDB,
		smallDataDB:       smallDataDB,
		cfg:               cfg,
		masterCache:       masterCache,
		staticMasterCache: staticMasterCache,
	}
}

// Start はサーバーを開始します
func (s *Server) Start() error {
	port := ":" + strconv.Itoa(s.cfg.AppPort)
	slog.Info("Starting server on port " + port)

	defer close(s.startDone)
	startConfig := echo.StartConfig{
		Address:         port,
		GracefulTimeout: time.Duration(s.cfg.ShutdownTimeoutSeconds) * time.Second,
	}
	if err := startConfig.Start(s.startCtx, s.echo); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("Failed to start server", "error", err)
		return err
	}

	return nil
}

// Shutdown はサーバーを正常に終了します
func (s *Server) Shutdown(ctx context.Context) error {
	var shutdownErrs []error

	if s.echo != nil {
		s.cancelStart()
		select {
		case <-s.startDone:
		case <-ctx.Done():
			slog.Error("Failed to shutdown echo server", "error", ctx.Err())
			shutdownErrs = append(shutdownErrs, ctx.Err())
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

	if s.smallDataDB != nil {
		if err := s.smallDataDB.Close(); err != nil {
			slog.Error("Failed to close small data database connection", "error", err)
			shutdownErrs = append(shutdownErrs, err)
		}
	}

	return errors.Join(shutdownErrs...)
}
