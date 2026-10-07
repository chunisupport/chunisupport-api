package usecase

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
	"uuid"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username/usernametest"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeChartStatsBatchJobRepository struct {
	mu      sync.Mutex
	jobs    map[uuid.UUID]entity.ChartStatsBatchJob
	limit   int
	saveErr error
}

func newFakeChartStatsBatchJobRepository(jobs ...*entity.ChartStatsBatchJob) *fakeChartStatsBatchJobRepository {
	repo := &fakeChartStatsBatchJobRepository{jobs: make(map[uuid.UUID]entity.ChartStatsBatchJob)}
	for _, job := range jobs {
		repo.jobs[job.ID()] = *job
	}
	return repo
}

func (r *fakeChartStatsBatchJobRepository) Save(_ context.Context, job *entity.ChartStatsBatchJob) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.saveErr != nil {
		return r.saveErr
	}
	r.jobs[job.ID()] = *job
	return nil
}

func (r *fakeChartStatsBatchJobRepository) FindByID(_ context.Context, id uuid.UUID) (*entity.ChartStatsBatchJob, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	job, ok := r.jobs[id]
	if !ok {
		return nil, repository.ErrChartStatsBatchJobNotFound
	}
	return &job, nil
}

func (r *fakeChartStatsBatchJobRepository) FindRunning(context.Context) ([]*entity.ChartStatsBatchJob, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var running []*entity.ChartStatsBatchJob
	for _, job := range r.jobs {
		if job.IsRunning() {
			running = append(running, &job)
		}
	}
	return running, nil
}

func (r *fakeChartStatsBatchJobRepository) ListRecent(_ context.Context, limit int) ([]*entity.ChartStatsBatchJob, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.limit = limit
	jobs := make([]*entity.ChartStatsBatchJob, 0, len(r.jobs))
	for _, job := range r.jobs {
		jobs = append(jobs, &job)
	}
	return jobs, nil
}

func (r *fakeChartStatsBatchJobRepository) only(t *testing.T) entity.ChartStatsBatchJob {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	require.Len(t, r.jobs, 1)
	for _, job := range r.jobs {
		return job
	}
	return entity.ChartStatsBatchJob{}
}

type fakeChartStatsBatchRunner struct {
	err    error
	cancel context.CancelFunc
	called bool
}

func (r *fakeChartStatsBatchRunner) Execute(ctx context.Context) (ChartStatsBatchResult, error) {
	r.called = true
	if r.cancel != nil {
		// err がある場合は、キャンセルと同時に別のエラーも返すケースを再現します。
		r.cancel()
		if r.err != nil {
			return ChartStatsBatchResult{}, r.err
		}
		return ChartStatsBatchResult{}, ctx.Err()
	}
	return ChartStatsBatchResult{}, r.err
}

var chartStatsBatchJobNow = time.Date(2026, 9, 28, 3, 0, 0, 0, time.UTC)

func newChartStatsBatchRequester(t *testing.T) entity.ChartStatsBatchJobRequester {
	t.Helper()
	return entity.ChartStatsBatchJobRequester{UserID: 10, Username: usernametest.New(t, "adminuser")}
}

func newTestChartStatsBatchJobUsecase(
	backgroundCtx context.Context,
	lockProvider *fakeSongBatchLockProvider,
	repo *fakeChartStatsBatchJobRepository,
	runner *fakeChartStatsBatchRunner,
) *ChartStatsBatchJobUsecase {
	uc := NewChartStatsBatchJobUsecase(backgroundCtx, lockProvider, repo, runner)
	uc.now = func() time.Time { return chartStatsBatchJobNow }
	return uc
}

func TestChartStatsBatchJobUsecase_RunFromCLI(t *testing.T) {
	tests := []struct {
		name            string
		runner          *fakeChartStatsBatchRunner
		cancelDuringRun bool
		wantErr         bool
		expectedStatus  entity.ChartStatsBatchJobStatus
		expectedMessage string
	}{
		{
			name:           "成功した場合は成功を記録する",
			runner:         &fakeChartStatsBatchRunner{},
			expectedStatus: entity.ChartStatsBatchJobStatusSucceeded,
		},
		{
			name:            "失敗した場合は失敗理由を記録してエラーを返す",
			runner:          &fakeChartStatsBatchRunner{err: errors.New("replace chart stats: deadlock")},
			wantErr:         true,
			expectedStatus:  entity.ChartStatsBatchJobStatusFailed,
			expectedMessage: "replace chart stats: deadlock",
		},
		{
			name:            "実行中にキャンセルされた場合は中断を記録する",
			runner:          &fakeChartStatsBatchRunner{},
			cancelDuringRun: true,
			wantErr:         true,
			expectedStatus:  entity.ChartStatsBatchJobStatusInterrupted,
			expectedMessage: context.Canceled.Error(),
		},
		{
			name:            "キャンセルと同時に別のエラーで失敗した場合もエラー内容を中断の理由として残す",
			runner:          &fakeChartStatsBatchRunner{err: errors.New("replace chart stats: connection reset")},
			cancelDuringRun: true,
			wantErr:         true,
			expectedStatus:  entity.ChartStatsBatchJobStatusInterrupted,
			expectedMessage: "replace chart stats: connection reset",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)
			if tt.cancelDuringRun {
				tt.runner.cancel = cancel
			}
			lockProvider := &fakeSongBatchLockProvider{acquired: true}
			repo := newFakeChartStatsBatchJobRepository()
			uc := newTestChartStatsBatchJobUsecase(context.Background(), lockProvider, repo, tt.runner)

			// When
			acquired, err := uc.RunFromCLI(ctx)

			// Then
			assert.True(t, acquired)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, []string{info.ChartStatsBatchLockName}, lockProvider.names)
			assert.True(t, lockProvider.lock.released)
			job := repo.only(t)
			assert.Equal(t, entity.ChartStatsBatchJobTriggerCLI, job.Trigger())
			assert.Equal(t, tt.expectedStatus, job.Status())
			assert.Equal(t, tt.expectedMessage, job.ErrorMessage())
		})
	}
}

func TestChartStatsBatchJobUsecase_RunFromCLI_ロックを取得できない場合は実行しない(t *testing.T) {
	// Given
	runner := &fakeChartStatsBatchRunner{}
	repo := newFakeChartStatsBatchJobRepository()
	uc := newTestChartStatsBatchJobUsecase(context.Background(), &fakeSongBatchLockProvider{}, repo, runner)

	// When
	acquired, err := uc.RunFromCLI(context.Background())

	// Then
	assert.False(t, acquired)
	assert.NoError(t, err)
	assert.False(t, runner.called)
	assert.Empty(t, repo.jobs)
}

func TestChartStatsBatchJobUsecase_ロック取得後に取り残された実行中ジョブを中断扱いにする(t *testing.T) {
	// Given
	orphan := entity.StartChartStatsBatchJobFromCLI(uuid.NewV4(), chartStatsBatchJobNow.Add(-time.Hour))
	repo := newFakeChartStatsBatchJobRepository(orphan)
	uc := newTestChartStatsBatchJobUsecase(context.Background(), &fakeSongBatchLockProvider{acquired: true}, repo, &fakeChartStatsBatchRunner{})

	// When
	_, err := uc.RunFromCLI(context.Background())

	// Then
	require.NoError(t, err)
	found, err := repo.FindByID(context.Background(), orphan.ID())
	require.NoError(t, err)
	assert.Equal(t, entity.ChartStatsBatchJobStatusInterrupted, found.Status())
	require.NotNil(t, found.FinishedAt())
	assert.Equal(t, chartStatsBatchJobNow, *found.FinishedAt())
}

func TestChartStatsBatchJobUsecase_ジョブを記録できない場合はロックを解放して実行しない(t *testing.T) {
	// Given
	lockProvider := &fakeSongBatchLockProvider{acquired: true}
	repo := newFakeChartStatsBatchJobRepository()
	repo.saveErr = errors.New("db down")
	runner := &fakeChartStatsBatchRunner{}
	uc := newTestChartStatsBatchJobUsecase(context.Background(), lockProvider, repo, runner)

	// When
	acquired, err := uc.RunFromCLI(context.Background())

	// Then
	assert.False(t, acquired)
	assert.Error(t, err)
	assert.True(t, lockProvider.lock.released)
	assert.False(t, runner.called)
}

func TestChartStatsBatchJobUsecase_StartFromAdmin(t *testing.T) {
	// Given
	lockProvider := &fakeSongBatchLockProvider{acquired: true}
	repo := newFakeChartStatsBatchJobRepository()
	runner := &fakeChartStatsBatchRunner{}
	uc := newTestChartStatsBatchJobUsecase(context.Background(), lockProvider, repo, runner)

	// When
	requester := newChartStatsBatchRequester(t)
	job, err := uc.StartFromAdmin(context.Background(), requester)
	uc.Wait()

	// Then
	require.NoError(t, err)
	assert.Equal(t, entity.ChartStatsBatchJobStatusRunning, job.Status())
	assert.Equal(t, &requester, job.Requester())
	assert.True(t, runner.called)
	assert.True(t, lockProvider.lock.released)
	saved, err := repo.FindByID(context.Background(), job.ID())
	require.NoError(t, err)
	assert.Equal(t, entity.ChartStatsBatchJobTriggerAdmin, saved.Trigger())
	assert.Equal(t, entity.ChartStatsBatchJobStatusSucceeded, saved.Status())
}

func TestChartStatsBatchJobUsecase_StartFromAdmin_実行中の場合は受け付けない(t *testing.T) {
	// Given
	runner := &fakeChartStatsBatchRunner{}
	repo := newFakeChartStatsBatchJobRepository()
	uc := newTestChartStatsBatchJobUsecase(context.Background(), &fakeSongBatchLockProvider{}, repo, runner)

	// When
	requester := newChartStatsBatchRequester(t)
	_, err := uc.StartFromAdmin(context.Background(), requester)
	uc.Wait()

	// Then
	assert.ErrorIs(t, err, ErrChartStatsBatchAlreadyRunning)
	assert.False(t, runner.called)
	assert.Empty(t, repo.jobs)
}

func TestChartStatsBatchJobUsecase_StartFromAdmin_リクエストがキャンセルされても実行を続ける(t *testing.T) {
	// Given: 受付直後に終了する HTTP リクエストの context
	requestCtx, cancelRequest := context.WithCancel(context.Background())
	repo := newFakeChartStatsBatchJobRepository()
	uc := newTestChartStatsBatchJobUsecase(context.Background(), &fakeSongBatchLockProvider{acquired: true}, repo, &fakeChartStatsBatchRunner{})

	// When
	requester := newChartStatsBatchRequester(t)
	job, err := uc.StartFromAdmin(requestCtx, requester)
	cancelRequest()
	uc.Wait()

	// Then
	require.NoError(t, err)
	saved, err := repo.FindByID(context.Background(), job.ID())
	require.NoError(t, err)
	assert.Equal(t, entity.ChartStatsBatchJobStatusSucceeded, saved.Status())
}

func TestChartStatsBatchJobUsecase_StartFromAdmin_停止処理後は受け付けない(t *testing.T) {
	tests := []struct {
		name  string
		setup func(uc *ChartStatsBatchJobUsecase, cancel context.CancelFunc)
	}{
		{name: "停止シグナルを受けた後は受け付けない", setup: func(_ *ChartStatsBatchJobUsecase, cancel context.CancelFunc) { cancel() }},
		{name: "完了待ちを始めた後は受け付けない", setup: func(uc *ChartStatsBatchJobUsecase, _ context.CancelFunc) { uc.Wait() }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			backgroundCtx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)
			lockProvider := &fakeSongBatchLockProvider{acquired: true}
			runner := &fakeChartStatsBatchRunner{}
			uc := newTestChartStatsBatchJobUsecase(backgroundCtx, lockProvider, newFakeChartStatsBatchJobRepository(), runner)
			tt.setup(uc, cancel)

			// When
			requester := newChartStatsBatchRequester(t)
			_, err := uc.StartFromAdmin(context.Background(), requester)

			// Then
			assert.ErrorIs(t, err, ErrChartStatsBatchUnavailable)
			assert.Empty(t, lockProvider.names)
			assert.False(t, runner.called)
		})
	}
}

func TestChartStatsBatchJobUsecase_List(t *testing.T) {
	// Given
	job := entity.StartChartStatsBatchJobFromCLI(uuid.NewV4(), chartStatsBatchJobNow)
	repo := newFakeChartStatsBatchJobRepository(job)
	uc := newTestChartStatsBatchJobUsecase(context.Background(), &fakeSongBatchLockProvider{}, repo, &fakeChartStatsBatchRunner{})

	// When
	jobs, err := uc.List(context.Background())

	// Then
	require.NoError(t, err)
	assert.Len(t, jobs, 1)
	assert.Equal(t, info.ChartStatsBatchJobHistoryLimit, repo.limit)
}

func TestChartStatsBatchJobUsecase_Get(t *testing.T) {
	job := entity.StartChartStatsBatchJobFromCLI(uuid.NewV4(), chartStatsBatchJobNow)
	tests := []struct {
		name        string
		id          string
		expectedErr error
	}{
		{name: "存在するジョブを返す", id: job.ID().String()},
		{name: "UUIDでないIDはエラー", id: "0199", expectedErr: ErrInvalidChartStatsBatchJobID},
		{name: "存在しないジョブはエラー", id: uuid.NewV4().String(), expectedErr: repository.ErrChartStatsBatchJobNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			repo := newFakeChartStatsBatchJobRepository(job)
			uc := newTestChartStatsBatchJobUsecase(context.Background(), &fakeSongBatchLockProvider{}, repo, &fakeChartStatsBatchRunner{})

			// When
			found, err := uc.Get(context.Background(), tt.id)

			// Then
			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, job.ID(), found.ID())
		})
	}
}
