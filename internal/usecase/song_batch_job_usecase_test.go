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
	"github.com/chunisupport/chunisupport-api/internal/domain/songbatch"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeSongBatchLock struct {
	released bool
}

func (l *fakeSongBatchLock) Release(context.Context) error {
	l.released = true
	return nil
}

type fakeSongBatchLockProvider struct {
	acquired bool
	lock     *fakeSongBatchLock
	names    []string
}

func (p *fakeSongBatchLockProvider) TryAcquire(_ context.Context, name string) (repository.BatchLock, bool, error) {
	p.names = append(p.names, name)
	if !p.acquired {
		return nil, false, nil
	}
	p.lock = &fakeSongBatchLock{}
	return p.lock, true, nil
}

type fakeSongBatchJobRepository struct {
	mu      sync.Mutex
	jobs    map[uuid.UUID]entity.SongBatchJob
	limit   int
	saveErr error
}

func newFakeSongBatchJobRepository(jobs ...*entity.SongBatchJob) *fakeSongBatchJobRepository {
	repo := &fakeSongBatchJobRepository{jobs: make(map[uuid.UUID]entity.SongBatchJob)}
	for _, job := range jobs {
		repo.jobs[job.ID()] = *job
	}
	return repo
}

func (r *fakeSongBatchJobRepository) Save(_ context.Context, job *entity.SongBatchJob) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.saveErr != nil {
		return r.saveErr
	}
	r.jobs[job.ID()] = *job
	return nil
}

func (r *fakeSongBatchJobRepository) FindByID(_ context.Context, id uuid.UUID) (*entity.SongBatchJob, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	job, ok := r.jobs[id]
	if !ok {
		return nil, repository.ErrSongBatchJobNotFound
	}
	return &job, nil
}

func (r *fakeSongBatchJobRepository) FindRunning(context.Context) ([]*entity.SongBatchJob, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var running []*entity.SongBatchJob
	for _, job := range r.jobs {
		if job.IsRunning() {
			running = append(running, &job)
		}
	}
	return running, nil
}

func (r *fakeSongBatchJobRepository) ListRecent(_ context.Context, limit int) ([]*entity.SongBatchJob, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.limit = limit
	jobs := make([]*entity.SongBatchJob, 0, len(r.jobs))
	for _, job := range r.jobs {
		jobs = append(jobs, &job)
	}
	return jobs, nil
}

func (r *fakeSongBatchJobRepository) only(t *testing.T) entity.SongBatchJob {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	require.Len(t, r.jobs, 1)
	for _, job := range r.jobs {
		return job
	}
	return entity.SongBatchJob{}
}

type fakeSongBatchRunner struct {
	result  SongBatchResult
	err     error
	cancel  context.CancelFunc
	called  bool
	request songbatch.RunRequest
}

func (r *fakeSongBatchRunner) Execute(ctx context.Context, req songbatch.RunRequest) (SongBatchResult, error) {
	r.called = true
	r.request = req
	if r.cancel != nil {
		// 実行中に停止シグナルを受けた状況を再現する
		r.cancel()
		return r.result, ctx.Err()
	}
	return r.result, r.err
}

type fakeOverpowerDenominatorInvalidator struct {
	invalidated bool
}

func (f *fakeOverpowerDenominatorInvalidator) Snapshot(context.Context) (*repository.OverpowerDenominatorSnapshot, error) {
	return nil, nil
}

func (f *fakeOverpowerDenominatorInvalidator) Invalidate(context.Context) {
	f.invalidated = true
}

var songBatchJobNow = time.Date(2026, 9, 26, 3, 0, 0, 0, time.UTC)

func newTestSongBatchJobUsecase(
	backgroundCtx context.Context,
	lockProvider *fakeSongBatchLockProvider,
	repo *fakeSongBatchJobRepository,
	runner *fakeSongBatchRunner,
	denominator *fakeOverpowerDenominatorInvalidator,
) *SongBatchJobUsecase {
	// 型付き nil をインターフェースへ渡すと非 nil 扱いになるため、未指定時は nil インターフェースにする
	var denominatorProvider repository.OverpowerDenominatorProvider
	if denominator != nil {
		denominatorProvider = denominator
	}
	uc := NewSongBatchJobUsecase(backgroundCtx, lockProvider, repo, runner, denominatorProvider)
	uc.now = func() time.Time { return songBatchJobNow }
	return uc
}

func TestSongBatchJobUsecase_RunFromCLI(t *testing.T) {
	tests := []struct {
		name                string
		runner              *fakeSongBatchRunner
		cancelDuringRun     bool
		wantErr             bool
		expectedStatus      entity.SongBatchJobStatus
		expectedWarnings    int
		expectedMessage     string
		expectedInvalidated bool
	}{
		{
			name:                "成功した場合は成功を記録してOVER POWER分母を無効化する",
			runner:              &fakeSongBatchRunner{},
			expectedStatus:      entity.SongBatchJobStatusSucceeded,
			expectedInvalidated: true,
		},
		{
			name:                "補完データソースを除外した場合は警告付き成功を記録する",
			runner:              &fakeSongBatchRunner{result: SongBatchResult{WarningCount: 2}},
			expectedStatus:      entity.SongBatchJobStatusSucceededWithWarnings,
			expectedWarnings:    2,
			expectedInvalidated: true,
		},
		{
			name:             "失敗した場合は失敗理由を記録してエラーを返す",
			runner:           &fakeSongBatchRunner{result: SongBatchResult{WarningCount: 1}, err: errors.New("required datasource official failed")},
			wantErr:          true,
			expectedStatus:   entity.SongBatchJobStatusFailed,
			expectedWarnings: 1,
			expectedMessage:  "required datasource official failed",
		},
		{
			name:             "実行中にキャンセルされた場合は中断までの警告件数とともに中断を記録する",
			runner:           &fakeSongBatchRunner{result: SongBatchResult{WarningCount: 1}},
			cancelDuringRun:  true,
			wantErr:          true,
			expectedStatus:   entity.SongBatchJobStatusInterrupted,
			expectedWarnings: 1,
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
			repo := newFakeSongBatchJobRepository()
			denominator := &fakeOverpowerDenominatorInvalidator{}
			uc := newTestSongBatchJobUsecase(context.Background(), lockProvider, repo, tt.runner, denominator)
			req := songbatch.NewRunRequest(false, true)

			// When
			acquired, err := uc.RunFromCLI(ctx, req)

			// Then
			assert.True(t, acquired)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, req, tt.runner.request)
			assert.Equal(t, []string{info.SongBatchLockName}, lockProvider.names)
			assert.True(t, lockProvider.lock.released)
			job := repo.only(t)
			assert.Equal(t, entity.SongBatchJobTriggerCLI, job.Trigger())
			assert.Equal(t, tt.expectedStatus, job.Status())
			assert.Equal(t, tt.expectedWarnings, job.WarningCount())
			assert.Equal(t, tt.expectedMessage, job.ErrorMessage())
			assert.Equal(t, tt.expectedInvalidated, denominator.invalidated)
		})
	}
}

func TestSongBatchJobUsecase_RunFromCLI_ロックを取得できない場合は実行しない(t *testing.T) {
	// Given
	runner := &fakeSongBatchRunner{}
	repo := newFakeSongBatchJobRepository()
	uc := newTestSongBatchJobUsecase(context.Background(), &fakeSongBatchLockProvider{}, repo, runner, nil)

	// When
	acquired, err := uc.RunFromCLI(context.Background(), songbatch.NewRunRequest(false, false))

	// Then
	assert.False(t, acquired)
	assert.NoError(t, err)
	assert.False(t, runner.called)
	assert.Empty(t, repo.jobs)
}

func TestSongBatchJobUsecase_ロック取得後に取り残された実行中ジョブを中断扱いにする(t *testing.T) {
	// Given
	orphan := entity.StartSongBatchJobFromCLI(uuid.NewV4(), songbatch.NewRunRequest(true, false), songBatchJobNow.Add(-time.Hour))
	repo := newFakeSongBatchJobRepository(orphan)
	uc := newTestSongBatchJobUsecase(context.Background(), &fakeSongBatchLockProvider{acquired: true}, repo, &fakeSongBatchRunner{}, nil)

	// When
	_, err := uc.RunFromCLI(context.Background(), songbatch.NewRunRequest(false, false))

	// Then
	require.NoError(t, err)
	found, err := repo.FindByID(context.Background(), orphan.ID())
	require.NoError(t, err)
	assert.Equal(t, entity.SongBatchJobStatusInterrupted, found.Status())
	require.NotNil(t, found.FinishedAt())
	assert.Equal(t, songBatchJobNow, *found.FinishedAt())
}

func TestSongBatchJobUsecase_StartFromAdmin(t *testing.T) {
	// Given
	lockProvider := &fakeSongBatchLockProvider{acquired: true}
	repo := newFakeSongBatchJobRepository()
	runner := &fakeSongBatchRunner{}
	denominator := &fakeOverpowerDenominatorInvalidator{}
	uc := newTestSongBatchJobUsecase(context.Background(), lockProvider, repo, runner, denominator)
	requester := entity.SongBatchJobRequester{UserID: 10, Username: username.MustNewUserName("adminuser")}
	req := songbatch.NewRunRequest(true, false)

	// When
	job, err := uc.StartFromAdmin(context.Background(), requester, req)
	uc.Wait()

	// Then
	require.NoError(t, err)
	assert.Equal(t, entity.SongBatchJobStatusRunning, job.Status())
	assert.Equal(t, &requester, job.Requester())
	assert.Equal(t, req, runner.request)
	assert.True(t, lockProvider.lock.released)
	saved, err := repo.FindByID(context.Background(), job.ID())
	require.NoError(t, err)
	assert.Equal(t, entity.SongBatchJobTriggerAdmin, saved.Trigger())
	assert.Equal(t, entity.SongBatchJobStatusSucceeded, saved.Status())
	assert.True(t, denominator.invalidated)
}

func TestSongBatchJobUsecase_StartFromAdmin_実行中の場合は受け付けない(t *testing.T) {
	// Given
	runner := &fakeSongBatchRunner{}
	repo := newFakeSongBatchJobRepository()
	uc := newTestSongBatchJobUsecase(context.Background(), &fakeSongBatchLockProvider{}, repo, runner, nil)
	requester := entity.SongBatchJobRequester{UserID: 10, Username: username.MustNewUserName("adminuser")}

	// When
	_, err := uc.StartFromAdmin(context.Background(), requester, songbatch.NewRunRequest(false, false))
	uc.Wait()

	// Then
	assert.ErrorIs(t, err, ErrSongBatchAlreadyRunning)
	assert.False(t, runner.called)
	assert.Empty(t, repo.jobs)
}

func TestSongBatchJobUsecase_StartFromAdmin_リクエストがキャンセルされても実行を続ける(t *testing.T) {
	// Given: 受付直後に終了する HTTP リクエストの context
	requestCtx, cancelRequest := context.WithCancel(context.Background())
	repo := newFakeSongBatchJobRepository()
	runner := &fakeSongBatchRunner{}
	uc := newTestSongBatchJobUsecase(context.Background(), &fakeSongBatchLockProvider{acquired: true}, repo, runner, nil)
	requester := entity.SongBatchJobRequester{UserID: 10, Username: username.MustNewUserName("adminuser")}

	// When
	job, err := uc.StartFromAdmin(requestCtx, requester, songbatch.NewRunRequest(false, false))
	cancelRequest()
	uc.Wait()

	// Then
	require.NoError(t, err)
	saved, err := repo.FindByID(context.Background(), job.ID())
	require.NoError(t, err)
	assert.Equal(t, entity.SongBatchJobStatusSucceeded, saved.Status())
}

func TestSongBatchJobUsecase_StartFromAdmin_停止処理後は受け付けない(t *testing.T) {
	tests := []struct {
		name  string
		setup func(uc *SongBatchJobUsecase, cancel context.CancelFunc)
	}{
		{name: "停止シグナルを受けた後は受け付けない", setup: func(_ *SongBatchJobUsecase, cancel context.CancelFunc) { cancel() }},
		{name: "完了待ちを始めた後は受け付けない", setup: func(uc *SongBatchJobUsecase, _ context.CancelFunc) { uc.Wait() }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			backgroundCtx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)
			lockProvider := &fakeSongBatchLockProvider{acquired: true}
			runner := &fakeSongBatchRunner{}
			uc := newTestSongBatchJobUsecase(backgroundCtx, lockProvider, newFakeSongBatchJobRepository(), runner, nil)
			tt.setup(uc, cancel)
			requester := entity.SongBatchJobRequester{UserID: 10, Username: username.MustNewUserName("adminuser")}

			// When
			_, err := uc.StartFromAdmin(context.Background(), requester, songbatch.NewRunRequest(false, false))

			// Then
			assert.ErrorIs(t, err, ErrSongBatchUnavailable)
			assert.Empty(t, lockProvider.names)
			assert.False(t, runner.called)
		})
	}
}

func TestSongBatchJobUsecase_ジョブを記録できない場合はロックを解放して実行しない(t *testing.T) {
	// Given
	lockProvider := &fakeSongBatchLockProvider{acquired: true}
	repo := newFakeSongBatchJobRepository()
	repo.saveErr = errors.New("db down")
	runner := &fakeSongBatchRunner{}
	uc := newTestSongBatchJobUsecase(context.Background(), lockProvider, repo, runner, nil)

	// When
	acquired, err := uc.RunFromCLI(context.Background(), songbatch.NewRunRequest(false, false))

	// Then
	assert.False(t, acquired)
	assert.Error(t, err)
	assert.True(t, lockProvider.lock.released)
	assert.False(t, runner.called)
}

func TestSongBatchJobUsecase_List(t *testing.T) {
	// Given
	job := entity.StartSongBatchJobFromCLI(uuid.NewV4(), songbatch.NewRunRequest(false, false), songBatchJobNow)
	repo := newFakeSongBatchJobRepository(job)
	uc := newTestSongBatchJobUsecase(context.Background(), &fakeSongBatchLockProvider{}, repo, &fakeSongBatchRunner{}, nil)

	// When
	jobs, err := uc.List(context.Background())

	// Then
	require.NoError(t, err)
	assert.Len(t, jobs, 1)
	assert.Equal(t, info.SongBatchJobHistoryLimit, repo.limit)
}

func TestSongBatchJobUsecase_Get(t *testing.T) {
	job := entity.StartSongBatchJobFromCLI(uuid.NewV4(), songbatch.NewRunRequest(false, false), songBatchJobNow)

	tests := []struct {
		name    string
		id      string
		wantErr error
	}{
		{name: "存在するジョブを取得できる", id: job.ID().String()},
		{name: "UUIDでないIDは不正なIDエラー", id: "not-a-uuid", wantErr: ErrInvalidSongBatchJobID},
		{name: "存在しないジョブは未検出エラー", id: uuid.NewV4().String(), wantErr: repository.ErrSongBatchJobNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			uc := newTestSongBatchJobUsecase(context.Background(), &fakeSongBatchLockProvider{}, newFakeSongBatchJobRepository(job), &fakeSongBatchRunner{}, nil)

			// When
			found, err := uc.Get(context.Background(), tt.id)

			// Then
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, job.ID(), found.ID())
		})
	}
}
