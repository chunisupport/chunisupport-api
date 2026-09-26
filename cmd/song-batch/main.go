// Command song-batch は外部データソースから楽曲・譜面データを取得し、MySQL へ統合するバッチです。
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/chunisupport/chunisupport-api/internal/config"
	"github.com/chunisupport/chunisupport-api/internal/domain/songbatch"
	"github.com/chunisupport/chunisupport-api/internal/infra/db"
	"github.com/chunisupport/chunisupport-api/internal/infra/logger"
	infrarepo "github.com/chunisupport/chunisupport-api/internal/infra/repository"
	infrasongbatch "github.com/chunisupport/chunisupport-api/internal/infra/songbatch"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
)

func main() {
	os.Exit(run())
}

// parseRunRequest はコマンドライン引数から実行条件を組み立てます。
func parseRunRequest(args []string) (songbatch.RunRequest, error) {
	flags := flag.NewFlagSet("song-batch", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	majorUpdate := flags.Bool("major-update", false, "大型アップデート用モード（公式データと追加楽曲のみを使用し、定数更新ルールを適用する）")
	fillMissingReleaseDate := flags.Bool("fill-missing-release-date", false, "どのデータソースにも日付がなくMySQLにも存在しない新規楽曲へ実行日(JST)をreleased_atとして補完する")
	if err := flags.Parse(args); err != nil {
		return songbatch.RunRequest{}, err
	}
	if flags.NArg() != 0 {
		return songbatch.RunRequest{}, fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	return songbatch.NewRunRequest(*majorUpdate, *fillMissingReleaseDate), nil
}

func run() int {
	req, err := parseRunRequest(os.Args[1:])
	if err != nil {
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

	jobUsecase := usecase.NewSongBatchJobUsecase(
		ctx,
		db.NewAdvisoryLockProvider(database),
		infrarepo.NewSongBatchJobRepository(database),
		infrasongbatch.NewSongBatchUsecase(database, cfg.SongBatch.WikiBaseURL),
		nil,
	)
	acquired, err := jobUsecase.RunFromCLI(ctx, req)
	if err != nil {
		slog.Error("楽曲バッチに失敗しました", "error", err)
		return 1
	}
	if !acquired {
		if req.LockConflictIsError() {
			slog.Error("別の楽曲バッチが実行中のため開始できません")
			return 1
		}
		slog.Info("別の楽曲バッチが実行中のためスキップします")
	}
	return 0
}
