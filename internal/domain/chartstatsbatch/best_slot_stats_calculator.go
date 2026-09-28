package chartstatsbatch

import (
	"math"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/ratingband"
)

type bestSlotStatsKey struct {
	chartID      int
	ratingBandID int
}

// BestSlotStatsCalculator は譜面ごとのベスト枠採用率をレーティング帯ごとに集計します。
type BestSlotStatsCalculator struct {
	ratingBands         []*ratingband.RatingBand
	eligiblePlayerCount map[int]int
	bestPlayerCount     map[bestSlotStatsKey]int
}

// NewBestSlotStatsCalculator は指定したレーティング帯で集計する BestSlotStatsCalculator を生成します。
func NewBestSlotStatsCalculator(ratingBands []*ratingband.RatingBand) *BestSlotStatsCalculator {
	return &BestSlotStatsCalculator{
		ratingBands:         ratingBands,
		eligiblePlayerCount: make(map[int]int, len(ratingBands)),
		bestPlayerCount:     make(map[bestSlotStatsKey]int),
	}
}

// AddEligiblePlayer は採用率の母数となるプレイヤーを加えます。
func (c *BestSlotStatsCalculator) AddEligiblePlayer(player EligiblePlayer) {
	for _, band := range c.ratingBands {
		if band.Contains(player.BestAverageRating) {
			c.eligiblePlayerCount[band.ID]++
		}
	}
}

// AddBestSlotRecord はベスト枠に入っている譜面を加えます。
func (c *BestSlotStatsCalculator) AddBestSlotRecord(record BestSlotRecord) {
	for _, band := range c.ratingBands {
		if band.Contains(record.BestAverageRating) {
			c.bestPlayerCount[bestSlotStatsKey{chartID: record.ChartID, ratingBandID: band.ID}]++
		}
	}
}

// Results は chartIDs のすべての譜面とすべてのレーティング帯について採用統計を返します。
// 採用者がいない譜面も0%として一覧に載せるため、chartIDs には集計対象の全譜面を渡します。
// chartIDs に含まれない譜面の採用記録は結果に含めません。
func (c *BestSlotStatsCalculator) Results(chartIDs []int) []*entity.ChartBestSlotStatsByRatingBand {
	results := make([]*entity.ChartBestSlotStatsByRatingBand, 0, len(chartIDs)*len(c.ratingBands))
	for _, chartID := range chartIDs {
		for _, band := range c.ratingBands {
			bestCount := c.bestPlayerCount[bestSlotStatsKey{chartID: chartID, ratingBandID: band.ID}]
			eligibleCount := c.eligiblePlayerCount[band.ID]
			results = append(results, &entity.ChartBestSlotStatsByRatingBand{
				ChartID:              chartID,
				RatingBandID:         band.ID,
				BestPlayerCount:      bestCount,
				EligiblePlayerCount:  eligibleCount,
				BestPlayerPercentage: bestPlayerPercentage(bestCount, eligibleCount),
			})
		}
	}
	return results
}

// bestPlayerPercentage は小数第4位で丸めた採用率を返します。母数が0の帯は採用率を定義できないため nil です。
func bestPlayerPercentage(bestPlayerCount, eligiblePlayerCount int) *float64 {
	if eligiblePlayerCount == 0 {
		return nil
	}
	percentage := float64(bestPlayerCount) / float64(eligiblePlayerCount) * 100
	percentage = math.Round(percentage*bestPlayerPercentageScale) / bestPlayerPercentageScale
	return &percentage
}
