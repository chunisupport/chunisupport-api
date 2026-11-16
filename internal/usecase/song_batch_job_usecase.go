package usecase

import (
	"context"
	"errors"
	"log/slog"
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
	// ErrSongBatchUnavailable はサーバー停止処理中のため新しいジョブを受け付けられないことを表します。
	ErrSongBatchUnavailable = errors.New("song batch is unavailable")
)

// SongBatchRunner は楽曲バッチ1回分を実行します。
type SongBatchRunner interface {
	Execute(ctx context.Context, req songbatch.RunRequest) (SongBatchResult, error)
}

// SongBatchJobUsecase は楽曲バッチの実行をジョブとして記録し、CLI と管理画面の両方から起動できるようにします。
// ジョブのライフサイクル（ロック・記録・受付停止）は batchJobLifecycle に任せ、楽曲バッチ固有の実行内容だけを持ちます。
type SongBatchJobUsecase struct {
	*batchJobLifecycle[*entity.SongBatchJob, SongBatchResult]

	runner      SongBatchRunner
	denominator repository.OverpowerDenominatorProvider
	songCatalog repository.SongCatalogProvider
}

// NewSongBatchJobUsecase は SongBatchJobUsecase を生成します。
// backgroundCtx は管理画面から起動したジョブの寿命で、サーバー停止時にキャンセルされるものを渡します。
// denominator と songCatalog は同期後に更新するプロセス内キャッシュで、キャッシュを持たない CLI では nil を渡します。
// CLI からの同期は、API プロセスの songCatalog が変更世代の定期確認で検知します。
func NewSongBatchJobUsecase(
	backgroundCtx context.Context,
	lockProvider repository.BatchLockProvider,
	jobRepo repository.SongBatchJobRepository,
	runner SongBatchRunner,
	denominator repository.OverpowerDenominatorProvider,
	songCatalog repository.SongCatalogProvider,
) *SongBatchJobUsecase {
	u := &SongBatchJobUsecase{
		runner:      runner,
		denominator: denominator,
		songCatalog: songCatalog,
	}
	u.batchJobLifecycle = newBatchJobLifecycle(backgroundCtx, lockProvider, jobRepo, batchJobSpec[*entity.SongBatchJob, SongBatchResult]{
		label:             "楽曲バッチ",
		lockName:          info.SongBatchLockName,
		historyLimit:      info.SongBatchJobHistoryLimit,
		finalizeTimeout:   info.SongBatchJobFinalizeTimeout,
		errAlreadyRunning: ErrSongBatchAlreadyRunning,
		errInvalidJobID:   ErrInvalidSongBatchJobID,
		errUnavailable:    ErrSongBatchUnavailable,
		logAttrs: func(job *entity.SongBatchJob) []any {
			return []any{"job_id", job.ID(), "trigger", job.Trigger(), "mode", job.Request().Mode}
		},
		execute: u.execute,
		// MySQL 同期のコミット直後に停止した可能性もあるため、取り残されたジョブの同期結果は不明として扱う
		interruptOrphan: func(job *entity.SongBatchJob, finishedAt time.Time) error {
			return job.Interrupt(job.WarningCount(), finishedAt)
		},
		complete: func(job *entity.SongBatchJob, result SongBatchResult, finishedAt time.Time) error {
			return job.Complete(result.WarningCount, finishedAt)
		},
		interrupt: func(job *entity.SongBatchJob, result SongBatchResult, _ error, finishedAt time.Time) error {
			return job.Interrupt(result.WarningCount, finishedAt)
		},
		fail: func(job *entity.SongBatchJob, result SongBatchResult, execErr error, finishedAt time.Time) error {
			return job.Fail(result.WarningCount, execErr.Error(), finishedAt)
		},
		completedLogAttrs: func(job *entity.SongBatchJob, result SongBatchResult, _ time.Time) []any {
			return []any{"status", job.Status(), "warning_count", result.WarningCount}
		},
	})
	return u
}

// RunFromCLI は CLI から楽曲バッチを同期実行します。
// 別の楽曲バッチが実行中の場合は実行せず acquired=false を返し、判断は呼び出し元（実行モード）に委ねます。
func (u *SongBatchJobUsecase) RunFromCLI(ctx context.Context, req songbatch.RunRequest) (acquired bool, err error) {
	return u.runFromCLI(ctx, func(id uuid.UUID, startedAt time.Time) *entity.SongBatchJob {
		return entity.StartSongBatchJobFromCLI(id, req, startedAt)
	})
}

// StartFromAdmin は管理画面からの実行要求を受け付け、バックグラウンドで楽曲バッチを実行します。
// 処理には数分かかるため、ジョブを記録した時点で呼び出し元へ返し、結果は履歴から確認させます。
func (u *SongBatchJobUsecase) StartFromAdmin(ctx context.Context, requester entity.SongBatchJobRequester, req songbatch.RunRequest) (*entity.SongBatchJob, error) {
	return u.startFromAdmin(ctx,
		func(id uuid.UUID, startedAt time.Time) *entity.SongBatchJob {
			return entity.StartSongBatchJobFromAdmin(id, req, requester, startedAt)
		},
		// バックグラウンド処理がジョブを更新するため、呼び出し元には開始時点の複製を返す
		func(job *entity.SongBatchJob) *entity.SongBatchJob {
			started := *job
			return &started
		},
	)
}

// execute は楽曲バッチを実行し、成功した場合はプロセス内キャッシュを更新します。
func (u *SongBatchJobUsecase) execute(ctx context.Context, job *entity.SongBatchJob, logger *slog.Logger) (SongBatchResult, error) {
	logger.Info("楽曲バッチを開始します", "fill_missing_release_date", job.Request().FillMissingReleaseDate)

	result, err := u.runner.Execute(ctx, job.Request())
	// 管理画面がジョブの完了を検知した時点で楽曲データが反映済みになるよう、完了の記録より先にキャッシュを更新する。
	// 再構築に失敗してもDB同期は成功しているため、ジョブは成功として記録し、失敗はログと定期確認での再試行に任せる。
	if err == nil {
		if u.denominator != nil {
			u.denominator.Invalidate(ctx)
		}
		refreshSongCatalog(ctx, u.songCatalog)
	}
	return result, err
}
