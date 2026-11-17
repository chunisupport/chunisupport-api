package dto

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/ratingband"
)

// CourseStatsResponse はコース統計APIのレスポンスです。
type CourseStatsResponse struct {
	CourseID string                        `json:"course_id"`
	Stats    []*CourseStatsByRatingBandDTO `json:"stats"`
}

// CourseStatsByRatingBandDTO はコースのレーティング帯別統計のDTOです。
type CourseStatsByRatingBandDTO struct {
	RatingBand   string              `json:"rating_band"`
	Combo        CourseComboStatsDTO `json:"combo"`
	Clear        CourseClearStatsDTO `json:"clear"`
	AverageScore *float64            `json:"average_score"`
	MedianScore  *float64            `json:"median_score"`
	PlayerCount  int                 `json:"player_count"`
}

// CourseComboStatsDTO はコースのコンボランプ別人数のDTOです。
type CourseComboStatsDTO struct {
	None int `json:"none"`
	FC   int `json:"fc"`
	AJ   int `json:"aj"`
}

// CourseClearStatsDTO はコースのクリア状態別人数のDTOです。
type CourseClearStatsDTO struct {
	Failed int `json:"failed"`
	Clear  int `json:"clear"`
}

// ToCourseStatsResponse は CourseStats を CourseStatsResponse に変換します。
func ToCourseStatsResponse(ctx context.Context, stats *entity.CourseStats, ratingBands []*ratingband.RatingBand) *CourseStatsResponse {
	if stats == nil {
		return nil
	}

	ratingBandLabels := make(map[int]string, len(ratingBands))
	for _, band := range ratingBands {
		ratingBandLabels[band.ID] = band.Label
	}

	statsDTO := make([]*CourseStatsByRatingBandDTO, 0, len(stats.Stats))
	for _, stat := range stats.Stats {
		label, ok := ratingBandLabels[stat.RatingBandID]
		if !ok {
			slog.WarnContext(ctx, "Rating band label not found", "rating_band_id", stat.RatingBandID)
			label = strconv.Itoa(stat.RatingBandID)
		}
		statsDTO = append(statsDTO, &CourseStatsByRatingBandDTO{
			RatingBand:   label,
			Combo:        CourseComboStatsDTO{None: stat.Combo.None, FC: stat.Combo.FC, AJ: stat.Combo.AJ},
			Clear:        CourseClearStatsDTO{Failed: stat.Clear.Failed, Clear: stat.Clear.Clear},
			AverageScore: stat.AverageScore,
			MedianScore:  stat.MedianScore,
			PlayerCount:  stat.PlayerCount,
		})
	}
	return &CourseStatsResponse{CourseID: stats.CourseDisplayID, Stats: statsDTO}
}
