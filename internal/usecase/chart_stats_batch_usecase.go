package usecase

import (
	"context"
	"fmt"

	"github.com/chunisupport/chunisupport-api/internal/domain/chartstatsbatch"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
)

// ChartStatsBatchResult は譜面統計バッチ1回分の件数です。
// 監視ログで集計元の欠落に気付けるよう、統計の種類ごとに読み取り件数と書き込み件数を分けて保持します。
type ChartStatsBatchResult struct {
	ChartRecordCount          int
	ChartStatsCount           int
	WorldsendChartRecordCount int
	WorldsendChartStatsCount  int
	EligiblePlayerCount       int
	BestSlotRecordCount       int
	BestSlotStatsCount        int
}

// ChartStatsBatchUsecase は譜面統計を集計し、統計テーブルを再構築します。
type ChartStatsBatchUsecase struct {
	repo repository.ChartStatsBatchRepository
}

// NewChartStatsBatchUsecase は ChartStatsBatchUsecase を生成します。
func NewChartStatsBatchUsecase(repo repository.ChartStatsBatchRepository) *ChartStatsBatchUsecase {
	return &ChartStatsBatchUsecase{repo: repo}
}

// Execute は譜面統計・WORLD'S END譜面統計・ベスト枠採用率を集計し、統計テーブルを丸ごと入れ替えます。
// すべてを集計し終えてから単一トランザクションで入れ替えるため、実行中や失敗時にも公開APIからは直前の統計が見え続けます。
func (u *ChartStatsBatchUsecase) Execute(ctx context.Context) (ChartStatsBatchResult, error) {
	ratingBands, err := u.repo.FindRatingBands(ctx)
	if err != nil {
		return ChartStatsBatchResult{}, fmt.Errorf("load rating bands: %w", err)
	}

	chartStats := chartstatsbatch.NewChartStatsCalculator(ratingBands)
	worldsendChartStats := chartstatsbatch.NewChartStatsCalculator(ratingBands)
	bestSlotStats := chartstatsbatch.NewBestSlotStatsCalculator(ratingBands)
	var result ChartStatsBatchResult
	var bestSlotChartIDs []int
	err = u.repo.StreamSource(ctx, repository.ChartStatsBatchSourceCallbacks{
		ChartRecord: func(record chartstatsbatch.ChartRecord) error {
			chartStats.Add(record)
			return nil
		},
		WorldsendChartRecord: func(record chartstatsbatch.ChartRecord) error {
			worldsendChartStats.Add(record)
			return nil
		},
		EligiblePlayer: func(player chartstatsbatch.EligiblePlayer) error {
			bestSlotStats.AddEligiblePlayer(player)
			result.EligiblePlayerCount++
			return nil
		},
		BestSlotRecord: func(record chartstatsbatch.BestSlotRecord) error {
			bestSlotStats.AddBestSlotRecord(record)
			result.BestSlotRecordCount++
			return nil
		},
		BestSlotChart: func(chartID int) error {
			bestSlotChartIDs = append(bestSlotChartIDs, chartID)
			return nil
		},
	})
	if err != nil {
		return ChartStatsBatchResult{}, fmt.Errorf("read chart stats source: %w", err)
	}

	snapshot := &chartstatsbatch.Snapshot{
		ChartStats:          chartStats.Results(),
		WorldsendChartStats: worldsendChartStats.Results(),
		BestSlotStats:       bestSlotStats.Results(bestSlotChartIDs),
	}
	result.ChartRecordCount = chartStats.RecordCount()
	result.ChartStatsCount = len(snapshot.ChartStats)
	result.WorldsendChartRecordCount = worldsendChartStats.RecordCount()
	result.WorldsendChartStatsCount = len(snapshot.WorldsendChartStats)
	result.BestSlotStatsCount = len(snapshot.BestSlotStats)

	if err := u.repo.ReplaceAll(ctx, snapshot); err != nil {
		return ChartStatsBatchResult{}, fmt.Errorf("replace chart stats: %w", err)
	}
	return result, nil
}
