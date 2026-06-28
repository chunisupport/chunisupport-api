package entity

import "time"

// ScoreHistoryEntry は譜面ごとの過去または現行のベスト状態を表します。
type ScoreHistoryEntry struct {
	Score       int
	ClearLampID int
	ComboLampID int
	FullChainID int
	UpdatedAt   time.Time
}

// SupportsScoreHistory は履歴保存・参照の対象難易度かを判定します。
func SupportsScoreHistory(difficulty string) bool {
	switch difficulty {
	case "EXPERT", "MASTER", "ULTIMA":
		return true
	default:
		return false
	}
}
