package entity

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/levelstar"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/notes"
)

// WorldsendChart は WORLD'S END 譜面エンティティを表します。
// WORLD'S END は1曲1譜面が保証されています。
type WorldsendChart struct {
	ID            int
	SongID        int
	LevelStar     *levelstar.LevelStar // WORLD'S END レベル（1～5）
	Attribute     *string              // WORLD'S END 属性（光、蔵、改、狂、etc.）
	Notes         *notes.Notes
	NotesDesigner *string
	UpdatedAt     *time.Time
}

// Clone は WORLD'S END 譜面を深く複製します。
func (c *WorldsendChart) Clone() *WorldsendChart {
	if c == nil {
		return nil
	}
	cloned := *c
	cloned.LevelStar = clonePointer(c.LevelStar)
	cloned.Attribute = clonePointer(c.Attribute)
	cloned.Notes = clonePointer(c.Notes)
	cloned.NotesDesigner = clonePointer(c.NotesDesigner)
	cloned.UpdatedAt = clonePointer(c.UpdatedAt)
	return &cloned
}
