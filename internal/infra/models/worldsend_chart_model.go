package models

import (
	"github.com/Qman110101/chunisupport-api/internal/domain/entity"
	"github.com/Qman110101/chunisupport-api/internal/domain/vo/notes"
)

// WorldsendChartModel はデータベース用の WORLD'S END 譜面モデルです。
type WorldsendChartModel struct {
	ID      int          `db:"id"`
	SongID  int          `db:"song_id"`
	WeStar  *int         `db:"we_star"`
	WeKanji *string      `db:"we_kanji"`
	Notes   *notes.Notes `db:"notes"`
}

// ToEntity は WorldsendChartModel を entity.WorldsendChart に変換します。
func (m *WorldsendChartModel) ToEntity() *entity.WorldsendChart {
	return &entity.WorldsendChart{
		ID:      m.ID,
		SongID:  m.SongID,
		WeStar:  m.WeStar,
		WeKanji: m.WeKanji,
		Notes:   m.Notes,
	}
}

// FromWorldsendChartEntity は entity.WorldsendChart を WorldsendChartModel に変換します。
func FromWorldsendChartEntity(e *entity.WorldsendChart) *WorldsendChartModel {
	return &WorldsendChartModel{
		ID:      e.ID,
		SongID:  e.SongID,
		WeStar:  e.WeStar,
		WeKanji: e.WeKanji,
		Notes:   e.Notes,
	}
}
