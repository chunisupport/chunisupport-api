package api_internal

import (
	"github.com/Qman110101/chunisupport-api/internal/domain/entity"
)

// ChartRankStatisticsDTO はランク別の人数統計を表します。
type ChartRankStatisticsDTO struct {
	S       int `json:"s"`        // Sランク人数
	SPlus   int `json:"s_plus"`   // S+ランク人数
	SS      int `json:"ss"`       // SSランク人数
	SSPlus  int `json:"ss_plus"`  // SS+ランク人数
	SSS     int `json:"sss"`      // SSSランク人数
	SSSPlus int `json:"sss_plus"` // SSS+ランク人数
}

// ChartLampStatisticsDTO はランプ別の人数統計を表します。
type ChartLampStatisticsDTO struct {
	AJ    int `json:"aj"`    // ALL JUSTICE人数
	FC    int `json:"fc"`    // FULL COMBO人数
	Other int `json:"other"` // その他ランプ人数
}

// ChartStatisticsByRatingDTO はレーティング帯ごとのランク・ランプ統計を表します。
type ChartStatisticsByRatingDTO struct {
	Rank ChartRankStatisticsDTO `json:"rank"` // ランク別統計
	Lamp ChartLampStatisticsDTO `json:"lamp"` // ランプ別統計
}

// ChartStatisticsDTO は譜面の統計情報を表します。
// キー: "15.0", "15.1", ..., "17.6", "17.7+"
// 統計データが存在する場合、全レーティング帯（15.0～17.7+）のデータを必ず含みます。
type ChartStatisticsDTO map[string]ChartStatisticsByRatingDTO

// getAllRatingTiers はすべてのレーティング帯のキーを返します（15.0～17.7+）。
func getAllRatingTiers() []string {
	tiers := make([]string, 0, 28)
	for i := 150; i <= 176; i++ {
		major := i / 10
		minor := i % 10
		tiers = append(tiers, string(rune('0'+major/10))+string(rune('0'+major%10))+"."+string(rune('0'+minor)))
	}
	tiers = append(tiers, "17.7+")
	return tiers
}

func NewEmptyChartStatisticsDTO() ChartStatisticsDTO {
	dto := make(ChartStatisticsDTO)
	for _, tier := range getAllRatingTiers() {
		dto[tier] = ChartStatisticsByRatingDTO{
			Rank: ChartRankStatisticsDTO{},
			Lamp: ChartLampStatisticsDTO{},
		}
	}
	return dto
}

// ToChartStatisticsDTO はエンティティのスライスからDTOに変換します。
// 統計データが存在しない場合はnilを返します。
// 統計データが存在する場合、全レーティング帯のデータを含みます（値が0でも省略しない）。
func ToChartStatisticsDTO(statsList []*entity.ChartStatistics) ChartStatisticsDTO {
	if len(statsList) == 0 {
		return nil
	}

	dto := NewEmptyChartStatisticsDTO()

	// ????????????????????
	statsMap := make(map[string]*entity.ChartStatistics)
	for _, stats := range statsList {
		statsMap[stats.GetRatingTierString()] = stats
	}

	// ?????????????????????0?????
	for _, tier := range getAllRatingTiers() {
		stats, ok := statsMap[tier]
		if !ok {
			continue
		}
		dto[tier] = ChartStatisticsByRatingDTO{
			Rank: ChartRankStatisticsDTO{
				S:       stats.RankS,
				SPlus:   stats.RankSPlus,
				SS:      stats.RankSS,
				SSPlus:  stats.RankSSPlus,
				SSS:     stats.RankSSS,
				SSSPlus: stats.RankSSSPlus,
			},
			Lamp: ChartLampStatisticsDTO{
				AJ:    stats.LampAJ,
				FC:    stats.LampFC,
				Other: stats.LampOther,
			},
		}
	}

	return dto
}
