package app

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
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
	cfg               config.Config
	masterCache       *masterdata.Cache
	staticMasterCache *masterdata.StaticCache
}

// NewServer は永続化済みの運用状態を読み込んでServerインスタンスを作成します。
func NewServer(ctx context.Context, db *sqlx.DB, cfg config.Config, masterCache *masterdata.Cache, staticMasterCache *masterdata.StaticCache, firebaseTokenVerifier usecase.TokenVerifier, firebaseUserDeleter usecase.FirebaseUserDeleter, echoLogWriter io.Writer) (*Server, error) {
	router, err := NewRouter(ctx, db, cfg, masterCache, staticMasterCache, firebaseTokenVerifier, firebaseUserDeleter, echoLogWriter)
	if err != nil {
		return nil, err
	}

	startCtx, cancelStart := context.WithCancel(context.Background())
	return &Server{
		echo:              router,
		startCtx:          startCtx,
		cancelStart:       cancelStart,
		startDone:         make(chan struct{}),
		db:                db,
		cfg:               cfg,
		masterCache:       masterCache,
		staticMasterCache: staticMasterCache,
	}, nil
}

// Start はサーバーを開始します
func (s *Server) Start() error {
	address := serverAddress(s.cfg.AppPort)
	slog.Info("Starting server", "address", address)

	defer close(s.startDone)
	startConfig := echo.StartConfig{
		Address:         address,
		GracefulTimeout: time.Duration(s.cfg.ShutdownTimeoutSeconds) * time.Second,
	}
	if err := startConfig.Start(s.startCtx, s.echo); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("Failed to start server", "error", err)
		return err
	}

	return nil
}

// serverAddress はNginxからのプロキシ接続だけを受け付けるループバック待受アドレスを返します。
func serverAddress(port int) string {
	return net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
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

	return errors.Join(shutdownErrs...)
}
