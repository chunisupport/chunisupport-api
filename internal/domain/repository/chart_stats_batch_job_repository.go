package repository

import (
	"context"
	"uuid"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// ChartStatsBatchJobRepository は譜面統計バッチジョブ集約の保存と復元を担当します。
type ChartStatsBatchJobRepository interface {
	// Save はジョブを新規作成または更新します。
	Save(ctx context.Context, job *entity.ChartStatsBatchJob) error
	// FindByID はジョブを取得します。存在しない場合は ErrChartStatsBatchJobNotFound を返します。
	FindByID(ctx context.Context, id uuid.UUID) (*entity.ChartStatsBatchJob, error)
	// FindRunning は実行中として記録されているジョブを返します。
	FindRunning(ctx context.Context) ([]*entity.ChartStatsBatchJob, error)
	// ListRecent は開始日時の新しい順に最大 limit 件のジョブを返します。
	ListRecent(ctx context.Context, limit int) ([]*entity.ChartStatsBatchJob, error)
}
