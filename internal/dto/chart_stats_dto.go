package dto

import (
	"log/slog"
	"strconv"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// ChartStatsResponse は譜面統計APIのレスポンスです。
type ChartStatsResponse struct {
	SongID string                         `json:"song_id"`
	Charts map[string]*ChartStatsChartDTO `json:"charts"`
}

// ChartStatsChartDTO は譜面ごとの統計情報です。
type ChartStatsChartDTO struct {
	Stats []*ChartStatsByRatingBandDTO `json:"stats"`
}

// ChartStatsByRatingBandDTO はレーティング帯別統計のDTOです。
type ChartStatsByRatingBandDTO struct {
	RatingBand   string             `json:"rating_band"`
	Rank         ChartRankStatsDTO  `json:"rank"`
	Combo        ChartComboStatsDTO `json:"combo"`
	Clear        ChartClearStatsDTO `json:"clear"`
	AverageScore *float64           `json:"average_score"`
	PlayerCount  int                `json:"player_count"`
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

// ChartClearStatsDTO はクリアランプ別人数のDTOです。
type ChartClearStatsDTO struct {
	Failed      int `json:"failed"`
	Clear       int `json:"clear"`
	Hard        int `json:"hard"`
	Brave       int `json:"brave"`
	Absolute    int `json:"absolute"`
	Catastrophy int `json:"catastrophy"`
}

// ToChartStatsResponse は SongChartStats を ChartStatsResponse に変換します。
func ToChartStatsResponse(stats *entity.SongChartStats, ratingBands []*entity.RatingBand) *ChartStatsResponse {
	if stats == nil {
		return nil
	}

	ratingBandLabels := make(map[int]string, len(ratingBands))
	for _, band := range ratingBands {
		ratingBandLabels[band.ID] = band.Label
	}

	charts := make(map[string]*ChartStatsChartDTO, len(stats.Charts))
	for key, chartStats := range stats.Charts {
		statsDTO := make([]*ChartStatsByRatingBandDTO, 0, len(chartStats))
		for _, stat := range chartStats {
			label, ok := ratingBandLabels[stat.RatingBandID]
			if !ok {
				slog.Warn("Rating band label not found", "rating_band_id", stat.RatingBandID)
				label = strconv.Itoa(stat.RatingBandID)
			}

			statsDTO = append(statsDTO, &ChartStatsByRatingBandDTO{
				RatingBand: label,
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
				Clear: ChartClearStatsDTO{
					Failed:      stat.Clear.Failed,
					Clear:       stat.Clear.Clear,
					Hard:        stat.Clear.Hard,
					Brave:       stat.Clear.Brave,
					Absolute:    stat.Clear.Absolute,
					Catastrophy: stat.Clear.Catastrophy,
				},
				AverageScore: stat.AverageScore,
				PlayerCount:  stat.PlayerCount,
			})
		}

		charts[key] = &ChartStatsChartDTO{
			Stats: statsDTO,
		}
	}

	return &ChartStatsResponse{
		SongID: stats.SongID,
		Charts: charts,
	}
}
