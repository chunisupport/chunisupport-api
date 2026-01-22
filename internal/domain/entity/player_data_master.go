package entity

import (
	"time"

	"github.com/Qman110101/chunisupport-api/internal/domain/vo/notes"
)

// PlayerDataSong はプレイヤーデータ登録に必要な楽曲マスタ情報です。
type PlayerDataSong struct {
	ID          int
	DisplayID   string
	Title       string
	Artist      string
	GenreID     *int
	BPM         *int
	ReleasedAt  *time.Time
	OfficialIdx string
	Jacket      *string
	IsDeleted   bool
}

// PlayerDataChart はプレイヤーデータ登録に必要な譜面マスタ情報です。
type PlayerDataChart struct {
	ID             int
	SongID         int
	DifficultyID   int
	Const          float64
	IsConstUnknown bool
	Notes          *notes.Notes
}

// PlayerDataWorldsendChart はプレイヤーデータ登録に必要なWORLD'S END譜面情報です。
type PlayerDataWorldsendChart struct {
	ID     int
	SongID int
}
