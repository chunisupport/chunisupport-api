package repository

import (
	"context"
	"database/sql"
	"errors"
	"uuid"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/infra/models"
	"github.com/jmoiron/sqlx"
)

// songBatchJobSelect は要求者のユーザー名を JOIN して取得する共通の SELECT 句です。
const songBatchJobSelect = `
SELECT
	j.id, j.mode, j.fill_missing_release_date, j.trigger_type, j.status,
	j.requested_by_user_id, u.username AS requested_by_username,
	j.started_at, j.finished_at, j.warning_count, j.error_message
FROM song_batch_jobs j
LEFT JOIN users u ON u.id = j.requested_by_user_id
`

type songBatchJobRepository struct {
	db *sqlx.DB
}

// NewSongBatchJobRepository は楽曲バッチジョブリポジトリを生成します。
func NewSongBatchJobRepository(db *sqlx.DB) domainrepo.SongBatchJobRepository {
	return &songBatchJobRepository{db: db}
}

// Save はジョブを更新し、存在しなければ新規作成します。
// ジョブの作成と終了記録は楽曲バッチのアドバイザリロック取得中に行うため、同一IDへの同時保存は発生しません。
func (r *songBatchJobRepository) Save(ctx context.Context, job *entity.SongBatchJob) error {
	model := models.FromSongBatchJobEntity(job)
	result, err := r.db.ExecContext(ctx, `
UPDATE song_batch_jobs
SET status = ?, finished_at = ?, warning_count = ?, error_message = ?
WHERE id = ?
`, model.Status, model.FinishedAt, model.WarningCount, model.ErrorMessage, model.ID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected > 0 {
		return nil
	}

	_, err = r.db.ExecContext(ctx, `
INSERT INTO song_batch_jobs (
	id, mode, fill_missing_release_date, trigger_type, status,
	requested_by_user_id, started_at, finished_at, warning_count, error_message
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`, model.ID, model.Mode, model.FillMissingReleaseDate, model.TriggerType, model.Status,
		model.RequestedByUserID, model.StartedAt, model.FinishedAt, model.WarningCount, model.ErrorMessage)
	return err
}

// FindByID はジョブを取得します。
func (r *songBatchJobRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.SongBatchJob, error) {
	var model models.SongBatchJobModel
	if err := r.db.GetContext(ctx, &model, songBatchJobSelect+`WHERE j.id = ?`, id[:]); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domainrepo.ErrSongBatchJobNotFound
		}
		return nil, err
	}
	job, err := model.ToEntity()
	if err != nil {
		return nil, errors.Join(domainrepo.ErrRepositoryOperationFailed, err)
	}
	return job, nil
}

// FindRunning は実行中として記録されているジョブを返します。
func (r *songBatchJobRepository) FindRunning(ctx context.Context) ([]*entity.SongBatchJob, error) {
	return r.selectJobs(ctx, songBatchJobSelect+`WHERE j.status = ?`, string(entity.SongBatchJobStatusRunning))
}

// ListRecent は開始日時の新しい順に最大 limit 件のジョブを返します。
func (r *songBatchJobRepository) ListRecent(ctx context.Context, limit int) ([]*entity.SongBatchJob, error) {
	return r.selectJobs(ctx, songBatchJobSelect+`ORDER BY j.started_at DESC, j.id DESC LIMIT ?`, limit)
}

func (r *songBatchJobRepository) selectJobs(ctx context.Context, query string, args ...any) ([]*entity.SongBatchJob, error) {
	var jobModels []models.SongBatchJobModel
	if err := r.db.SelectContext(ctx, &jobModels, query, args...); err != nil {
		return nil, err
	}
	jobs := make([]*entity.SongBatchJob, 0, len(jobModels))
	for i := range jobModels {
		job, err := jobModels[i].ToEntity()
		if err != nil {
			return nil, errors.Join(domainrepo.ErrRepositoryOperationFailed, err)
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

var _ domainrepo.SongBatchJobRepository = (*songBatchJobRepository)(nil)
