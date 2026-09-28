// Package chartstatsbatch は譜面統計バッチの集計ルールを表します。
// プレイヤーの記録をベスト枠平均レーティング帯ごとに集計し、公開APIが参照する統計テーブルの内容を組み立てます。
package chartstatsbatch

import "github.com/chunisupport/chunisupport-api/internal/domain/entity"

// ChartRecord は集計対象プレイヤーの譜面別記録です。
// 通常譜面とWORLD'S END譜面で共通に使い、ChartID にはそれぞれの譜面IDを入れます。
type ChartRecord struct {
	ChartID           int
	BestAverageRating float64
	Score             int
	ClearLampID       int
	ComboLampID       int
}

// EligiblePlayer はベスト枠採用率の母数に含めるプレイヤーです。
type EligiblePlayer struct {
	BestAverageRating float64
}

// BestSlotRecord はプレイヤーのベスト枠に入っている譜面1件です。
type BestSlotRecord struct {
	ChartID           int
	BestAverageRating float64
}

// Snapshot は1回の集計で再構築する統計テーブル全体の内容です。
// 一部だけが更新された状態を公開しないよう、リポジトリはこの単位で丸ごと入れ替えます。
type Snapshot struct {
	ChartStats []*entity.ChartStatsByRatingBand
	// WorldsendChartStats の ChartID は WORLD'S END 譜面IDです。
	WorldsendChartStats []*entity.ChartStatsByRatingBand
	BestSlotStats       []*entity.ChartBestSlotStatsByRatingBand
}
