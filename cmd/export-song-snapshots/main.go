package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/app/songexport"
	"github.com/chunisupport/chunisupport-api/internal/config"
	"github.com/chunisupport/chunisupport-api/internal/info"
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

	r2Config, err := config.LoadR2ConfigFromEnv()
	if err != nil {
		slog.Error("R2設定の読み込みに失敗しました", "error", err)
		return 1
	}
	r2Writer, err := objectstorage.NewR2Writer(r2Config)
	if err != nil {
		slog.Error("R2クライアントの初期化に失敗しました", "error", err)
		return 1
	}

	database, err := db.ConnectWithRetry(ctx, cfg.Database.DbConfig)
	if err != nil {
		slog.Error("DB接続に失敗しました", "error", err)
		return 1
	}
	defer database.Close()

	lock, acquired, err := db.NewAdvisoryLockProvider(database).TryAcquire(ctx, info.SongSnapshotExportBatchLockName)
	if err != nil {
		slog.Error("バッチロックの取得に失敗しました", "error", err)
		return 1
	}
	if !acquired {
		slog.Error("別の楽曲スナップショットエクスポートが実行中のため開始できません")
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
	exporter := songexport.NewExporter(
		songUsecase,
		worldsendUsecase,
		masterCache.GenreNamesByID,
		masterCache.DifficultyNamesByID,
		r2Writer,
	)

	startedAt := time.Now()
	result, err := exporter.Export(ctx)
	if err != nil {
		slog.Error("楽曲スナップショットのエクスポートに失敗しました", "error", err, "duration", time.Since(startedAt))
		return 1
	}
	slog.Info(
		"楽曲スナップショットのエクスポートが完了しました",
		"songs", result.SongCount,
		"worldsend_songs", result.WorldsendSongCount,
		"duration", time.Since(startedAt),
	)
	return 0
}
