package app

import (
	"context"
	"io"

	"github.com/chunisupport/chunisupport-api/internal/config"
	"github.com/chunisupport/chunisupport-api/internal/infra/masterdata"
	infra "github.com/chunisupport/chunisupport-api/internal/infra/repository"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v5"
)

// NewRouter はルートが設定された新しいEchoインスタンスを作成します
// echoLogWriterがnilの場合は、テストなどの直接構築時にアクセスログミドルウェアを無効化します。
func NewRouter(ctx context.Context, db *sqlx.DB, cfg config.Config, masterCache *masterdata.Cache, staticMasterCache *masterdata.StaticCache, firebaseTokenVerifier usecase.TokenVerifier, firebaseUserDeleter usecase.FirebaseUserDeleter, firebaseProviderFinder usecase.FirebaseProviderAccountFinder, echoLogWriter io.Writer) (*echo.Echo, error) {
	e, _, _, _, err := newRouter(ctx, ctx, db, cfg, masterCache, staticMasterCache, firebaseTokenVerifier, firebaseUserDeleter, firebaseProviderFinder, echoLogWriter)
	return e, err
}

// backgroundBatchJobs は管理画面から起動したバッチや楽曲カタログ・上位%スナップショットの世代確認など、DBを使うバックグラウンド処理です。
// サーバー停止時は DB 接続を閉じる前に Wait で完了を待ちます。
type backgroundBatchJobs interface {
	Wait()
}

// newRouter はルーターを構築し、サーバー停止時に完了を待つ必要があるバッチのユースケースと、共有キャッシュProviderも返します。
// HTTP設定（newEcho）、依存構築（buildRouterDependencies）、ルート登録（registerSystemRoutes / registerRoutes）を
// この順で組み立てる composition root です。
// ctx は管理画面から起動したバッチの寿命にもなるため、サーバー停止時にキャンセルされるものを渡します。
// 世代確認はここでは開始しません。起動は NewServer が行います。
func newRouter(startupCtx, ctx context.Context, db *sqlx.DB, cfg config.Config, masterCache *masterdata.Cache, staticMasterCache *masterdata.StaticCache, firebaseTokenVerifier usecase.TokenVerifier, firebaseUserDeleter usecase.FirebaseUserDeleter, firebaseProviderFinder usecase.FirebaseProviderAccountFinder, echoLogWriter io.Writer) (*echo.Echo, []backgroundBatchJobs, *infra.SongCatalogProvider, *infra.ScorePercentileProvider, error) {
	e, err := newEcho(cfg, echoLogWriter)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	deps, err := buildRouterDependencies(startupCtx, ctx, db, cfg, masterCache, staticMasterCache, firebaseTokenVerifier, firebaseUserDeleter, firebaseProviderFinder)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	registerSystemRoutes(e, deps.firebaseAuthenticatorStrict, deps.apiTokenUsecase, deps.maintenanceStateProvider, cfg)
	registerRoutes(e, deps.handlers, deps.firebaseAuthenticatorStrict, deps.firebaseAuthenticatorReadOptimized, deps.apiTokenUsecase, deps.maintenanceStateProvider, cfg)

	return e, deps.batchJobs, deps.songCatalogProvider, deps.scorePercentileProvider, nil
}
