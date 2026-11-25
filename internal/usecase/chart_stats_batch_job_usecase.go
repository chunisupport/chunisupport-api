package usecase

import (
	"context"
	"errors"
	"log/slog"
	"time"
	"uuid"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/info"
)

var (
	// ErrChartStatsBatchAlreadyRunning は別の譜面統計バッチが実行中のため開始できないことを表します。
	ErrChartStatsBatchAlreadyRunning = errors.New("chart stats batch is already running")
	// ErrInvalidChartStatsBatchJobID はジョブIDの形式が不正であることを表します。
	ErrInvalidChartStatsBatchJobID = errors.New("invalid chart stats batch job id")
	// ErrChartStatsBatchUnavailable はサーバー停止処理中のため新しいジョブを受け付けられないことを表します。
	ErrChartStatsBatchUnavailable = errors.New("chart stats batch is unavailable")
)

// ChartStatsBatchRunner は譜面統計バッチ1回分を実行します。
type ChartStatsBatchRunner interface {
	Execute(ctx context.Context) (ChartStatsBatchResult, error)
}

// ChartStatsBatchJobUsecase は譜面統計バッチの実行をジョブとして記録し、CLI と管理画面の両方から起動できるようにします。
// ジョブのライフサイクル（ロック・記録・受付停止）は batchJobLifecycle に任せ、譜面統計バッチ固有の実行内容だけを持ちます。
type ChartStatsBatchJobUsecase struct {
	*batchJobLifecycle[*entity.ChartStatsBatchJob, ChartStatsBatchResult]

	runner           ChartStatsBatchRunner
	scorePercentiles repository.ScorePercentileProvider
}

// NewChartStatsBatchJobUsecase は ChartStatsBatchJobUsecase を生成します。
// backgroundCtx は管理画面から起動したジョブの寿命で、サーバー停止時にキャンセルされるものを渡します。
// scorePercentiles は統計更新後に再構築するプロセス内キャッシュで、キャッシュを持たない CLI では nil を渡します。
func NewChartStatsBatchJobUsecase(
	backgroundCtx context.Context,
	lockProvider repository.BatchLockProvider,
	jobRepo repository.ChartStatsBatchJobRepository,
	runner ChartStatsBatchRunner,
	scorePercentiles repository.ScorePercentileProvider,
) *ChartStatsBatchJobUsecase {
	u := &ChartStatsBatchJobUsecase{runner: runner, scorePercentiles: scorePercentiles}
	u.batchJobLifecycle = newBatchJobLifecycle(backgroundCtx, lockProvider, jobRepo, batchJobSpec[*entity.ChartStatsBatchJob, ChartStatsBatchResult]{
		label:             "譜面統計バッチ",
		lockName:          info.ChartStatsBatchLockName,
		historyLimit:      info.ChartStatsBatchJobHistoryLimit,
		finalizeTimeout:   info.ChartStatsBatchJobFinalizeTimeout,
		errAlreadyRunning: ErrChartStatsBatchAlreadyRunning,
		errInvalidJobID:   ErrInvalidChartStatsBatchJobID,
		errUnavailable:    ErrChartStatsBatchUnavailable,
		logAttrs: func(job *entity.ChartStatsBatchJob) []any {
			return []any{"job_id", job.ID(), "trigger", job.Trigger()}
		},
		execute: u.execute,
		interruptOrphan: func(job *entity.ChartStatsBatchJob, finishedAt time.Time) error {
			return job.Interrupt("", finishedAt)
		},
		complete: func(job *entity.ChartStatsBatchJob, _ ChartStatsBatchResult, finishedAt time.Time) error {
			return job.Complete(finishedAt)
		},
		// キャンセルと同時に別のエラーで失敗した可能性もあるため、エラー内容を中断の理由として残す
		interrupt: func(job *entity.ChartStatsBatchJob, _ ChartStatsBatchResult, execErr error, finishedAt time.Time) error {
			return job.Interrupt(execErr.Error(), finishedAt)
		},
		fail: func(job *entity.ChartStatsBatchJob, _ ChartStatsBatchResult, execErr error, finishedAt time.Time) error {
			return job.Fail(execErr.Error(), finishedAt)
		},
		completedLogAttrs: func(job *entity.ChartStatsBatchJob, result ChartStatsBatchResult, finishedAt time.Time) []any {
			return []any{
				"elapsed", finishedAt.Sub(job.StartedAt()).String(),
				"chart_records", result.ChartRecordCount,
				"chart_stats", result.ChartStatsCount,
				"worldsend_chart_records", result.WorldsendChartRecordCount,
				"worldsend_chart_stats", result.WorldsendChartStatsCount,
				"best_slot_eligible_players", result.EligiblePlayerCount,
				"best_slot_records", result.BestSlotRecordCount,
				"best_slot_stats", result.BestSlotStatsCount,
				"course_records", result.CourseRecordCount,
				"course_stats", result.CourseStatsCount,
			}
		},
	})
	return u
}

// RunFromCLI は CLI から譜面統計バッチを同期実行します。
// 別の譜面統計バッチが実行中の場合は実行せず acquired=false を返します。
func (u *ChartStatsBatchJobUsecase) RunFromCLI(ctx context.Context) (acquired bool, err error) {
	return u.runFromCLI(ctx, func(id uuid.UUID, startedAt time.Time) *entity.ChartStatsBatchJob {
		return entity.StartChartStatsBatchJobFromCLI(id, startedAt)
	})
}

// StartFromAdmin は管理画面からの実行要求を受け付け、バックグラウンドで譜面統計バッチを実行します。
// 全記録の集計には時間がかかるため、ジョブを記録した時点で呼び出し元へ返し、結果は履歴から確認させます。
func (u *ChartStatsBatchJobUsecase) StartFromAdmin(ctx context.Context, requester entity.ChartStatsBatchJobRequester) (*entity.ChartStatsBatchJob, error) {
	return u.startFromAdmin(ctx,
		func(id uuid.UUID, startedAt time.Time) *entity.ChartStatsBatchJob {
			return entity.StartChartStatsBatchJobFromAdmin(id, requester, startedAt)
		},
		// バックグラウンド処理がジョブを更新するため、呼び出し元には開始時点の複製を返す
		func(job *entity.ChartStatsBatchJob) *entity.ChartStatsBatchJob {
			started := *job
			return &started
		},
	)
}

// execute は譜面統計バッチを実行し、成功した場合はプロセス内の上位%キャッシュを更新します。
func (u *ChartStatsBatchJobUsecase) execute(ctx context.Context, _ *entity.ChartStatsBatchJob, logger *slog.Logger) (ChartStatsBatchResult, error) {
	logger.Info("譜面統計バッチを開始します")
	result, err := u.runner.Execute(ctx)
	// 管理画面がジョブの完了を検知した時点で上位%が反映済みになるよう、完了の記録より先にキャッシュを更新する。
	// 再構築に失敗してもDB更新は成功しているため、ジョブは成功として記録し、失敗はログと定期確認での再試行に任せる。
	if err == nil && u.scorePercentiles != nil {
		refreshCtx, cancel := context.WithTimeout(ctx, info.ScorePercentileRefreshTimeout)
		refreshErr := u.scorePercentiles.Refresh(refreshCtx)
		cancel()
		if refreshErr != nil && !errors.Is(refreshErr, context.Canceled) {
			logger.ErrorContext(ctx, "failed to refresh score percentiles after chart stats batch", "error", refreshErr)
		}
	}
	return result, err
}
