package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"
	"uuid"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/info"
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

// Save は実行中のジョブを更新し、存在しなければ新規作成します。
// 新規作成時は保持上限（info.SongBatchJobHistoryLimit）を超えた古いジョブを削除します。
// ジョブの作成と終了記録は楽曲バッチのアドバイザリロック取得中に行うため、同一IDへの同時保存は発生しません。
// 終了済みの行は更新しないため、取り残されたジョブとして中断扱いにした行を、ロックを失った元のプロセスが上書きすることもありません
// （その場合は INSERT が主キー重複で失敗します）。
// 本番のDSNは clientFoundRows=true のため、値が変わらない UPDATE でもマッチした行数が返り、存在判定に使えます。
func (r *songBatchJobRepository) Save(ctx context.Context, job *entity.SongBatchJob) error {
	model := models.FromSongBatchJobEntity(job)
	result, err := r.db.ExecContext(ctx, `
UPDATE song_batch_jobs
SET status = ?, finished_at = ?, warning_count = ?, error_message = ?
WHERE id = ? AND status = ?
`, model.Status, model.FinishedAt, model.WarningCount, model.ErrorMessage, model.ID, string(entity.SongBatchJobStatusRunning))
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

	return r.insertAndTrim(ctx, model)
}

// insertAndTrim はジョブを新規作成し、保持上限を超えた古いジョブを同じトランザクションで削除します。
// 新規作成は楽曲バッチのロック取得中に取り残されたジョブを片付けた後で行うため、削除対象は終了済みのジョブだけです。
func (r *songBatchJobRepository) insertAndTrim(ctx context.Context, model *models.SongBatchJobModel) (err error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
			return
		}
		err = tx.Commit()
	}()

	if _, err = tx.ExecContext(ctx, `
INSERT INTO song_batch_jobs (
	id, mode, fill_missing_release_date, trigger_type, status,
	requested_by_user_id, started_at, finished_at, warning_count, error_message
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`, model.ID, model.Mode, model.FillMissingReleaseDate, model.TriggerType, model.Status,
		model.RequestedByUserID, model.StartedAt, model.FinishedAt, model.WarningCount, model.ErrorMessage); err != nil {
		return err
	}

	var cutoff time.Time
	err = tx.GetContext(ctx, &cutoff, `
SELECT started_at FROM song_batch_jobs
ORDER BY started_at DESC
LIMIT 1 OFFSET ?
`, info.SongBatchJobHistoryLimit-1)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `DELETE FROM song_batch_jobs WHERE started_at < ?`, cutoff)
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
