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
	"github.com/chunisupport/chunisupport-api/internal/domain/songbatch"
	"github.com/chunisupport/chunisupport-api/internal/info"
)

var (
	// ErrSongBatchAlreadyRunning は別の楽曲バッチが実行中のため開始できないことを表します。
	ErrSongBatchAlreadyRunning = errors.New("song batch is already running")
	// ErrInvalidSongBatchJobID はジョブIDの形式が不正であることを表します。
	ErrInvalidSongBatchJobID = errors.New("invalid song batch job id")
)

// SongBatchRunner は楽曲バッチ1回分を実行します。
type SongBatchRunner interface {
	Execute(ctx context.Context, req songbatch.RunRequest) (SongBatchResult, error)
}

// SongBatchJobUsecase は楽曲バッチの実行をジョブとして記録し、CLI と管理画面の両方から起動できるようにします。
// どちらの経路も同じアドバイザリロックを取得するため、同時に実行される楽曲バッチは常に1つです。
type SongBatchJobUsecase struct {
	backgroundCtx context.Context
	lockProvider  repository.BatchLockProvider
	jobRepo       repository.SongBatchJobRepository
	runner        SongBatchRunner
	denominator   repository.OverpowerDenominatorProvider
	now           func() time.Time
	wg            sync.WaitGroup
}

// NewSongBatchJobUsecase は SongBatchJobUsecase を生成します。
// backgroundCtx は管理画面から起動したジョブの寿命で、サーバー停止時にキャンセルされるものを渡します。
// denominator は同期後に無効化するプロセス内キャッシュで、キャッシュを持たない CLI では nil を渡します。
func NewSongBatchJobUsecase(
	backgroundCtx context.Context,
	lockProvider repository.BatchLockProvider,
	jobRepo repository.SongBatchJobRepository,
	runner SongBatchRunner,
	denominator repository.OverpowerDenominatorProvider,
) *SongBatchJobUsecase {
	return &SongBatchJobUsecase{
		backgroundCtx: backgroundCtx,
		lockProvider:  lockProvider,
		jobRepo:       jobRepo,
		runner:        runner,
		denominator:   denominator,
		now:           time.Now,
	}
}

// RunFromCLI は CLI から楽曲バッチを同期実行します。
// 別の楽曲バッチが実行中の場合は実行せず acquired=false を返し、判断は呼び出し元（実行モード）に委ねます。
func (u *SongBatchJobUsecase) RunFromCLI(ctx context.Context, req songbatch.RunRequest) (acquired bool, err error) {
	lock, job, acquired, err := u.begin(ctx, func(id uuid.UUID, startedAt time.Time) *entity.SongBatchJob {
		return entity.StartSongBatchJobFromCLI(id, req, startedAt)
	})
	if err != nil || !acquired {
		return acquired, err
	}
	return true, u.run(ctx, lock, job)
}

// StartFromAdmin は管理画面からの実行要求を受け付け、バックグラウンドで楽曲バッチを実行します。
// 処理には数分かかるため、ジョブを記録した時点で呼び出し元へ返し、結果は履歴から確認させます。
func (u *SongBatchJobUsecase) StartFromAdmin(ctx context.Context, requester entity.SongBatchJobRequester, req songbatch.RunRequest) (*entity.SongBatchJob, error) {
	lock, job, acquired, err := u.begin(ctx, func(id uuid.UUID, startedAt time.Time) *entity.SongBatchJob {
		return entity.StartSongBatchJobFromAdmin(id, req, requester, startedAt)
	})
	if err != nil {
		return nil, err
	}
	if !acquired {
		return nil, ErrSongBatchAlreadyRunning
	}

	// バックグラウンド処理がジョブを更新するため、呼び出し元には開始時点の複製を返す
	started := *job
	u.wg.Go(func() {
		_ = u.run(u.backgroundCtx, lock, job)
	})
	return &started, nil
}

// Wait はバックグラウンドで実行中のジョブが結果を記録し終えるまで待ちます。
// サーバー停止時、DB接続を閉じる前に呼び出します。
func (u *SongBatchJobUsecase) Wait() {
	u.wg.Wait()
}

// List は直近の実行履歴を開始日時の新しい順に返します。
func (u *SongBatchJobUsecase) List(ctx context.Context) ([]*entity.SongBatchJob, error) {
	return u.jobRepo.ListRecent(ctx, info.SongBatchJobHistoryLimit)
}

// Get は指定したジョブを返します。
func (u *SongBatchJobUsecase) Get(ctx context.Context, id string) (*entity.SongBatchJob, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return nil, ErrInvalidSongBatchJobID
	}
	return u.jobRepo.FindByID(ctx, parsed)
}

// begin はロックを取得し、取り残されたジョブを片付けてから新しいジョブを記録します。
// 失敗した場合はロックを解放して返します。
func (u *SongBatchJobUsecase) begin(
	ctx context.Context,
	newJob func(id uuid.UUID, startedAt time.Time) *entity.SongBatchJob,
) (repository.BatchLock, *entity.SongBatchJob, bool, error) {
	lock, acquired, err := u.lockProvider.TryAcquire(ctx, info.SongBatchLockName)
	if err != nil || !acquired {
		return nil, nil, acquired, err
	}

	if err := u.interruptOrphanedJobs(ctx); err != nil {
		u.releaseLock(ctx, lock)
		return nil, nil, false, err
	}
	job := newJob(uuid.NewV4(), u.now())
	if err := u.jobRepo.Save(ctx, job); err != nil {
		u.releaseLock(ctx, lock)
		return nil, nil, false, err
	}
	return lock, job, true, nil
}

// interruptOrphanedJobs はプロセス停止などで終了を記録できなかったジョブを中断扱いにします。
// ロックを取得できた時点で他に実行中の楽曲バッチは存在しないため、RUNNING のまま残った行はすべて取り残されたものです。
// 取得のたびに片付けるため対象は通常0件で、多くても数件に限られます。
func (u *SongBatchJobUsecase) interruptOrphanedJobs(ctx context.Context) error {
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
		slog.Warn("取り残された楽曲バッチジョブを中断扱いにしました", "job_id", orphan.ID())
	}
	return nil
}

// run は楽曲バッチを実行して結果を記録し、ロックを解放します。
func (u *SongBatchJobUsecase) run(ctx context.Context, lock repository.BatchLock, job *entity.SongBatchJob) error {
	logger := slog.With("job_id", job.ID(), "trigger", job.Trigger(), "mode", job.Request().Mode)
	logger.Info("楽曲バッチを開始します", "fill_missing_release_date", job.Request().FillMissingReleaseDate)

	result, execErr := u.runner.Execute(ctx, job.Request())

	// 停止シグナルでキャンセルされた後も結果を記録できるよう、実行とは独立した期限を使う
	finalizeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), info.SongBatchJobFinalizeTimeout)
	defer cancel()
	defer u.releaseLock(finalizeCtx, lock)

	finishedAt := u.now()
	var finishErr error
	switch {
	case execErr == nil:
		finishErr = job.Complete(result.WarningCount, finishedAt)
		logger.Info("楽曲バッチが完了しました", "status", job.Status(), "warning_count", result.WarningCount)
	case ctx.Err() != nil:
		finishErr = job.Interrupt(finishedAt)
		logger.Warn("楽曲バッチが中断されました", "error", execErr)
	default:
		finishErr = job.Fail(result.WarningCount, execErr.Error(), finishedAt)
		logger.Error("楽曲バッチに失敗しました", "error", execErr)
	}
	if finishErr != nil {
		logger.Error("楽曲バッチの結果を反映できませんでした", "error", finishErr)
	} else if err := u.jobRepo.Save(finalizeCtx, job); err != nil {
		logger.Error("楽曲バッチの結果を記録できませんでした", "error", err)
	}

	if execErr == nil && u.denominator != nil {
		u.denominator.Invalidate(finalizeCtx)
	}
	return execErr
}

func (u *SongBatchJobUsecase) releaseLock(ctx context.Context, lock repository.BatchLock) {
	if err := lock.Release(context.WithoutCancel(ctx)); err != nil {
		slog.Error("楽曲バッチのロック解放に失敗しました", "error", err)
	}
}
