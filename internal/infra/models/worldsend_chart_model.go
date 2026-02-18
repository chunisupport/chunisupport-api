package models

import (
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/notes"
)

// WorldsendChartModel はデータベース用の WORLD'S END 譜面モデルです。
type WorldsendChartModel struct {
	ID        int          `db:"id"`
	SongID    int          `db:"song_id"`
	LevelStar *int         `db:"level_star"`
	Attribute *string      `db:"attribute"`
	Notes     *notes.Notes `db:"notes"`
}

func (m *WorldsendChartModel) ToEntity() *entity.WorldsendChart {
	return &entity.WorldsendChart{
		ID:        m.ID,
		SongID:    m.SongID,
		LevelStar: m.LevelStar,
		Attribute: m.Attribute,
		Notes:     m.Notes,
	}
}

func FromWorldsendChartEntity(e *entity.WorldsendChart) *WorldsendChartModel {
	return &WorldsendChartModel{
		ID:        e.ID,
		SongID:    e.SongID,
		LevelStar: e.LevelStar,
		Attribute: e.Attribute,
		Notes:     e.Notes,
	}
}
