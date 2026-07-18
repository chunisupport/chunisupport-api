package api_internal

import (
	"log/slog"
	"math"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/ratingband"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
)

// BestSlotRankingResponse はベスト枠平均レート帯別の譜面採用率ランキングです。
type BestSlotRankingResponse struct {
	RatingBand          string                    `json:"rating_band"`
	EligiblePlayerCount int                       `json:"eligible_player_count"`
	Ranking             []BestSlotRankingEntryDTO `json:"ranking"`
	NextCursor          *string                   `json:"next_cursor"`
}

// BestSlotRankingEntryDTO はランキング内の譜面1件です。
type BestSlotRankingEntryDTO struct {
	Rank                 int                     `json:"rank"`
	Song                 BestSlotRankingSongDTO  `json:"song"`
	Chart                BestSlotRankingChartDTO `json:"chart"`
	BestPlayerCount      int                     `json:"best_player_count"`
	BestPlayerPercentage float64                 `json:"best_player_percentage"`
	AverageScore         *float64                `json:"average_score"`
}

// BestSlotRankingSongDTO はランキング表示に必要な楽曲情報です。
type BestSlotRankingSongDTO struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// BestSlotRankingChartDTO はランキング表示に必要な譜面情報です。
type BestSlotRankingChartDTO struct {
	Difficulty     string  `json:"difficulty"`
	Const          float64 `json:"const"`
	IsConstUnknown bool    `json:"is_const_unknown"`
}

// SingleChartBestSlotStatsResponse は単一譜面のベスト枠採用統計です。
type SingleChartBestSlotStatsResponse struct {
	SongID string                              `json:"song_id"`
	Stats  []ChartBestSlotStatsByRatingBandDTO `json:"stats"`
}

// ChartBestSlotStatsByRatingBandDTO はレート帯別のベスト枠採用統計です。
type ChartBestSlotStatsByRatingBandDTO struct {
	RatingBand           string   `json:"rating_band"`
	BestPlayerCount      int      `json:"best_player_count"`
	EligiblePlayerCount  int      `json:"eligible_player_count"`
	BestPlayerPercentage *float64 `json:"best_player_percentage"`
}

// ToBestSlotRankingResponse はユースケース結果を内部APIレスポンスへ変換します。
func ToBestSlotRankingResponse(result *usecase.BestSlotRankingResult, nextCursor *string) *BestSlotRankingResponse {
	if result == nil {
		return nil
	}
	response := &BestSlotRankingResponse{
		RatingBand: result.RatingBand, EligiblePlayerCount: result.EligiblePlayerCount,
		Ranking: make([]BestSlotRankingEntryDTO, 0, len(result.Ranking)), NextCursor: nextCursor,
	}
	for _, entry := range result.Ranking {
		response.Ranking = append(response.Ranking, BestSlotRankingEntryDTO{
			Rank:            entry.Rank,
			Song:            BestSlotRankingSongDTO{ID: entry.SongID, Title: entry.Title},
			Chart:           BestSlotRankingChartDTO{Difficulty: entry.Difficulty, Const: entry.ChartConst.Float64(), IsConstUnknown: entry.IsConstUnknown},
			BestPlayerCount: entry.BestPlayerCount, BestPlayerPercentage: roundBestSlotPercentage(entry.BestPlayerPercentage),
			AverageScore: entry.AverageScore,
		})
	}
	return response
}

// ToSingleChartBestSlotStatsResponse は内部レート帯IDを公開ラベルへ変換します。
func ToSingleChartBestSlotStatsResponse(stats *entity.SingleChartBestSlotStats, bands []*ratingband.RatingBand) *SingleChartBestSlotStatsResponse {
	if stats == nil {
		return nil
	}
	labels := make(map[int]string, len(bands))
	for _, band := range bands {
		labels[band.ID] = band.Label
	}
	response := &SingleChartBestSlotStatsResponse{SongID: stats.SongID, Stats: make([]ChartBestSlotStatsByRatingBandDTO, 0, len(stats.Stats))}
	for _, stat := range stats.Stats {
		label, ok := labels[stat.RatingBandID]
		if !ok {
			slog.Warn("Rating band label not found for best-slot stats", "rating_band_id", stat.RatingBandID)
			label = "UNKNOWN"
		}
		response.Stats = append(response.Stats, ChartBestSlotStatsByRatingBandDTO{
			RatingBand: label, BestPlayerCount: stat.BestPlayerCount,
			EligiblePlayerCount: stat.EligiblePlayerCount, BestPlayerPercentage: roundBestSlotPercentagePointer(stat.BestPlayerPercentage),
		})
	}
	return response
}

func roundBestSlotPercentage(value float64) float64 {
	return math.Round(value*10000) / 10000
}

func roundBestSlotPercentagePointer(value *float64) *float64 {
	if value == nil {
		return nil
	}
	rounded := roundBestSlotPercentage(*value)
	return &rounded
}
