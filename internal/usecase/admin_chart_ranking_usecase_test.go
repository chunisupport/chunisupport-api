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

func TestAdminChartRankingUsecase_GetStandard_上位レコードと全プレイ人数を返す(t *testing.T) {
	// Given
	chartConst, err := chartconstant.NewChartConstant(14.5)
	require.NoError(t, err)
	updatedAt := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
	query := &adminChartRankingQueryMock{
		standardData: &repository.AdminChartRankingData{
			Chart: &repository.AdminChartRankingChart{
				SongDisplayID: "0000000000000001",
				SongTitle:     "楽曲名",
				SongArtist:    "アーティスト",
				Difficulty:    "MASTER",
				Const:         chartConst,
			},
			Records: []*repository.AdminChartRankingRecord{
				{Username: "user1", PlayerName: "PLAYER1", Score: 1_009_500, ComboLamp: "ALL JUSTICE", UpdatedAt: updatedAt},
				{Username: "user2", PlayerName: "PLAYER2", Score: 1_009_500, ComboLamp: "FULL COMBO", UpdatedAt: updatedAt},
				{Username: "user3", PlayerName: "PLAYER3", Score: 1_009_000, ComboLamp: "NONE", UpdatedAt: updatedAt},
			},
			Total: 234,
		},
	}
	u := NewAdminChartRankingUsecase(query)

	// When
	got, err := u.GetStandard(context.Background(), "0000000000000001", "MASTER")

	// Then
	require.NoError(t, err)
	require.Len(t, got.Ranking, 3)
	assert.Equal(t, []int{1, 1, 3}, []int{got.Ranking[0].Rank, got.Ranking[1].Rank, got.Ranking[2].Rank})
	assert.Equal(t, 234, got.Total)
	assert.Nil(t, got.Ranking[2].ComboLamp)
	assert.Positive(t, got.Ranking[0].Rating)
	assert.Equal(t, 100, query.gotLimit)
}

func TestAdminChartRankingUsecase_GetWorldsend_WORLD送信固有情報を返す(t *testing.T) {
	// Given
	levelStar := 5
	attribute := "狂"
	query := &adminChartRankingQueryMock{
		worldsendData: &repository.AdminChartRankingData{
			Chart: &repository.AdminChartRankingChart{
				SongDisplayID:  "0000000000000002",
				SongTitle:      "WE楽曲名",
				SongArtist:     "WEアーティスト",
				Difficulty:     "WORLD'S END",
				LevelStar:      &levelStar,
				Attribute:      &attribute,
				IsConstUnknown: true,
				IsWorldsend:    true,
			},
			Records: []*repository.AdminChartRankingRecord{{Username: "user1", Score: 1_010_000}},
			Total:   1,
		},
	}
	u := NewAdminChartRankingUsecase(query)

	// When
	got, err := u.GetWorldsend(context.Background(), "0000000000000002")

	// Then
	require.NoError(t, err)
	assert.True(t, got.Chart.IsWorldsend)
	assert.Equal(t, &levelStar, got.Chart.LevelStar)
	assert.Equal(t, &attribute, got.Chart.Attribute)
	assert.Zero(t, got.Ranking[0].Rating)
	assert.Equal(t, 100, query.gotLimit)
}

func TestAdminChartRankingUsecase_譜面がない場合はErrChartNotFound(t *testing.T) {
	tests := []struct {
		name string
		get  func(AdminChartRankingUsecase) (*AdminChartRankingResult, error)
	}{
		{
			name: "通常譜面",
			get: func(u AdminChartRankingUsecase) (*AdminChartRankingResult, error) {
				return u.GetStandard(context.Background(), "0000000000000001", "MASTER")
			},
		},
		{
			name: "WORLD'S END",
			get: func(u AdminChartRankingUsecase) (*AdminChartRankingResult, error) {
				return u.GetWorldsend(context.Background(), "0000000000000002")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			u := NewAdminChartRankingUsecase(&adminChartRankingQueryMock{})

			// When
			got, err := tt.get(u)

			// Then
			assert.Nil(t, got)
			assert.ErrorIs(t, err, ErrChartNotFound)
		})
	}
}

type adminChartRankingQueryMock struct {
	standardData  *repository.AdminChartRankingData
	worldsendData *repository.AdminChartRankingData
	gotLimit      int
}

func (m *adminChartRankingQueryMock) GetStandard(_ context.Context, _ string, _ string, limit int) (*repository.AdminChartRankingData, error) {
	m.gotLimit = limit
	return m.standardData, nil
}

func (m *adminChartRankingQueryMock) GetWorldsend(_ context.Context, _ string, limit int) (*repository.AdminChartRankingData, error) {
	m.gotLimit = limit
	return m.worldsendData, nil
}
