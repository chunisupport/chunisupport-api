package entity

import (
	"time"
)

// Song は楽曲エンティティを表します
type Song struct {
	ID          int
	DisplayID   string
	Title       string
	Artist      string
	GenreID     *int
	BPM         *int
	ReleasedAt  *time.Time
	OfficialIdx string
	Jacket      *string
	IsWorldsend bool
	IsDeleted   bool
}
