package entity

import (
	"github.com/Qman110101/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/Qman110101/chunisupport-api/internal/domain/vo/notes"
)

// Chart は譜面エンティティを表します
type Chart struct {
	ID             int
	SongID         int
	DifficultyID   int
	Const          chartconstant.ChartConstant
	IsConstUnknown bool
	Notes          *notes.Notes
}
