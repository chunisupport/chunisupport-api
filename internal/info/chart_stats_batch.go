package info

import "time"

// 譜面統計バッチ（chart-stats-batch）で使用する定数です。
const (
	// ChartStatsBatchLockName は chart-stats-batch の全起動経路（CLI・管理画面）で共有する MySQL アドバイザリロック名です。
	ChartStatsBatchLockName = "chunisupport:chart-stats-batch"
	// ChartStatsBatchInsertChunkSize は統計テーブルへの一括挿入で1文に含める行数です。
	// 譜面統計は1行23列あるため、BulkInsertChunkSize では MySQL のプレースホルダー上限（65,535）を超えます。
	ChartStatsBatchInsertChunkSize = 1000
	// ChartStatsBatchJobHistoryLimit は保持し、管理画面へ返す実行履歴の最大件数です。
	// 新しいジョブを記録した時点で、これを超えた古いジョブを削除します。
	ChartStatsBatchJobHistoryLimit = 50
	// ChartStatsBatchJobFinalizeTimeout は実行結果の記録とロック解放に使う猶予です。
	// 停止シグナルで実行がキャンセルされた後も、中断を記録できるよう独立したタイムアウトを使います。
	ChartStatsBatchJobFinalizeTimeout = 10 * time.Second
)
