package api_internal

import "time"

// PlayerPlayHistoryResponse はプレイ履歴取得レスポンスです。
type PlayerPlayHistoryResponse struct {
	Entries []PlayerPlayHistoryEntry `json:"entries"`
}

// PlayerPlayHistoryEntry はプレイ履歴の表示用エントリです。
type PlayerPlayHistoryEntry struct {
	Title      *string   `json:"title"`
	Difficulty *string   `json:"difficulty"`
	Score      int       `json:"score"`
	ClearLamp  *string   `json:"clear_lamp"`
	ComboLamp  *string   `json:"combo_lamp"`
	FullChain  *string   `json:"full_chain"`
	PlayedAt   time.Time `json:"played_at"`
}
