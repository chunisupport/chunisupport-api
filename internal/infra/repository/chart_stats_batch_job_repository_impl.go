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

// chartStatsBatchJobSelect は要求者のユーザー名を JOIN して取得する共通の SELECT 句です。
const chartStatsBatchJobSelect = `
SELECT
	j.id, j.trigger_type, j.status,
	j.requested_by_user_id, u.username AS requested_by_username,
	j.started_at, j.finished_at, j.error_message
FROM chart_stats_batch_jobs j
LEFT JOIN users u ON u.id = j.requested_by_user_id
`

type chartStatsBatchJobRepository struct {
	db *sqlx.DB
}

// NewChartStatsBatchJobRepository は譜面統計バッチジョブリポジトリを生成します。
func NewChartStatsBatchJobRepository(db *sqlx.DB) domainrepo.ChartStatsBatchJobRepository {
	return &chartStatsBatchJobRepository{db: db}
}

// Save は実行中のジョブを更新し、存在しなければ新規作成します。
// 新規作成時は保持上限（info.ChartStatsBatchJobHistoryLimit）を超えた古いジョブを削除します。
// ジョブの作成と終了記録は譜面統計バッチのアドバイザリロック取得中に行うため、同一IDへの同時保存は発生しません。
// 終了済みの行は更新しないため、取り残されたジョブとして中断扱いにした行を、ロックを失った元のプロセスが上書きすることもありません
// （その場合は INSERT が主キー重複で失敗します）。
// 本番のDSNは clientFoundRows=true のため、値が変わらない UPDATE でもマッチした行数が返り、存在判定に使えます。
func (r *chartStatsBatchJobRepository) Save(ctx context.Context, job *entity.ChartStatsBatchJob) error {
	model := models.FromChartStatsBatchJobEntity(job)
	result, err := r.db.ExecContext(ctx, `
UPDATE chart_stats_batch_jobs
SET status = ?, finished_at = ?, error_message = ?
WHERE id = ? AND status = ?
`, model.Status, model.FinishedAt, model.ErrorMessage, model.ID, string(entity.ChartStatsBatchJobStatusRunning))
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
// 新規作成はロック取得中に取り残されたジョブを片付けた後で行うため、削除対象は終了済みのジョブだけです。
func (r *chartStatsBatchJobRepository) insertAndTrim(ctx context.Context, model *models.ChartStatsBatchJobModel) (err error) {
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
INSERT INTO chart_stats_batch_jobs (
	id, trigger_type, status, requested_by_user_id, started_at, finished_at, error_message
) VALUES (?, ?, ?, ?, ?, ?, ?)
`, model.ID, model.TriggerType, model.Status, model.RequestedByUserID, model.StartedAt, model.FinishedAt, model.ErrorMessage); err != nil {
		return err
	}

	var cutoff time.Time
	err = tx.GetContext(ctx, &cutoff, `
SELECT started_at FROM chart_stats_batch_jobs
ORDER BY started_at DESC
LIMIT 1 OFFSET ?
`, info.ChartStatsBatchJobHistoryLimit-1)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `DELETE FROM chart_stats_batch_jobs WHERE started_at < ?`, cutoff)
	return err
}

// FindByID はジョブを取得します。
func (r *chartStatsBatchJobRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.ChartStatsBatchJob, error) {
	var model models.ChartStatsBatchJobModel
	if err := r.db.GetContext(ctx, &model, chartStatsBatchJobSelect+`WHERE j.id = ?`, id[:]); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domainrepo.ErrChartStatsBatchJobNotFound
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
func (r *chartStatsBatchJobRepository) FindRunning(ctx context.Context) ([]*entity.ChartStatsBatchJob, error) {
	return r.selectJobs(ctx, chartStatsBatchJobSelect+`WHERE j.status = ?`, string(entity.ChartStatsBatchJobStatusRunning))
}

// ListRecent は開始日時の新しい順に最大 limit 件のジョブを返します。
func (r *chartStatsBatchJobRepository) ListRecent(ctx context.Context, limit int) ([]*entity.ChartStatsBatchJob, error) {
	return r.selectJobs(ctx, chartStatsBatchJobSelect+`ORDER BY j.started_at DESC, j.id DESC LIMIT ?`, limit)
}

func (r *chartStatsBatchJobRepository) selectJobs(ctx context.Context, query string, args ...any) ([]*entity.ChartStatsBatchJob, error) {
	var jobModels []models.ChartStatsBatchJobModel
	if err := r.db.SelectContext(ctx, &jobModels, query, args...); err != nil {
		return nil, err
	}
	jobs := make([]*entity.ChartStatsBatchJob, 0, len(jobModels))
	for i := range jobModels {
		job, err := jobModels[i].ToEntity()
		if err != nil {
			return nil, errors.Join(domainrepo.ErrRepositoryOperationFailed, err)
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

var _ domainrepo.ChartStatsBatchJobRepository = (*chartStatsBatchJobRepository)(nil)
