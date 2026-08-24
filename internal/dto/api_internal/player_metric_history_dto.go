package api_internal

import "time"

// PlayerMetricHistoryResponse はプレイヤー公式指標のタイムラインです。
type PlayerMetricHistoryResponse struct {
	Entries []PlayerMetricHistoryEntry `json:"entries"`
}

// PlayerMetricHistoryEntry は公式RATING・公式OVER POWER・公式OP%の履歴1件です。
type PlayerMetricHistoryEntry struct {
	Rating           float64   `json:"rating"`
	Overpower        float64   `json:"overpower"`
	OverpowerPercent *float64  `json:"overpower_percent"`
	DataCollectedAt  time.Time `json:"data_collected_at"`
}
