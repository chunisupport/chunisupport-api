package entity

import "time"

type PlayerPlayLog struct {
	ChartID          *int
	WorldsendChartID *int
	DifficultyID     *int
	Jacket           *string
	Score            int
	ClearLampID      int
	ComboLampID      int
	FullChainID      int
	PlayedAt         time.Time
}

// PlayerPlayHistory は管理画面で表示するプレイ履歴の表示用データです。
// 譜面が現行マスタと照合できない場合、Title は nil になります。
type PlayerPlayHistory struct {
	Title      *string
	Difficulty *string
	Score      int
	ClearLamp  *string
	ComboLamp  *string
	FullChain  *string
	PlayedAt   time.Time
}

type PlayLogChartCandidate struct {
	ID           int
	Title        string
	Jacket       string
	DifficultyID int // WORLD'S END は通常難易度マスタに含まれないため0を使います。
}
