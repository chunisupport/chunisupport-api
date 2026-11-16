package usecase

import (
	"context"
	"log/slog"
	"sync"
	"time"
	"uuid"

	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
)

// batchJob はライフサイクル管理の対象となるバッチジョブ集約です。
type batchJob interface {
	ID() uuid.UUID
}

// batchJobRepository はバッチジョブ集約の保存と復元を担当します。
// 各バッチのジョブリポジトリ（repository.SongBatchJobRepository など）がそのまま満たします。
type batchJobRepository[J batchJob] interface {
	Save(ctx context.Context, job J) error
	FindByID(ctx context.Context, id uuid.UUID) (J, error)
	FindRunning(ctx context.Context) ([]J, error)
	ListRecent(ctx context.Context, limit int) ([]J, error)
}

// batchJobSpec はバッチごとに異なる設定と処理をまとめたものです。
type batchJobSpec[J batchJob, R any] struct {
	// label はログに使うバッチの名前です（例: 「楽曲バッチ」）。
	label           string
	lockName        string
	historyLimit    int
	finalizeTimeout time.Duration

	errAlreadyRunning error
	errInvalidJobID   error
	errUnavailable    error

	// logAttrs はジョブの全ログに含める項目を返します。
	logAttrs func(job J) []any
	// execute はバッチ1回分を実行します。成功時の後処理（キャッシュ更新など）もここで行い、完了の記録より先に終わらせます。
	execute func(ctx context.Context, job J, logger *slog.Logger) (R, error)
	// interruptOrphan はプロセス停止などで終了を記録できなかったジョブを中断扱いにします。
	interruptOrphan func(job J, finishedAt time.Time) error

	// complete・interrupt・fail は実行結果に応じてジョブを終了状態へ遷移させます。
	// 遷移メソッドの引数はエンティティごとに異なるため、バッチごとに差し替えます。
	complete  func(job J, result R, finishedAt time.Time) error
	interrupt func(job J, result R, execErr error, finishedAt time.Time) error
	fail      func(job J, result R, execErr error, finishedAt time.Time) error
	// completedLogAttrs は成功時のログに含める項目を返します。
	completedLogAttrs func(job J, result R, finishedAt time.Time) []any
}

// batchJobLifecycle はバッチ実行をジョブとして記録するライフサイクルを管理します。
// ロック取得・取り残されたジョブの片付け・ジョブ記録・結果の記録・ロック解放・受付停止をバッチ間で共通化し、
// エンティティ固有の状態遷移とログだけを batchJobSpec で差し替えます。
// 同じアドバイザリロックを取得するため、CLI と管理画面のどちらから起動しても同時に実行されるジョブは常に1つです。
type batchJobLifecycle[J batchJob, R any] struct {
	backgroundCtx context.Context
	lockProvider  repository.BatchLockProvider
	jobRepo       batchJobRepository[J]
	spec          batchJobSpec[J, R]
	now           func() time.Time

	// mu は受付停止の判定と WaitGroup への登録を不可分にし、Wait 中の登録を防ぎます。
	mu     sync.Mutex
	closed bool
	wg     sync.WaitGroup
}

// newBatchJobLifecycle は batchJobLifecycle を生成します。
// backgroundCtx は管理画面から起動したジョブの寿命で、サーバー停止時にキャンセルされるものを渡します。
func newBatchJobLifecycle[J batchJob, R any](
	backgroundCtx context.Context,
	lockProvider repository.BatchLockProvider,
	jobRepo batchJobRepository[J],
	spec batchJobSpec[J, R],
) *batchJobLifecycle[J, R] {
	return &batchJobLifecycle[J, R]{
		backgroundCtx: backgroundCtx,
		lockProvider:  lockProvider,
		jobRepo:       jobRepo,
		spec:          spec,
		now:           time.Now,
	}
}

// runFromCLI はバッチを同期実行します。
// 別のジョブが実行中の場合は実行せず acquired=false を返し、判断は呼び出し元に委ねます。
func (l *batchJobLifecycle[J, R]) runFromCLI(ctx context.Context, newJob func(id uuid.UUID, startedAt time.Time) J) (acquired bool, err error) {
	lock, job, acquired, err := l.begin(ctx, newJob)
	if err != nil || !acquired {
		return acquired, err
	}
	return true, l.run(ctx, lock, job)
}

// startFromAdmin はジョブを記録した時点で返し、バックグラウンドでバッチを実行します。
// 処理には時間がかかるため、結果は履歴から確認させます。
// start はバックグラウンド処理の開始前に呼ばれ、呼び出し元へ返すジョブ（開始時点の複製など）を作ります。
func (l *batchJobLifecycle[J, R]) startFromAdmin(
	ctx context.Context,
	newJob func(id uuid.UUID, startedAt time.Time) J,
	start func(job J) J,
) (J, error) {
	var zero J
	l.mu.Lock()
	defer l.mu.Unlock()
	// 停止シグナル後に受け付けても即座に中断されるだけのため、ロックを取る前に拒否する
	if l.closed || l.backgroundCtx.Err() != nil {
		return zero, l.spec.errUnavailable
	}

	lock, job, acquired, err := l.begin(ctx, newJob)
	if err != nil {
		return zero, err
	}
	if !acquired {
		return zero, l.spec.errAlreadyRunning
	}

	started := start(job)
	l.wg.Go(func() {
		_ = l.run(l.backgroundCtx, lock, job)
	})
	return started, nil
}

// Wait は新しいジョブの受付を停止し、バックグラウンドで実行中のジョブが結果を記録し終えるまで待ちます。
// サーバー停止時、DB接続を閉じる前に呼び出します。
func (l *batchJobLifecycle[J, R]) Wait() {
	l.mu.Lock()
	l.closed = true
	l.mu.Unlock()
	l.wg.Wait()
}

// List は直近の実行履歴を開始日時の新しい順に返します。
func (l *batchJobLifecycle[J, R]) List(ctx context.Context) ([]J, error) {
	return l.jobRepo.ListRecent(ctx, l.spec.historyLimit)
}

// Get は指定したジョブを返します。
func (l *batchJobLifecycle[J, R]) Get(ctx context.Context, id string) (J, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		var zero J
		return zero, l.spec.errInvalidJobID
	}
	return l.jobRepo.FindByID(ctx, parsed)
}

// begin はロックを取得し、取り残されたジョブを片付けてから新しいジョブを記録します。
// 失敗した場合はロックを解放して返します。
func (l *batchJobLifecycle[J, R]) begin(
	ctx context.Context,
	newJob func(id uuid.UUID, startedAt time.Time) J,
) (repository.BatchLock, J, bool, error) {
	var zero J
	lock, acquired, err := l.lockProvider.TryAcquire(ctx, l.spec.lockName)
	if err != nil || !acquired {
		return nil, zero, acquired, err
	}

	if err := l.interruptOrphanedJobs(ctx); err != nil {
		l.releaseLock(ctx, lock)
		return nil, zero, false, err
	}
	job := newJob(uuid.NewV4(), l.now())
	if err := l.jobRepo.Save(ctx, job); err != nil {
		l.releaseLock(ctx, lock)
		return nil, zero, false, err
	}
	return lock, job, true, nil
}

// interruptOrphanedJobs はプロセス停止などで終了を記録できなかったジョブを中断扱いにします。
// ロックを取得できた時点で他に実行中のジョブは存在しないため、RUNNING のまま残った行はすべて取り残されたものです。
// 取得のたびに片付けるため対象は通常0件で、多くても数件に限られます。
func (l *batchJobLifecycle[J, R]) interruptOrphanedJobs(ctx context.Context) error {
	orphans, err := l.jobRepo.FindRunning(ctx)
	if err != nil {
		return err
	}
	for _, orphan := range orphans {
		if err := l.spec.interruptOrphan(orphan, l.now()); err != nil {
			return err
		}
		if err := l.jobRepo.Save(ctx, orphan); err != nil {
			return err
		}
		slog.WarnContext(ctx, "取り残された"+l.spec.label+"ジョブを中断扱いにしました", "job_id", orphan.ID())
	}
	return nil
}

// run はバッチを実行して結果を記録し、ロックを解放します。
func (l *batchJobLifecycle[J, R]) run(ctx context.Context, lock repository.BatchLock, job J) error {
	logger := slog.With(l.spec.logAttrs(job)...)

	result, execErr := l.spec.execute(ctx, job, logger)
	// 実行や記録に時間がかかっても解放できるよう、解放専用の期限を使う
	defer l.releaseLock(ctx, lock)

	// 停止シグナルでキャンセルされた後も結果を記録できるよう、実行とは独立した期限を使う
	finalizeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), l.spec.finalizeTimeout)
	defer cancel()

	finishedAt := l.now()
	var finishErr error
	switch {
	case execErr == nil:
		finishErr = l.spec.complete(job, result, finishedAt)
		logger.Info(l.spec.label+"が完了しました", l.spec.completedLogAttrs(job, result, finishedAt)...)
	case ctx.Err() != nil:
		finishErr = l.spec.interrupt(job, result, execErr, finishedAt)
		logger.Warn(l.spec.label+"が中断されました", "error", execErr)
	default:
		finishErr = l.spec.fail(job, result, execErr, finishedAt)
		logger.Error(l.spec.label+"に失敗しました", "error", execErr)
	}
	if finishErr != nil {
		logger.Error(l.spec.label+"の結果を反映できませんでした", "error", finishErr)
	} else if err := l.jobRepo.Save(finalizeCtx, job); err != nil {
		logger.Error(l.spec.label+"の結果を記録できませんでした", "error", err)
	}
	return execErr
}

// releaseLock はロックを解放します。
// リクエストのキャンセルや他の処理の期限に関係なく解放できるよう、解放専用の期限だけを持つ context を使います。
func (l *batchJobLifecycle[J, R]) releaseLock(ctx context.Context, lock repository.BatchLock) {
	releaseCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), l.spec.finalizeTimeout)
	defer cancel()
	if err := lock.Release(releaseCtx); err != nil {
		slog.ErrorContext(releaseCtx, l.spec.label+"のロック解放に失敗しました", "error", err)
	}
}
