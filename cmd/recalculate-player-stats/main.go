package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/config"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/infra/db"
	"github.com/chunisupport/chunisupport-api/internal/infra/logger"
	infrarepo "github.com/chunisupport/chunisupport-api/internal/infra/repository"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
)

func main() {
	os.Exit(run())
}

func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("設定の読み込みに失敗しました", "error", err)
		return 1
	}
	handler, err := logger.NewHandler(cfg.Logging)
	if err != nil {
		slog.Error("ロガーの初期化に失敗しました", "error", err)
		return 1
	}
	slog.SetDefault(slog.New(handler))
	defer handler.Close()
	database, err := db.ConnectWithRetry(ctx, cfg.Database.DbConfig)
	if err != nil {
		slog.Error("DB接続に失敗しました", "error", err)
		return 1
	}
	defer database.Close()
	lock, acquired, err := db.NewAdvisoryLockProvider(database).TryAcquire(ctx, info.PlayerStatsBatchLockName)
	if err != nil {
		slog.Error("バッチロックの取得に失敗しました", "error", err)
		return 1
	}
	if !acquired {
		slog.Info("別の再計算バッチが実行中のためスキップします")
		return 0
	}
	defer func() {
		releaseCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := lock.Release(releaseCtx); err != nil {
			slog.Error("バッチロックの解放に失敗しました", "error", err)
		}
	}()
	batchUsecase := usecase.NewPlayerStatsRecalculationBatchUsecase(infrarepo.NewPlayerStatsBatchRepository(database))
	start := time.Now()
	result, executeErr := batchUsecase.Execute(ctx)
	slog.Info("プレイヤー統計再計算バッチを終了しました",
		"started_at", result.StartedAt, "operational_date", result.OperationalDate.Format(time.DateOnly),
		"current_version", result.CurrentVersion, "upper_bound_player_id", result.UpperBoundPlayerID,
		"processed", result.Processed, "success", result.Success, "current_preserved", result.CurrentPreserved,
		"legacy_rebuilt", result.LegacyRebuilt, "conflict_skipped", result.ConflictSkipped,
		"deleted_skipped", result.DeletedSkipped, "failed", result.Failed,
		"last_player_id", result.LastPlayerID, "duration", time.Since(start))
	if ctx.Err() != nil {
		return 0
	}
	if executeErr != nil {
		slog.Error("プレイヤー統計再計算バッチに失敗しました", "error", executeErr)
		return 1
	}
	return 0
}
