package api_internal

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

// CourseScoreHistoryResponse はコース単位のスコア履歴レスポンスです。
type CourseScoreHistoryResponse struct {
	Entries []CourseScoreHistoryEntry `json:"entries"`
}

// CourseScoreHistoryEntry はコーススコア履歴の1件です。
type CourseScoreHistoryEntry struct {
	Score     int       `json:"score"`
	IsClear   bool      `json:"is_clear"`
	ComboLamp *string   `json:"combo_lamp"`
	UpdatedAt time.Time `json:"updated_at"`
}
