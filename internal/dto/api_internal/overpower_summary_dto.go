package api_internal

import "time"

// OverpowerSummaryItem は OVER POWER 集計の1項目を表します。
type OverpowerSummaryItem struct {
	CurrentOP   float64 `json:"current_op"`
	MaxOP       float64 `json:"max_op"`
	Percent     float64 `json:"percent"`
	TargetCount int     `json:"target_count"`
	PlayedCount int     `json:"played_count"`
}

// OverpowerSummaryResponse は OVER POWER 集計APIのレスポンスです。
type OverpowerSummaryResponse struct {
	UpdatedAt    time.Time                       `json:"updated_at"`
	Overall      OverpowerSummaryItem            `json:"overall"`
	Genres       map[string]OverpowerSummaryItem `json:"genres"`
	Difficulties map[string]OverpowerSummaryItem `json:"difficulties"`
	Levels       map[string]OverpowerSummaryItem `json:"levels"`
}
