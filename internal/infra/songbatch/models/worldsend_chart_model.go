package models

import (
	"github.com/chunisupport/chunisupport-api/internal/domain/songbatch/entity"
)

// WorldsEndChartModelForUpsert はUPSERT操作用のWORLD'S END譜面モデルです
type WorldsEndChartModelForUpsert struct {
	SongID    int     `db:"song_id"`
	LevelStar *int    `db:"level_star"`
	Attribute *string `db:"attribute"`
}

// FromWorldsEndChartEntityForUpsert はWorldsEndChartエンティティからUPSERT用モデルを生成します
func FromWorldsEndChartEntityForUpsert(c *entity.WorldsEndChart) *WorldsEndChartModelForUpsert {
	return &WorldsEndChartModelForUpsert{
		SongID:    c.SongID(),
		LevelStar: c.WeStar().IntPtr(),
		Attribute: c.WeKanji().StringPtr(),
	}
}
