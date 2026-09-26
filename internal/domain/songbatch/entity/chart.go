package entity

import (
	"github.com/chunisupport/chunisupport-api/internal/domain/songbatch/difficulty"
	vo "github.com/chunisupport/chunisupport-api/internal/domain/songbatch/valueobject"
)

// Chart は譜面を表すドメインエンティティ
type Chart struct {
	songID         int
	difficultyID   difficulty.ID
	level          vo.Level
	isConstUnknown bool
	notes          *int
}

// NewChart は新しいChartエンティティを生成します
func NewChart(songID int, difficultyID difficulty.ID, level vo.Level, isConstUnknown bool) *Chart {
	return &Chart{
		songID:         songID,
		difficultyID:   difficultyID,
		level:          level,
		isConstUnknown: isConstUnknown,
	}
}

// SongID は楽曲IDを返します
func (c *Chart) SongID() int { return c.songID }

// DifficultyID は難易度IDを返します
func (c *Chart) DifficultyID() difficulty.ID { return c.difficultyID }

// Level はレベル（定数）を返します
func (c *Chart) Level() vo.Level { return c.level }

// IsConstUnknown は定数が不明かどうかを返します
func (c *Chart) IsConstUnknown() bool { return c.isConstUnknown }

// Notes はノーツ数を返します
func (c *Chart) Notes() *int { return c.notes }

// SetNotes はノーツ数を設定します
func (c *Chart) SetNotes(notes int) {
	c.notes = &notes
}
