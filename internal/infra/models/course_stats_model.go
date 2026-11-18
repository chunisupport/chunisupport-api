package models

import "github.com/chunisupport/chunisupport-api/internal/domain/entity"

// CourseStatsByRatingBandModel はコース×レーティング帯の統計の永続化モデルです。
type CourseStatsByRatingBandModel struct {
	CourseID     int      `db:"course_id"`
	RatingBandID int      `db:"rating_band_id"`
	ComboNone    int      `db:"combo_none"`
	ComboFC      int      `db:"combo_fc"`
	ComboAJ      int      `db:"combo_aj"`
	ClearFailed  int      `db:"clear_failed"`
	ClearClear   int      `db:"clear_clear"`
	AverageScore *float64 `db:"average_score"`
	MedianScore  *float64 `db:"median_score"`
	PlayerCount  int      `db:"player_count"`
}

// ToEntity は永続化モデルをドメインエンティティへ変換します。
func (m CourseStatsByRatingBandModel) ToEntity() *entity.CourseStatsByRatingBand {
	return &entity.CourseStatsByRatingBand{
		CourseID:     m.CourseID,
		RatingBandID: m.RatingBandID,
		Combo:        entity.CourseComboStats{None: m.ComboNone, FC: m.ComboFC, AJ: m.ComboAJ},
		Clear:        entity.CourseClearStats{Failed: m.ClearFailed, Clear: m.ClearClear},
		AverageScore: m.AverageScore,
		MedianScore:  m.MedianScore,
		PlayerCount:  m.PlayerCount,
	}
}

// CourseStatsByRatingBandModelFromEntity はドメインエンティティを永続化モデルへ変換します。
func CourseStatsByRatingBandModelFromEntity(stats *entity.CourseStatsByRatingBand) CourseStatsByRatingBandModel {
	return CourseStatsByRatingBandModel{
		CourseID:     stats.CourseID,
		RatingBandID: stats.RatingBandID,
		ComboNone:    stats.Combo.None,
		ComboFC:      stats.Combo.FC,
		ComboAJ:      stats.Combo.AJ,
		ClearFailed:  stats.Clear.Failed,
		ClearClear:   stats.Clear.Clear,
		AverageScore: stats.AverageScore,
		MedianScore:  stats.MedianScore,
		PlayerCount:  stats.PlayerCount,
	}
}
