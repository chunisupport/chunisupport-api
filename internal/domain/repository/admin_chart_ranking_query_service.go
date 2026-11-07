package repository

import (
	"context"
)

type AdminChartRankingData struct {
	Chart   *ChartRankingChart
	Records []*ChartRankingRecord
	Total   int
}

type AdminChartRankingQueryService interface {
	GetStandard(ctx context.Context, displayID string, difficulty string, limit int) (*AdminChartRankingData, error)
	GetWorldsend(ctx context.Context, displayID string, limit int) (*AdminChartRankingData, error)
}
