package repository

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/ratingband"
)

// ChartStatsRepository は統計データの参照を扱うリポジトリです。
type ChartStatsRepository interface {
	// FindRatingBands はレーティング帯マスタ一覧を返します。
	FindRatingBands(ctx context.Context, exec Executor) ([]*ratingband.RatingBand, error)
	// FindChartStatsByChartIDs は譜面ID一覧に対する統計を返します。
	FindChartStatsByChartIDs(ctx context.Context, exec Executor, chartIDs []int) ([]*entity.ChartStatsByRatingBand, error)
}
