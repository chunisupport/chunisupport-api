package repository

import (
	"context"
)

// FriendChartRankingChart はフレンドランキング対象の譜面情報です（通常 / WORLD'S END）。
// 共通項目に加え、譜面IDと難易度IDを持ちます。
type FriendChartRankingChart struct {
	ChartRankingChart
	ChartID         int
	ChartDifficulty int
}

// FriendChartRankingRecord は譜面単位フレンドランキングの1件です。
// 共通項目に加え、自分の判定に使うユーザーIDを持ちます。
type FriendChartRankingRecord struct {
	ChartRankingRecord
	UserID int
}

// FriendChartRankingQueryService は譜面単位フレンドランキングの読み取りを扱います。
type FriendChartRankingQueryService interface {
	FindChart(ctx context.Context, exec Executor, displayID string, difficulty string) (*FriendChartRankingChart, error)
	FindWorldsendChart(ctx context.Context, exec Executor, displayID string) (*FriendChartRankingChart, error)
	ListRecords(ctx context.Context, exec Executor, userID int, chartID int) ([]*FriendChartRankingRecord, error)
	ListWorldsendRecords(ctx context.Context, exec Executor, userID int, worldsendChartID int) ([]*FriendChartRankingRecord, error)
}
