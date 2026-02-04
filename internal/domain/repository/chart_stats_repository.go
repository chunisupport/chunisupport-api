package repository

import (
	"context"

	"github.com/Qman110101/chunisupport-api/internal/domain/entity"
)

// ChartStatsRepository は統計データの参照を扱うリポジトリです。
type ChartStatsRepository interface {
	// FindRatingBands はレーティング帯マスタ一覧を返します。
	FindRatingBands(ctx context.Context) ([]*entity.RatingBand, error)
	// FindChartStatsByChartIDs は譜面ID一覧に対する統計を返します。
	FindChartStatsByChartIDs(ctx context.Context, chartIDs []int) ([]*entity.ChartStatsByRatingBand, error)
}
