package api_internal

import "time"

// PlayerMetricHistoryResponse はプレイヤー公式指標のタイムラインです。
type PlayerMetricHistoryResponse struct {
	Entries []PlayerMetricHistoryEntry `json:"entries"`
}

// PlayerMetricHistoryEntry は公式RATINGと公式OVER POWERの履歴1件です。
type PlayerMetricHistoryEntry struct {
	Rating          float64   `json:"rating"`
	Overpower       float64   `json:"overpower"`
	DataCollectedAt time.Time `json:"data_collected_at"`
}
