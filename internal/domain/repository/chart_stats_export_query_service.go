package repository

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
)

// ChartStatsExportRank は率の丸め方を利用側で選べるよう排他的件数を保持します。
type ChartStatsExportRank struct {
	AAAL int
	S    int
	SP   int
	SS   int
	SSP  int
	SSS  int
	SSSP int
	Max  int
}

// ChartStatsExportCombo は累積達成率を利用側で算出できるよう排他的件数を保持します。
type ChartStatsExportCombo struct {
	None int
	FC   int
	AJ   int
	AJC  int
}

// ChartStatsExportItem はDBの数値IDを公開境界へ持ち出さないため表示用IDを保持します。
type ChartStatsExportItem struct {
	SongDisplayID  string
	SongTitle      string
	Difficulty     string
	ChartConst     chartconstant.ChartConstant
	IsConstUnknown bool
	PlayerCount    int
	Rank           ChartStatsExportRank
	Combo          ChartStatsExportCombo
}

// WorldsendChartStatsExportItem は通常譜面の定数と混同しないよう星・属性を独立して保持します。
type WorldsendChartStatsExportItem struct {
	SongDisplayID string
	SongTitle     string
	LevelStar     *int
	Attribute     *string
	PlayerCount   int
	Rank          ChartStatsExportRank
	Combo         ChartStatsExportCombo
}

// ChartStatsExportSnapshot は通常譜面とWORLD'S ENDの空判定を個別に行えるよう分けて保持します。
type ChartStatsExportSnapshot struct {
	Charts          []ChartStatsExportItem
	WorldsendCharts []WorldsendChartStatsExportItem
}

// ChartStatsExportQueryService は公開統計JSONに必要な全譜面を一括取得します。
type ChartStatsExportQueryService interface {
	Get(ctx context.Context) (*ChartStatsExportSnapshot, error)
}
