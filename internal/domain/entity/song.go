package entity

import (
	"time"
)

// Song は楽曲エンティティ（集約ルート）を表します。
// Charts フィールドは常に初期化された状態（最低でも空スライス）でなければなりません。
// nil は許容されません。Song を構築する際は必ず Charts: []*Chart{} 以上の値を設定してください。
type Song struct {
	ID             int
	DisplayID      string
	Title          string
	Artist         string
	GenreID        *int
	BPM            *int
	ReleasedAt     *time.Time
	OfficialIdx    string
	Jacket         *string
	Charts         []*Chart
	MaxChartConst  float64
	IsMaxOPUnknown bool
	IsWorldsend    bool
	IsDeleted      bool
}
