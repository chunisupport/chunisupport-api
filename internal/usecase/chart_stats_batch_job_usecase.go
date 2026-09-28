package usecase

import (
	"context"
	"errors"
	"log/slog"
	"sync"
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
// どちらの経路も同じアドバイザリロックを取得するため、同時に実行される譜面統計バッチは常に1つです。
type ChartStatsBatchJobUsecase struct {
	backgroundCtx context.Context
	lockProvider  repository.BatchLockProvider
	jobRepo       repository.ChartStatsBatchJobRepository
	runner        ChartStatsBatchRunner
	now           func() time.Time

	// mu は受付停止の判定と WaitGroup への登録を不可分にし、Wait 中の登録を防ぎます。
	mu     sync.Mutex
	closed bool
	wg     sync.WaitGroup
}

// NewChartStatsBatchJobUsecase は ChartStatsBatchJobUsecase を生成します。
// backgroundCtx は管理画面から起動したジョブの寿命で、サーバー停止時にキャンセルされるものを渡します。
func NewChartStatsBatchJobUsecase(
	backgroundCtx context.Context,
	lockProvider repository.BatchLockProvider,
	jobRepo repository.ChartStatsBatchJobRepository,
	runner ChartStatsBatchRunner,
) *ChartStatsBatchJobUsecase {
	return &ChartStatsBatchJobUsecase{
		backgroundCtx: backgroundCtx,
		lockProvider:  lockProvider,
		jobRepo:       jobRepo,
		runner:        runner,
		now:           time.Now,
	}
}

// RunFromCLI は CLI から譜面統計バッチを同期実行します。
// 別の譜面統計バッチが実行中の場合は実行せず acquired=false を返します。
func (u *ChartStatsBatchJobUsecase) RunFromCLI(ctx context.Context) (acquired bool, err error) {
	lock, job, acquired, err := u.begin(ctx, func(id uuid.UUID, startedAt time.Time) *entity.ChartStatsBatchJob {
		return entity.StartChartStatsBatchJobFromCLI(id, startedAt)
	})
	if err != nil || !acquired {
		return acquired, err
	}
	return true, u.run(ctx, lock, job)
}

// StartFromAdmin は管理画面からの実行要求を受け付け、バックグラウンドで譜面統計バッチを実行します。
// 全記録の集計には時間がかかるため、ジョブを記録した時点で呼び出し元へ返し、結果は履歴から確認させます。
func (u *ChartStatsBatchJobUsecase) StartFromAdmin(ctx context.Context, requester entity.ChartStatsBatchJobRequester) (*entity.ChartStatsBatchJob, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	// 停止シグナル後に受け付けても即座に中断されるだけのため、ロックを取る前に拒否する
	if u.closed || u.backgroundCtx.Err() != nil {
		return nil, ErrChartStatsBatchUnavailable
	}

	lock, job, acquired, err := u.begin(ctx, func(id uuid.UUID, startedAt time.Time) *entity.ChartStatsBatchJob {
		return entity.StartChartStatsBatchJobFromAdmin(id, requester, startedAt)
	})
	if err != nil {
		return nil, err
	}
	if !acquired {
		return nil, ErrChartStatsBatchAlreadyRunning
	}

	// バックグラウンド処理がジョブを更新するため、呼び出し元には開始時点の複製を返す
	started := *job
	u.wg.Go(func() {
		_ = u.run(u.backgroundCtx, lock, job)
	})
	return &started, nil
}

// Wait は新しいジョブの受付を停止し、バックグラウンドで実行中のジョブが結果を記録し終えるまで待ちます。
// サーバー停止時、DB接続を閉じる前に呼び出します。
func (u *ChartStatsBatchJobUsecase) Wait() {
	u.mu.Lock()
	u.closed = true
	u.mu.Unlock()
	u.wg.Wait()
}

// List は直近の実行履歴を開始日時の新しい順に返します。
func (u *ChartStatsBatchJobUsecase) List(ctx context.Context) ([]*entity.ChartStatsBatchJob, error) {
	return u.jobRepo.ListRecent(ctx, info.ChartStatsBatchJobHistoryLimit)
}

// Get は指定したジョブを返します。
func (u *ChartStatsBatchJobUsecase) Get(ctx context.Context, id string) (*entity.ChartStatsBatchJob, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return nil, ErrInvalidChartStatsBatchJobID
	}
	return u.jobRepo.FindByID(ctx, parsed)
}

// begin はロックを取得し、取り残されたジョブを片付けてから新しいジョブを記録します。
// 失敗した場合はロックを解放して返します。
func (u *ChartStatsBatchJobUsecase) begin(
	ctx context.Context,
	newJob func(id uuid.UUID, startedAt time.Time) *entity.ChartStatsBatchJob,
) (repository.BatchLock, *entity.ChartStatsBatchJob, bool, error) {
	lock, acquired, err := u.lockProvider.TryAcquire(ctx, info.ChartStatsBatchLockName)
	if err != nil || !acquired {
		return nil, nil, acquired, err
	}

	if err := u.interruptOrphanedJobs(ctx); err != nil {
		u.releaseLockAfterFailure(ctx, lock)
		return nil, nil, false, err
	}
	job := newJob(uuid.NewV4(), u.now())
	if err := u.jobRepo.Save(ctx, job); err != nil {
		u.releaseLockAfterFailure(ctx, lock)
		return nil, nil, false, err
	}
	return lock, job, true, nil
}

// interruptOrphanedJobs はプロセス停止などで終了を記録できなかったジョブを中断扱いにします。
// ロックを取得できた時点で他に実行中の譜面統計バッチは存在しないため、RUNNING のまま残った行はすべて取り残されたものです。
// 取得のたびに片付けるため対象は通常0件で、多くても数件に限られます。
func (u *ChartStatsBatchJobUsecase) interruptOrphanedJobs(ctx context.Context) error {
	orphans, err := u.jobRepo.FindRunning(ctx)
	if err != nil {
		return err
	}
	for _, orphan := range orphans {
		if err := orphan.Interrupt(u.now()); err != nil {
			return err
		}
		if err := u.jobRepo.Save(ctx, orphan); err != nil {
			return err
		}
		slog.Warn("取り残された譜面統計バッチジョブを中断扱いにしました", "job_id", orphan.ID())
	}
	return nil
}

// run は譜面統計バッチを実行して結果を記録し、ロックを解放します。
func (u *ChartStatsBatchJobUsecase) run(ctx context.Context, lock repository.BatchLock, job *entity.ChartStatsBatchJob) error {
	logger := slog.With("job_id", job.ID(), "trigger", job.Trigger())
	logger.Info("譜面統計バッチを開始します")

	result, execErr := u.runner.Execute(ctx)

	// 停止シグナルでキャンセルされた後も結果を記録できるよう、実行とは独立した期限を使う
	finalizeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), info.ChartStatsBatchJobFinalizeTimeout)
	defer cancel()
	defer u.releaseLock(finalizeCtx, lock)

	finishedAt := u.now()
	var finishErr error
	switch {
	case execErr == nil:
		finishErr = job.Complete(finishedAt)
		logger.Info("譜面統計バッチが完了しました",
			"elapsed", finishedAt.Sub(job.StartedAt()).String(),
			"chart_records", result.ChartRecordCount,
			"chart_stats", result.ChartStatsCount,
			"worldsend_chart_records", result.WorldsendChartRecordCount,
			"worldsend_chart_stats", result.WorldsendChartStatsCount,
			"best_slot_eligible_players", result.EligiblePlayerCount,
			"best_slot_records", result.BestSlotRecordCount,
			"best_slot_stats", result.BestSlotStatsCount,
		)
	case ctx.Err() != nil:
		finishErr = job.Interrupt(finishedAt)
		logger.Warn("譜面統計バッチが中断されました", "error", execErr)
	default:
		finishErr = job.Fail(execErr.Error(), finishedAt)
		logger.Error("譜面統計バッチに失敗しました", "error", execErr)
	}
	if finishErr != nil {
		logger.Error("譜面統計バッチの結果を反映できませんでした", "error", finishErr)
	} else if err := u.jobRepo.Save(finalizeCtx, job); err != nil {
		logger.Error("譜面統計バッチの結果を記録できませんでした", "error", err)
	}
	return execErr
}

// releaseLockAfterFailure はジョブ開始前の失敗時にロックを解放します。
// リクエストのキャンセルに関係なく解放できるよう、期限だけを持つ context を使います。
func (u *ChartStatsBatchJobUsecase) releaseLockAfterFailure(ctx context.Context, lock repository.BatchLock) {
	releaseCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), info.ChartStatsBatchJobFinalizeTimeout)
	defer cancel()
	u.releaseLock(releaseCtx, lock)
}

// releaseLock はロックを解放します。ctx には期限付きの context を渡します。
func (u *ChartStatsBatchJobUsecase) releaseLock(ctx context.Context, lock repository.BatchLock) {
	if err := lock.Release(ctx); err != nil {
		slog.Error("譜面統計バッチのロック解放に失敗しました", "error", err)
	}
}
