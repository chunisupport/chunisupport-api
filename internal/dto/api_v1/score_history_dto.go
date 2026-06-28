package api_v1

import "time"

// ScoreHistoryResponse は譜面単位のスコア履歴レスポンスです。
type ScoreHistoryResponse struct {
	Entries []ScoreHistoryEntry `json:"entries"`
}

// ScoreHistoryEntry はスコア履歴の1件です。
type ScoreHistoryEntry struct {
	Score     int       `json:"score"`
	ClearLamp *string   `json:"clear_lamp"`
	ComboLamp *string   `json:"combo_lamp"`
	FullChain *string   `json:"full_chain"`
	UpdatedAt time.Time `json:"updated_at"`
}
