package models

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/levelstar"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/notes"
)

// WorldsendChartModel はデータベース用の WORLD'S END 譜面モデルです。
type WorldsendChartModel struct {
	ID            int                  `db:"id"`
	SongID        int                  `db:"song_id"`
	LevelStar     *levelstar.LevelStar `db:"level_star"`
	Attribute     *string              `db:"attribute"`
	Notes         *notes.Notes         `db:"notes"`
	NotesDesigner *string              `db:"notes_designer"`
	UpdatedAt     *time.Time           `db:"updated_at"`
}

func (m *WorldsendChartModel) ToEntity() *entity.WorldsendChart {
	return &entity.WorldsendChart{
		ID:            m.ID,
		SongID:        m.SongID,
		LevelStar:     m.LevelStar,
		Attribute:     m.Attribute,
		Notes:         m.Notes,
		NotesDesigner: m.NotesDesigner,
		UpdatedAt:     m.UpdatedAt,
	}
}

func FromWorldsendChartEntity(e *entity.WorldsendChart) *WorldsendChartModel {
	return &WorldsendChartModel{
		ID:            e.ID,
		SongID:        e.SongID,
		LevelStar:     e.LevelStar,
		Attribute:     e.Attribute,
		Notes:         e.Notes,
		NotesDesigner: e.NotesDesigner,
		UpdatedAt:     e.UpdatedAt,
	}
}
