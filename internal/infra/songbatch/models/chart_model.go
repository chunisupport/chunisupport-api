package models

import (
	"github.com/chunisupport/chunisupport-api/internal/domain/songbatch/entity"
	"github.com/chunisupport/chunisupport-api/internal/utils"
)

// ChartModelForUpsert はUPSERT操作用の譜面モデルです
type ChartModelForUpsert struct {
	SongID         int     `db:"song_id"`
	DifficultyID   int     `db:"difficulty_id"`
	Const          float64 `db:"const"`
	IsConstUnknown int     `db:"is_const_unknown"`
}

// FromChartEntityForUpsert はChartエンティティからUPSERT用モデルを生成します
func FromChartEntityForUpsert(c *entity.Chart) *ChartModelForUpsert {
	return &ChartModelForUpsert{
		SongID:         c.SongID(),
		DifficultyID:   c.DifficultyID().Int(),
		Const:          c.Level().Float64(),
		IsConstUnknown: utils.BoolToInt(c.IsConstUnknown()),
	}
}
