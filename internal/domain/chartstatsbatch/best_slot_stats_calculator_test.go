package chartstatsbatch

import (
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/ratingband"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBestSlotStatsCalculator_全譜面と全レーティング帯の行を作る(t *testing.T) {
	// Given
	min17, max171, min171, max172, min18 := 17.0, 17.1, 17.1, 17.2, 18.0
	calc := NewBestSlotStatsCalculator([]*ratingband.RatingBand{
		{ID: 0},
		{ID: 22, MinInclusive: &min17, MaxExclusive: &max171},
		{ID: 23, MinInclusive: &min171, MaxExclusive: &max172},
		{ID: 99, MinInclusive: &min18},
	})
	calc.AddEligiblePlayer(EligiblePlayer{BestAverageRating: 17.05})
	calc.AddEligiblePlayer(EligiblePlayer{BestAverageRating: 17.09})
	calc.AddEligiblePlayer(EligiblePlayer{BestAverageRating: 17.15})
	calc.AddBestSlotRecord(BestSlotRecord{ChartID: 10, BestAverageRating: 17.05})
	calc.AddBestSlotRecord(BestSlotRecord{ChartID: 10, BestAverageRating: 17.15})

	// When
	results := calc.Results([]int{10, 20})

	// Then
	require.Len(t, results, 8)
	assert.Equal(t, &entity.ChartBestSlotStatsByRatingBand{
		ChartID: 10, RatingBandID: 0, BestPlayerCount: 2, EligiblePlayerCount: 3, BestPlayerPercentage: float64Ptr(66.6667),
	}, findBestSlotStats(t, results, 10, 0))
	assert.Equal(t, &entity.ChartBestSlotStatsByRatingBand{
		ChartID: 10, RatingBandID: 22, BestPlayerCount: 1, EligiblePlayerCount: 2, BestPlayerPercentage: float64Ptr(50),
	}, findBestSlotStats(t, results, 10, 22))
	assert.Equal(t, &entity.ChartBestSlotStatsByRatingBand{
		ChartID: 20, RatingBandID: 22, BestPlayerCount: 0, EligiblePlayerCount: 2, BestPlayerPercentage: float64Ptr(0),
	}, findBestSlotStats(t, results, 20, 22), "誰も採用していない譜面は0%")
	assert.Equal(t, &entity.ChartBestSlotStatsByRatingBand{
		ChartID: 10, RatingBandID: 99,
	}, findBestSlotStats(t, results, 10, 99), "対象プレイヤーがいない帯は採用率をnilにする")
}

func TestBestSlotStatsCalculator_対象外の譜面の採用は無視する(t *testing.T) {
	// Given: 削除済み楽曲などで対象譜面に含まれない譜面の採用記録
	calc := NewBestSlotStatsCalculator([]*ratingband.RatingBand{{ID: 0}})
	calc.AddEligiblePlayer(EligiblePlayer{BestAverageRating: 17.0})
	calc.AddBestSlotRecord(BestSlotRecord{ChartID: 999, BestAverageRating: 17.0})

	// When
	results := calc.Results([]int{10})

	// Then
	require.Len(t, results, 1)
	assert.Equal(t, 10, results[0].ChartID)
	assert.Zero(t, results[0].BestPlayerCount)
}

func findBestSlotStats(t *testing.T, stats []*entity.ChartBestSlotStatsByRatingBand, chartID, bandID int) *entity.ChartBestSlotStatsByRatingBand {
	t.Helper()
	for _, stat := range stats {
		if stat.ChartID == chartID && stat.RatingBandID == bandID {
			return stat
		}
	}
	require.FailNow(t, "統計が見つかりません", "chart=%d band=%d", chartID, bandID)
	return nil
}
