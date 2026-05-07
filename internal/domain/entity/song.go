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
	UpdatedAt      *time.Time
}

// NewSong は Charts を必ず非nilで初期化した Song を生成します。
func NewSong() *Song {
	return &Song{
		Charts: []*Chart{},
	}
}

// IsActive は楽曲が有効（削除されていない）かを判定します。
func (s *Song) IsActive() bool {
	return !s.IsDeleted
}

// HasDifficultyChart は指定された難易度IDの譜面を持つかを判定します。
func (s *Song) HasDifficultyChart(difficultyID int) bool {
	for _, chart := range s.Charts {
		if chart.DifficultyID == difficultyID {
			return true
		}
	}
	return false
}

// Delete は楽曲を論理削除します。
func (s *Song) Delete() {
	s.IsDeleted = true
}

// Restore は楽曲を復活させます。
func (s *Song) Restore() {
	s.IsDeleted = false
}
