package usecase

import (
	"context"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/ratingband"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type bestSlotRankingQueryServiceMock struct {
	result       *repository.BestSlotRankingPage
	err          error
	ratingBandID int
	limit        int
}

func (m *bestSlotRankingQueryServiceMock) List(_ context.Context, ratingBandID int, cursor *repository.BestSlotRankingCursor, limit int) (*repository.BestSlotRankingPage, error) {
	m.ratingBandID = ratingBandID
	m.limit = limit
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

func TestBestSlotRankingUsecase_Get_レート帯ラベルを内部IDへ解決する(t *testing.T) {
	// Given
	percentage := 25.0
	query := &bestSlotRankingQueryServiceMock{result: &repository.BestSlotRankingPage{
		EligiblePlayerCount: 40,
		Items: []repository.BestSlotRankingItem{{
			Rank:                 1,
			SongDisplayID:        "0000000000000001",
			SongTitle:            "楽曲名",
			Difficulty:           "MASTER",
			ChartConst:           chartconstant.ChartConstant(14.8),
			BestPlayerCount:      10,
			BestPlayerPercentage: percentage,
		}},
	}}
	provider := &StubChartStatsMasterProvider{bands: []*ratingband.RatingBand{{ID: 22, Label: "17.0"}}}
	u := NewBestSlotRankingUsecase(query, provider)

	// When
	result, err := u.Get(context.Background(), "17.0", nil, 50)

	// Then
	require.NoError(t, err)
	assert.Equal(t, "17.0", result.RatingBand)
	assert.Equal(t, 40, result.EligiblePlayerCount)
	require.Len(t, result.Ranking, 1)
	assert.Equal(t, "0000000000000001", result.Ranking[0].SongID)
	assert.Equal(t, "楽曲名", result.Ranking[0].Title)
	assert.Equal(t, 22, query.ratingBandID)
	assert.Equal(t, 50, query.limit)
}

func TestBestSlotRankingUsecase_Get_別レート帯のカーソルはエラー(t *testing.T) {
	// Given
	u := NewBestSlotRankingUsecase(
		&bestSlotRankingQueryServiceMock{},
		&StubChartStatsMasterProvider{bands: []*ratingband.RatingBand{{ID: 22, Label: "17.0"}}},
	)
	cursor := &repository.BestSlotRankingCursor{RatingBand: "17.1"}

	// When
	result, err := u.Get(context.Background(), "17.0", cursor, 50)

	// Then
	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrInvalidBestSlotRankingCursor)
}

func TestBestSlotRankingUsecase_Get_存在しないレート帯はエラー(t *testing.T) {
	// Given
	u := NewBestSlotRankingUsecase(
		&bestSlotRankingQueryServiceMock{},
		&StubChartStatsMasterProvider{bands: []*ratingband.RatingBand{{ID: 22, Label: "17.0"}}},
	)

	// When
	result, err := u.Get(context.Background(), "17.9", nil, 50)

	// Then
	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrInvalidRatingBand)
}
