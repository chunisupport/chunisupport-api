package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/app/handler/compat/chunirec"
	"github.com/chunisupport/chunisupport-api/internal/app/handler/compat/reiwa"
	"github.com/chunisupport/chunisupport-api/internal/app/staticdataexport"
	"github.com/chunisupport/chunisupport-api/internal/config"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/infra/cloudflare"
	"github.com/chunisupport/chunisupport-api/internal/infra/db"
	"github.com/chunisupport/chunisupport-api/internal/infra/logger"
	"github.com/chunisupport/chunisupport-api/internal/infra/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/infra/objectstorage"
	infrarepo "github.com/chunisupport/chunisupport-api/internal/infra/repository"
	"github.com/chunisupport/chunisupport-api/internal/infra/transaction"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
)

func main() {
	os.Exit(run())
}

func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.LoadBatchConfig()
	if err != nil {
		slog.Error("設定の読み込みに失敗しました", "error", err)
		return 1
	}
	logHandler, err := logger.NewHandler(cfg.Logging)
	if err != nil {
		slog.Error("ロガーの初期化に失敗しました", "error", err)
		return 1
	}
	slog.SetDefault(slog.New(logHandler))
	defer logHandler.Close()

	objectStorageConfig, err := config.LoadObjectStorageConfigFromEnv()
	if err != nil {
		slog.Error("オブジェクトストレージ設定の読み込みに失敗しました", "error", err)
		return 1
	}
	objectStorageWriter, err := objectstorage.NewWriter(objectStorageConfig)
	if err != nil {
		slog.Error("オブジェクトストレージクライアントの初期化に失敗しました", "error", err)
		return 1
	}
	cloudflareCacheConfig, err := config.LoadCloudflareCacheConfigFromEnv()
	if err != nil {
		slog.Error("Cloudflareキャッシュ設定の読み込みに失敗しました", "error", err)
		return 1
	}
	cachePurger := cloudflare.NewCachePurger(cloudflareCacheConfig)

	database, err := db.ConnectWithRetry(ctx, cfg.Database.DbConfig)
	if err != nil {
		slog.Error("DB接続に失敗しました", "error", err)
		return 1
	}
	defer database.Close()

	lock, acquired, err := db.NewAdvisoryLockProvider(database).TryAcquire(ctx, info.StaticDataExportBatchLockName)
	if err != nil {
		slog.Error("バッチロックの取得に失敗しました", "error", err)
		return 1
	}
	if !acquired {
		slog.Error("別の静的データエクスポートが実行中のため開始できません")
		return 1
	}
	defer func() {
		releaseCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := lock.Release(releaseCtx); err != nil {
			slog.Error("バッチロックの解放に失敗しました", "error", err)
		}
	}()

	masterCache, err := masterdata.Preload(ctx, database)
	if err != nil {
		slog.Error("マスターデータの読み込みに失敗しました", "error", err)
		return 1
	}

	transactionManager := transaction.NewTransactionManager(database)
	songUsecase := usecase.NewSongUsecase(
		infrarepo.NewSongRepository(database),
		masterCache,
		transactionManager,
		database,
	)
	worldsendUsecase := usecase.NewWorldsendUsecase(
		infrarepo.NewWorldsendChartRepository(database),
		transactionManager,
		database,
	)
	exporter := staticdataexport.NewExporter(
		songUsecase,
		worldsendUsecase,
		masterCache.GenreNamesByID,
		masterCache.DifficultyNamesByID,
		func(songs []*entity.Song) (any, int) {
			response := chunirec.ToMusicShowAllResponse(songs, masterCache.SongMasters())
			return response, len(response)
		},
		func(songs []*entity.Song) (any, int) {
			response := reiwa.ToChunithmRecordOriginalResponse(songs, masterCache)
			return response, len(response)
		},
		objectStorageWriter,
		cachePurger,
	)

	startedAt := time.Now()
	result, err := exporter.Export(ctx)
	if err != nil {
		slog.Error("静的データのエクスポートに失敗しました", "error", err, "duration", time.Since(startedAt))
		return 1
	}
	slog.Info(
		"静的データのエクスポートが完了しました",
		"songs", result.SongCount,
		"worldsend_songs", result.WorldsendSongCount,
		"chunirec_songs", result.ChunirecSongCount,
		"reiwa_records", result.ReiwaRecordCount,
		"duration", time.Since(startedAt),
	)
	return 0
}
