package repository

import (
	"context"
	"uuid"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// SongBatchJobRepository は楽曲バッチジョブ集約の保存と復元を担当します。
type SongBatchJobRepository interface {
	// Save はジョブを新規作成または更新します。
	Save(ctx context.Context, job *entity.SongBatchJob) error
	// FindByID はジョブを取得します。存在しない場合は ErrSongBatchJobNotFound を返します。
	FindByID(ctx context.Context, id uuid.UUID) (*entity.SongBatchJob, error)
	// FindRunning は実行中として記録されているジョブを返します。
	FindRunning(ctx context.Context) ([]*entity.SongBatchJob, error)
	// ListRecent は開始日時の新しい順に最大 limit 件のジョブを返します。
	ListRecent(ctx context.Context, limit int) ([]*entity.SongBatchJob, error)
}
