package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFriendChartRankingUsecase_GetStandard_同点を同順位にする(t *testing.T) {
	// Given
	chartConst, err := chartconstant.NewChartConstant(14.5)
	require.NoError(t, err)
	updatedAt := time.Date(2026, 7, 9, 12, 0, 0, 0, time.UTC)
	query := &friendChartRankingQueryMock{
		chart: &repository.FriendChartRankingChart{
			SongDisplayID:  "0000000000000001",
			SongTitle:      "楽曲名",
			SongArtist:     "アーティスト",
			Difficulty:     "MASTER",
			Const:          chartConst,
			IsConstUnknown: false,
			ChartID:        10,
		},
		records: []*repository.FriendChartRankingRecord{
			{UserID: 2, Username: "friend1", PlayerName: "FRIEND1", Score: 1_009_500, ComboLamp: "ALL JUSTICE", UpdatedAt: updatedAt},
			{UserID: 3, Username: "friend2", PlayerName: "FRIEND2", Score: 1_009_500, ComboLamp: "FULL COMBO", UpdatedAt: updatedAt},
			{UserID: 1, Username: "me", PlayerName: "ME", Score: 1_009_000, ComboLamp: "NONE", UpdatedAt: updatedAt},
		},
	}
	u := NewFriendChartRankingUsecase(nil, query)

	// When
	got, err := u.GetStandard(context.Background(), 1, "0000000000000001", "MASTER")

	// Then
	require.NoError(t, err)
	require.Len(t, got.Ranking, 3)
	assert.Equal(t, []int{1, 1, 3}, []int{got.Ranking[0].Rank, got.Ranking[1].Rank, got.Ranking[2].Rank})
	require.NotNil(t, got.MyRank)
	assert.Equal(t, 3, *got.MyRank)
	assert.Equal(t, 3, got.Total)
	assert.True(t, got.Ranking[2].IsSelf)
	assert.Nil(t, got.Ranking[2].ComboLamp)
}

func TestFriendChartRankingUsecase_GetStandard_譜面がない場合はErrChartNotFound(t *testing.T) {
	// Given
	u := NewFriendChartRankingUsecase(nil, &friendChartRankingQueryMock{})

	// When
	got, err := u.GetStandard(context.Background(), 1, "0000000000000001", "MASTER")

	// Then
	assert.Nil(t, got)
	assert.ErrorIs(t, err, ErrChartNotFound)
}

type friendChartRankingQueryMock struct {
	chart   *repository.FriendChartRankingChart
	records []*repository.FriendChartRankingRecord
}

func (m *friendChartRankingQueryMock) FindChart(ctx context.Context, exec repository.Executor, displayID string, difficulty string) (*repository.FriendChartRankingChart, error) {
	return m.chart, nil
}

func (m *friendChartRankingQueryMock) ListRecords(ctx context.Context, exec repository.Executor, userID int, chartID int) ([]*repository.FriendChartRankingRecord, error) {
	return m.records, nil
}
