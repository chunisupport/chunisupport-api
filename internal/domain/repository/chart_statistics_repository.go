package repository

import (
	"context"

	"github.com/Qman110101/chunisupport-api/internal/domain/entity"
)

// ChartStatisticsRepository は譜面統計情報の永続化を担当するリポジトリです。
// レーティング帯別のランク・ランプ人数を管理します。
type ChartStatisticsRepository interface {
	// FindByChartID は指定された譜面IDの全レーティング帯統計を取得します。
	// 統計データが存在しない場合は空のスライスを返します。
	FindByChartID(ctx context.Context, exec Executor, chartID int) ([]*entity.ChartStatistics, error)

	// FindByChartIDs は複数の譜面IDの統計を一括取得します。
	// 統計データが存在しない譜面は結果に含まれません。
	FindByChartIDs(ctx context.Context, exec Executor, chartIDs []int) ([]*entity.ChartStatistics, error)

	// Save は譜面統計を保存または更新します。
	// 既存レコードがあれば更新、なければ挿入します（UPSERT）。
	Save(ctx context.Context, exec Executor, stats *entity.ChartStatistics) error

	// BulkSave は複数の譜面統計を一括保存または更新します。
	// バッチ処理での効率的な更新に使用します。
	BulkSave(ctx context.Context, exec Executor, statsList []*entity.ChartStatistics) error

	// DeleteByChartID は指定された譜面の全統計データを削除します。
	DeleteByChartID(ctx context.Context, exec Executor, chartID int) error
}
