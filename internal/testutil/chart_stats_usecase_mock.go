package testutil

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// MockChartStatsUsecase は譜面統計ユースケースのテスト用モックです。
type MockChartStatsUsecase struct {
	Stats       *entity.SongChartStats
	SingleStats *entity.SingleChartStats
	Err         error
}

func (m *MockChartStatsUsecase) GetSongStatsByDisplayID(ctx context.Context, displayID string, requesterAccountTypeID *int) (*entity.SongChartStats, error) {
	return m.Stats, m.Err
}

func (m *MockChartStatsUsecase) GetChartStatsByDisplayIDAndDifficulty(ctx context.Context, displayID, difficultyName string, requesterAccountTypeID *int) (*entity.SingleChartStats, error) {
	return m.SingleStats, m.Err
}
