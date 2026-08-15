package entity

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/notes"
)

// Chart は譜面エンティティを表します
type Chart struct {
	ID             int
	SongID         int
	DifficultyID   int
	Const          chartconstant.ChartConstant
	IsConstUnknown bool
	Notes          *notes.Notes
	NotesDesigner  *string
	UpdatedAt      *time.Time
}

// ChangeConstant は譜面定数を変更し、定数不明状態を解除します。
// ChartConstant は生成時に検証済みのため、常に有効な値だけを受け取ります。
func (c *Chart) ChangeConstant(constant chartconstant.ChartConstant) {
	c.Const = constant
	c.IsConstUnknown = false
}
