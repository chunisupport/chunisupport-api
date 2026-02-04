package testutil

import (
	"context"

	"github.com/Qman110101/chunisupport-api/internal/domain/entity"
)

// MockChartStatsUsecase は譜面統計ユースケースのテスト用モックです。
type MockChartStatsUsecase struct {
	Stats *entity.SongChartStats
	Err   error
}

func (m *MockChartStatsUsecase) GetSongStatsByDisplayID(ctx context.Context, displayID string) (*entity.SongChartStats, error) {
	return m.Stats, m.Err
}
