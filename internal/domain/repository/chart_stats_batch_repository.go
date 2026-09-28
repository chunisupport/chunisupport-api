package repository

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/chartstatsbatch"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/ratingband"
)

// ChartStatsBatchSourceCallbacks は譜面統計バッチの集計元データを受け取るコールバックです。
// 全件をメモリに載せずに集計できるよう、リポジトリは1件ずつ呼び出します。
type ChartStatsBatchSourceCallbacks struct {
	// ChartRecord は通常譜面の記録を譜面ID順に受け取ります。
	ChartRecord func(chartstatsbatch.ChartRecord) error
	// WorldsendChartRecord は WORLD'S END 譜面の記録を譜面ID順に受け取ります。
	WorldsendChartRecord func(chartstatsbatch.ChartRecord) error
	// EligiblePlayer はベスト枠採用率の母数となるプレイヤーを受け取ります。
	EligiblePlayer func(chartstatsbatch.EligiblePlayer) error
	// BestSlotRecord はベスト枠に入っている譜面を受け取ります。
	BestSlotRecord func(chartstatsbatch.BestSlotRecord) error
	// BestSlotChart はベスト枠採用率の集計対象となる譜面IDを受け取ります。
	BestSlotChart func(chartID int) error
}

// ChartStatsBatchRepository は譜面統計バッチの集計元の読み取りと、統計テーブルの再構築を担当します。
type ChartStatsBatchRepository interface {
	// FindRatingBands はレーティング帯を表示順に返します。
	FindRatingBands(ctx context.Context) ([]*ratingband.RatingBand, error)
	// StreamSource は集計元データを単一の一貫したスナップショットから読み取り、コールバックへ渡します。
	StreamSource(ctx context.Context, callbacks ChartStatsBatchSourceCallbacks) error
	// ReplaceAll は統計テーブルの内容を snapshot へ単一トランザクションで入れ替えます。
	// 失敗した場合は既存の統計を維持します。
	ReplaceAll(ctx context.Context, snapshot *chartstatsbatch.Snapshot) error
}
