package dto

import "github.com/Qman110101/chunisupport-api/internal/domain/entity"

// ChartStatsResponse は譜面統計APIのレスポンスです。
type ChartStatsResponse struct {
	SongID      string                         `json:"song_id"`
	RatingBands []*RatingBandDTO               `json:"rating_bands"`
	Charts      map[string]*ChartStatsChartDTO `json:"charts"`
}

// RatingBandDTO はレーティング帯マスタのDTOです。
type RatingBandDTO struct {
	ID           int      `json:"id"`
	Label        string   `json:"label"`
	MinInclusive *float64 `json:"min_inclusive"`
	MaxExclusive *float64 `json:"max_exclusive"`
	SortOrder    int      `json:"sort_order"`
}

// ChartStatsChartDTO は譜面ごとの統計情報です。
type ChartStatsChartDTO struct {
	Stats []*ChartStatsByRatingBandDTO `json:"stats"`
}

// ChartStatsByRatingBandDTO はレーティング帯別統計のDTOです。
type ChartStatsByRatingBandDTO struct {
	RatingBandID int                `json:"rating_band_id"`
	Rank         ChartRankStatsDTO  `json:"rank"`
	Combo        ChartComboStatsDTO `json:"combo"`
	Clear        map[string]int     `json:"clear"`
}

// ChartRankStatsDTO はランク別人数のDTOです。
type ChartRankStatsDTO struct {
	AAAL int `json:"aaal"`
	S    int `json:"s"`
	SP   int `json:"sp"`
	SS   int `json:"ss"`
	SSP  int `json:"ssp"`
	SSS  int `json:"sss"`
	SSSP int `json:"sssp"`
	Max  int `json:"max"`
}

// ChartComboStatsDTO はコンボランプ別人数のDTOです。
type ChartComboStatsDTO struct {
	None int `json:"none"`
	FC   int `json:"fc"`
	AJ   int `json:"aj"`
}

// ToChartStatsResponse は SongChartStats を ChartStatsResponse に変換します。
func ToChartStatsResponse(stats *entity.SongChartStats) *ChartStatsResponse {
	if stats == nil {
		return nil
	}

	ratingBands := make([]*RatingBandDTO, 0, len(stats.RatingBands))
	for _, band := range stats.RatingBands {
		ratingBands = append(ratingBands, &RatingBandDTO{
			ID:           band.ID,
			Label:        band.Label,
			MinInclusive: band.MinInclusive,
			MaxExclusive: band.MaxExclusive,
			SortOrder:    band.SortOrder,
		})
	}

	charts := make(map[string]*ChartStatsChartDTO, len(stats.Charts))
	for key, chartStats := range stats.Charts {
		statsDTO := make([]*ChartStatsByRatingBandDTO, 0, len(chartStats))
		for _, stat := range chartStats {
			clearStats := make(map[string]int, len(stat.Clear))
			for clearKey, value := range stat.Clear {
				clearStats[clearKey] = value
			}

			statsDTO = append(statsDTO, &ChartStatsByRatingBandDTO{
				RatingBandID: stat.RatingBandID,
				Rank: ChartRankStatsDTO{
					AAAL: stat.Rank.AAAL,
					S:    stat.Rank.S,
					SP:   stat.Rank.SP,
					SS:   stat.Rank.SS,
					SSP:  stat.Rank.SSP,
					SSS:  stat.Rank.SSS,
					SSSP: stat.Rank.SSSP,
					Max:  stat.Rank.Max,
				},
				Combo: ChartComboStatsDTO{
					None: stat.Combo.None,
					FC:   stat.Combo.FC,
					AJ:   stat.Combo.AJ,
				},
				Clear: clearStats,
			})
		}

		charts[key] = &ChartStatsChartDTO{
			Stats: statsDTO,
		}
	}

	return &ChartStatsResponse{
		SongID:      stats.SongID,
		RatingBands: ratingBands,
		Charts:      charts,
	}
}
