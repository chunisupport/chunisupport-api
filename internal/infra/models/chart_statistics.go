package models

import (
	"time"

	"github.com/Qman110101/chunisupport-api/internal/domain/entity"
)

// ChartStatistics はデータベースの chart_statistics テーブルを表します。
type ChartStatistics struct {
	ChartID          int       `db:"chart_id"`
	RatingTier       int       `db:"rating_tier"`
	RankSCount       int       `db:"rank_s_count"`
	RankSPlusCount   int       `db:"rank_s_plus_count"`
	RankSSCount      int       `db:"rank_ss_count"`
	RankSSPlusCount  int       `db:"rank_ss_plus_count"`
	RankSSSCount     int       `db:"rank_sss_count"`
	RankSSSPlusCount int       `db:"rank_sss_plus_count"`
	LampAJCount      int       `db:"lamp_aj_count"`
	LampFCCount      int       `db:"lamp_fc_count"`
	LampOtherCount   int       `db:"lamp_other_count"`
	TotalCount       int       `db:"total_count"`
	UpdatedAt        time.Time `db:"updated_at"`
}

// ToEntity はデータベースモデルをドメインエンティティに変換します。
func (m *ChartStatistics) ToEntity() *entity.ChartStatistics {
	return &entity.ChartStatistics{
		ChartID:     m.ChartID,
		RatingTier:  m.RatingTier,
		RankS:       m.RankSCount,
		RankSPlus:   m.RankSPlusCount,
		RankSS:      m.RankSSCount,
		RankSSPlus:  m.RankSSPlusCount,
		RankSSS:     m.RankSSSCount,
		RankSSSPlus: m.RankSSSPlusCount,
		LampAJ:      m.LampAJCount,
		LampFC:      m.LampFCCount,
		LampOther:   m.LampOtherCount,
		TotalCount:  m.TotalCount,
		UpdatedAt:   m.UpdatedAt,
	}
}

// FromEntity はドメインエンティティをデータベースモデルに変換します。
func FromEntity(e *entity.ChartStatistics) *ChartStatistics {
	return &ChartStatistics{
		ChartID:          e.ChartID,
		RatingTier:       e.RatingTier,
		RankSCount:       e.RankS,
		RankSPlusCount:   e.RankSPlus,
		RankSSCount:      e.RankSS,
		RankSSPlusCount:  e.RankSSPlus,
		RankSSSCount:     e.RankSSS,
		RankSSSPlusCount: e.RankSSSPlus,
		LampAJCount:      e.LampAJ,
		LampFCCount:      e.LampFC,
		LampOtherCount:   e.LampOther,
		TotalCount:       e.TotalCount,
		UpdatedAt:        e.UpdatedAt,
	}
}
