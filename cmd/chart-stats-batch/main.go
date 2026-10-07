// Command chart-stats-batch はプレイヤーの記録を集計し、譜面統計・ベスト枠採用率の統計テーブルを再構築するバッチです。
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/chunisupport/chunisupport-api/internal/config"
	"github.com/chunisupport/chunisupport-api/internal/infra/db"
	"github.com/chunisupport/chunisupport-api/internal/infra/logger"
	infrarepo "github.com/chunisupport/chunisupport-api/internal/infra/repository"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
)

func main() {
	os.Exit(run())
}

// validateArgs は引数がないことを確認します。
// --dry-run などの未知の引数を無視すると、更新されないと誤認したまま統計を書き換えるおそれがあるため、エラーにします。
func validateArgs(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("unexpected arguments: %v", args)
	}
	return nil
}

func run() int {
	if err := validateArgs(os.Args[1:]); err != nil {
		slog.Error("引数が不正です", "error", err)
		return 2
	}

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

	database, err := db.ConnectWithRetry(ctx, cfg.Database.DbConfig)
	if err != nil {
		slog.Error("DB接続に失敗しました", "error", err)
		return 1
	}
	defer database.Close()

	jobUsecase := usecase.NewChartStatsBatchJobUsecase(
		ctx,
		db.NewAdvisoryLockProvider(database),
		infrarepo.NewChartStatsBatchJobRepository(database),
		usecase.NewChartStatsBatchUsecase(infrarepo.NewChartStatsBatchRepository(database)),
	)
	acquired, err := jobUsecase.RunFromCLI(ctx)
	if err != nil {
		slog.Error("譜面統計バッチに失敗しました", "error", err)
		return 1
	}
	if !acquired {
		slog.Info("別の譜面統計バッチが実行中のためスキップします")
	}
	return 0
}
