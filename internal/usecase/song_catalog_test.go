package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/songbatch"
	"github.com/chunisupport/chunisupport-api/internal/domain/songcatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubSongCatalogProvider は楽曲カタログの参照結果と再構築の呼び出し回数を制御します。
type stubSongCatalogProvider struct {
	catalog      *songcatalog.Catalog
	snapshotErr  error
	refreshErr   error
	refreshCount int
	// refreshCtxErr と refreshHasDeadline は、最後の Refresh に渡された ctx の状態です。
	refreshCtxErr      error
	refreshHasDeadline bool
	// onRefresh は Refresh の呼び出し時点の状態を検証するためのフックです。
	onRefresh func()
}

func (s *stubSongCatalogProvider) Snapshot(context.Context) (*songcatalog.Catalog, error) {
	if s.snapshotErr != nil {
		return nil, s.snapshotErr
	}
	return s.catalog, nil
}

func (s *stubSongCatalogProvider) Refresh(ctx context.Context) error {
	s.refreshCount++
	if s.onRefresh != nil {
		s.onRefresh()
	}
	s.refreshCtxErr = ctx.Err()
	_, s.refreshHasDeadline = ctx.Deadline()
	return s.refreshErr
}

func TestRefreshSongCatalog(t *testing.T) {
	tests := []struct {
		name       string
		refreshErr error
	}{
		{name: "再構築に成功する", refreshErr: nil},
		{name: "再構築に失敗しても呼び出し元を止めない", refreshErr: errors.New("db down")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			catalog := &stubSongCatalogProvider{refreshErr: tt.refreshErr}
			// 保存後に応答前のリクエストが切断されても、再構築はキャンセルしない。
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			// When
			refreshSongCatalog(ctx, catalog)

			// Then
			assert.Equal(t, 1, catalog.refreshCount)
			assert.NoError(t, catalog.refreshCtxErr, "切断されたリクエストのキャンセルを引き継がない")
			assert.True(t, catalog.refreshHasDeadline, "再構築には上限時間を設ける")
		})
	}
}

func TestRefreshSongCatalog_カタログがない場合は何もしない(t *testing.T) {
	// When / Then
	assert.NotPanics(t, func() { refreshSongCatalog(context.Background(), nil) })
}

func TestSongUsecase_GetSongsUpdatedAt_公開中の楽曲カタログの更新時刻を返す(t *testing.T) {
	// Given
	updatedAt := time.Date(2026, time.November, 13, 1, 2, 3, 0, time.UTC)
	catalog := &stubSongCatalogProvider{catalog: songcatalog.New(3, nil, nil, &updatedAt)}
	mockRepo := new(MockSongRepository)
	uc := NewSongUsecaseWithCascadeDelete(mockRepo, nil, nil, nil, nil, nil, nil, catalog)

	// When
	result, err := uc.GetSongsUpdatedAt(context.Background())

	// Then
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.UpdatedAt)
	assert.Equal(t, updatedAt, *result.UpdatedAt)
	assert.Equal(t, uint64(3), result.Revision)
	// DBへ問い合わせないこと
	mockRepo.AssertNotCalled(t, "FindLatestUpdatedAt")
}

func TestSongUsecase_GetSongsUpdatedAt_カタログを構築できない場合はエラーを返す(t *testing.T) {
	// Given
	catalog := &stubSongCatalogProvider{snapshotErr: repository.ErrRepositoryOperationFailed}
	uc := NewSongUsecaseWithCascadeDelete(new(MockSongRepository), nil, nil, nil, nil, nil, nil, catalog)

	// When
	result, err := uc.GetSongsUpdatedAt(context.Background())

	// Then
	assert.ErrorIs(t, err, repository.ErrRepositoryOperationFailed)
	assert.Nil(t, result)
}

// newUserUsecaseWithSongCatalogForTest は未プレイ補完に楽曲カタログを使う UserUsecase を作ります。
func newUserUsecaseWithSongCatalogForTest(
	userRepo repository.UserRepository,
	playerRepo repository.PlayerRepository,
	playerRecordRepo repository.PlayerRecordRepository,
	worldsendRecordRepo repository.WorldsendRecordRepository,
	masterProvider userMasterProvider,
	songs []*entity.Song,
	worldsendSongs []*entity.WorldsendSongWithChart,
) UserUsecase {
	return NewUserUsecaseWithFirebaseDeleterAndOverpowerDenominator(
		nil, userRepo, playerRepo, playerRecordRepo, worldsendRecordRepo,
		nil, nil, masterProvider, nil, nil, nil, nil, nil,
		&stubSongCatalogProvider{catalog: songcatalog.New(1, songs, worldsendSongs, nil)},
		nil, nil, nil,
	)
}

func TestSongUsecase_DeleteSong_保存成功後だけ楽曲カタログを再構築する(t *testing.T) {
	tests := []struct {
		name             string
		saveErr          error
		wantRefreshCount int
	}{
		{name: "保存に成功すると応答前に再構築する", saveErr: nil, wantRefreshCount: 1},
		{name: "保存に失敗すると再構築しない", saveErr: repository.ErrRepositoryOperationFailed, wantRefreshCount: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			ctx := context.Background()
			mockRepo := new(MockSongRepository)
			mockExec := new(MockExecutor)
			catalog := &stubSongCatalogProvider{}
			uc := NewSongUsecaseWithCascadeDelete(mockRepo, nil, &passthroughTransactionManager{tx: mockExec}, mockExec, nil, nil, nil, catalog)
			song := &entity.Song{ID: 1, DisplayID: "S001", Charts: []*entity.Chart{}}
			mockRepo.On("FindByDisplayIDForChange", ctx, mockExec, "S001").Return(song, nil).Once()
			mockRepo.On("Save", ctx, mockExec, song).Return(tt.saveErr).Once()

			// When
			err := uc.DeleteSong(ctx, "S001")

			// Then
			assert.ErrorIs(t, err, tt.saveErr)
			assert.Equal(t, tt.wantRefreshCount, catalog.refreshCount)
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestSongUsecase_保存成功後に再構築が失敗しても保存結果は成功として返す(t *testing.T) {
	// Given
	ctx := context.Background()
	mockRepo := new(MockSongRepository)
	mockExec := new(MockExecutor)
	catalog := &stubSongCatalogProvider{refreshErr: repository.ErrRepositoryOperationFailed}
	uc := NewSongUsecaseWithCascadeDelete(mockRepo, nil, &passthroughTransactionManager{tx: mockExec}, mockExec, nil, nil, nil, catalog)
	song := &entity.Song{ID: 1, DisplayID: "S001", IsDeleted: true, Charts: []*entity.Chart{}}
	mockRepo.On("FindByDisplayIDForChange", ctx, mockExec, "S001").Return(song, nil).Once()
	mockRepo.On("Save", ctx, mockExec, song).Return(nil).Once()

	// When
	err := uc.RestoreSong(ctx, "S001")

	// Then
	assert.NoError(t, err)
	assert.Equal(t, 1, catalog.refreshCount)
}

func TestWorldsendUsecase_DeleteWorldsendSong_保存成功後だけ楽曲カタログを再構築する(t *testing.T) {
	tests := []struct {
		name             string
		saveErr          error
		wantRefreshCount int
	}{
		{name: "保存に成功すると応答前に再構築する", saveErr: nil, wantRefreshCount: 1},
		{name: "保存に失敗すると再構築しない", saveErr: repository.ErrRepositoryOperationFailed, wantRefreshCount: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			ctx := context.Background()
			mockRepo := new(MockWorldsendChartRepository)
			mockExec := new(MockExecutor)
			catalog := &stubSongCatalogProvider{}
			uc := NewWorldsendUsecase(mockRepo, &passthroughTransactionManager{tx: mockExec}, mockExec, catalog)
			songWithChart := &entity.WorldsendSongWithChart{Song: &entity.Song{ID: 1, DisplayID: "W001", Charts: []*entity.Chart{}}}
			mockRepo.On("FindByDisplayIDForChange", ctx, mockExec, "W001").Return(songWithChart, nil).Once()
			mockRepo.On("SaveSong", ctx, mockExec, songWithChart.Song).Return(tt.saveErr).Once()

			// When
			err := uc.DeleteWorldsendSong(ctx, "W001")

			// Then
			assert.ErrorIs(t, err, tt.saveErr)
			assert.Equal(t, tt.wantRefreshCount, catalog.refreshCount)
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestSongBatchJobUsecase_同期に成功した場合だけ楽曲カタログを再構築する(t *testing.T) {
	tests := []struct {
		name             string
		runner           *fakeSongBatchRunner
		wantRefreshCount int
	}{
		{name: "成功", runner: &fakeSongBatchRunner{}, wantRefreshCount: 1},
		{name: "失敗", runner: &fakeSongBatchRunner{err: errors.New("sync failed")}, wantRefreshCount: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			catalog := &stubSongCatalogProvider{}
			uc := newTestSongBatchJobUsecase(context.Background(), &fakeSongBatchLockProvider{acquired: true}, newFakeSongBatchJobRepository(), tt.runner, nil)
			uc.songCatalog = catalog

			// When
			_, _ = uc.RunFromCLI(context.Background(), songbatch.NewRunRequest(false, false))

			// Then
			assert.Equal(t, tt.wantRefreshCount, catalog.refreshCount)
		})
	}
}

func TestSongUsecase_GetSongsUpdatedAt_カタログを持たない構成では内部エラーを返す(t *testing.T) {
	// Given
	uc := NewSongUsecase(new(MockSongRepository), nil, nil, nil)

	// When
	result, err := uc.GetSongsUpdatedAt(context.Background())

	// Then
	assert.ErrorIs(t, err, ErrInternalError)
	assert.Nil(t, result)
}

func TestSongBatchJobUsecase_キャッシュ反映後にジョブの完了を記録する(t *testing.T) {
	tests := []struct {
		name       string
		refreshErr error
	}{
		{name: "再構築に成功した場合", refreshErr: nil},
		{name: "再構築に失敗してもDB同期は成功として記録する", refreshErr: repository.ErrRepositoryOperationFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			repo := newFakeSongBatchJobRepository()
			catalog := &stubSongCatalogProvider{refreshErr: tt.refreshErr}
			var statusAtRefresh entity.SongBatchJobStatus
			catalog.onRefresh = func() {
				job := repo.only(t)
				statusAtRefresh = job.Status()
			}
			uc := newTestSongBatchJobUsecase(context.Background(), &fakeSongBatchLockProvider{acquired: true}, repo, &fakeSongBatchRunner{}, nil)
			uc.songCatalog = catalog

			// When
			_, err := uc.RunFromCLI(context.Background(), songbatch.NewRunRequest(false, false))

			// Then: 管理画面が完了を検知した時点で、キャッシュの反映を試み終えている
			require.NoError(t, err)
			assert.Equal(t, entity.SongBatchJobStatusRunning, statusAtRefresh)
			job := repo.only(t)
			assert.Equal(t, entity.SongBatchJobStatusSucceeded, job.Status())
		})
	}
}
