package usecase

import (
	"context"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockWorldsendChartRepository は WorldsendChartRepository のモックです。
type MockWorldsendChartRepository struct {
	mock.Mock
}

func (m *MockWorldsendChartRepository) FindAll(ctx context.Context, exec repository.Executor, includeDeleted bool) ([]*repository.WorldsendSongWithChart, error) {
	args := m.Called(ctx, exec, includeDeleted)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*repository.WorldsendSongWithChart), args.Error(1)
}

func (m *MockWorldsendChartRepository) FindByDisplayID(ctx context.Context, exec repository.Executor, displayID string) (*repository.WorldsendSongWithChart, error) {
	args := m.Called(ctx, exec, displayID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.WorldsendSongWithChart), args.Error(1)
}

func (m *MockWorldsendChartRepository) SaveSong(ctx context.Context, exec repository.Executor, song *entity.Song) error {
	args := m.Called(ctx, exec, song)
	return args.Error(0)
}

func (m *MockWorldsendChartRepository) UpdateSongs(ctx context.Context, exec repository.Executor, songs []*entity.Song, charts []*entity.WorldsendChart) error {
	args := m.Called(ctx, exec, songs, charts)
	return args.Error(0)
}

// MockTransactionManager は TransactionManager のモックです。
type MockTransactionManager struct {
	mock.Mock
}

func (m *MockTransactionManager) Transactional(ctx context.Context, fn func(tx repository.Executor) error) error {
	args := m.Called(ctx, fn)
	return args.Error(0)
}

func TestGetAllWorldsendSongs_WithDeletedSongs_RequiresEditorPermission(t *testing.T) {
	tests := []struct {
		name                   string
		includeDeleted         bool
		requesterAccountTypeID *int
		expectedIncludeDeleted bool
		description            string
	}{
		{
			name:                   "EDITOR権限あり_includeDeleted=true_削除済みを含む",
			includeDeleted:         true,
			requesterAccountTypeID: intPtr(info.AccountTypeEditor),
			expectedIncludeDeleted: true,
			description:            "EDITOR権限がある場合、includeDeleted=trueで削除済み楽曲を取得できる",
		},
		{
			name:                   "ADMIN権限あり_includeDeleted=true_削除済みを含む",
			includeDeleted:         true,
			requesterAccountTypeID: intPtr(info.AccountTypeAdmin),
			expectedIncludeDeleted: true,
			description:            "ADMIN権限がある場合、includeDeleted=trueで削除済み楽曲を取得できる",
		},
		{
			name:                   "PLAYER権限のみ_includeDeleted=true_削除済みを除外",
			includeDeleted:         true,
			requesterAccountTypeID: intPtr(info.AccountTypePlayer),
			expectedIncludeDeleted: false,
			description:            "PLAYER権限の場合、includeDeleted=trueでも削除済み楽曲は除外される",
		},
		{
			name:                   "権限なし_includeDeleted=true_削除済みを除外",
			includeDeleted:         true,
			requesterAccountTypeID: nil,
			expectedIncludeDeleted: false,
			description:            "権限がない場合、includeDeleted=trueでも削除済み楽曲は除外される",
		},
		{
			name:                   "権限なし_includeDeleted=false_削除済みを除外",
			includeDeleted:         false,
			requesterAccountTypeID: nil,
			expectedIncludeDeleted: false,
			description:            "includeDeleted=falseの場合、削除済み楽曲は除外される",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockRepo := new(MockWorldsendChartRepository)
			mockTM := new(MockTransactionManager)
			mockExec := new(MockExecutor)

			usecase := NewWorldsendUsecase(mockRepo, mockTM, mockExec)

			ctx := context.Background()

			// 期待されるリポジトリの呼び出し
			expectedSongs := []*repository.WorldsendSongWithChart{
				{
					Song: &entity.Song{
						ID:          1,
						DisplayID:   "WE001",
						Title:       "Active Song",
						IsWorldsend: true,
						IsDeleted:   false,
					},
					Chart: &entity.WorldsendChart{
						ID:     1,
						SongID: 1,
					},
				},
			}

			// tt.expectedIncludeDeleted に基づいてリポジトリが呼び出されることを期待
			mockRepo.On("FindAll", ctx, mockExec, tt.expectedIncludeDeleted).Return(expectedSongs, nil)

			// When
			result, err := usecase.GetAllWorldsendSongs(ctx, tt.includeDeleted, tt.requesterAccountTypeID)

			// Then
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, expectedSongs, result)
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestGetWorldsendSongByDisplayID_DeletedSongPermission(t *testing.T) {
	activeSong := &repository.WorldsendSongWithChart{
		Song: &entity.Song{
			ID:          1,
			DisplayID:   "WE001",
			Title:       "Active Song",
			IsWorldsend: true,
			IsDeleted:   false,
		},
		Chart: &entity.WorldsendChart{ID: 1, SongID: 1},
	}
	deletedSong := &repository.WorldsendSongWithChart{
		Song: &entity.Song{
			ID:          2,
			DisplayID:   "WE002",
			Title:       "Deleted Song",
			IsWorldsend: true,
			IsDeleted:   true,
		},
		Chart: &entity.WorldsendChart{ID: 2, SongID: 2},
	}

	tests := []struct {
		name string
		// Given
		displayID              string
		requesterAccountTypeID *int
		repoReturn             *repository.WorldsendSongWithChart
		repoErr                error
		// Then
		wantResult *repository.WorldsendSongWithChart
		wantErr    error
	}{
		{
			name:                   "有効な楽曲は誰でも取得できる",
			displayID:              "WE001",
			requesterAccountTypeID: nil,
			repoReturn:             activeSong,
			wantResult:             activeSong,
		},
		{
			name:                   "削除済み楽曲はEDITOR権限で取得できる",
			displayID:              "WE002",
			requesterAccountTypeID: intPtr(info.AccountTypeEditor),
			repoReturn:             deletedSong,
			wantResult:             deletedSong,
		},
		{
			name:                   "削除済み楽曲はPLAYER権限ではErrSongNotFoundになる",
			displayID:              "WE002",
			requesterAccountTypeID: intPtr(info.AccountTypePlayer),
			repoReturn:             deletedSong,
			wantErr:                repository.ErrSongNotFound,
		},
		{
			name:                   "削除済み楽曲は権限なしではErrSongNotFoundになる",
			displayID:              "WE002",
			requesterAccountTypeID: nil,
			repoReturn:             deletedSong,
			wantErr:                repository.ErrSongNotFound,
		},
		{
			name:                   "存在しない楽曲はErrSongNotFoundを返す",
			displayID:              "WE999",
			requesterAccountTypeID: intPtr(info.AccountTypeAdmin),
			repoErr:                repository.ErrSongNotFound,
			wantErr:                repository.ErrSongNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockRepo := new(MockWorldsendChartRepository)
			mockTM := new(MockTransactionManager)
			mockExec := new(MockExecutor)

			uc := NewWorldsendUsecase(mockRepo, mockTM, mockExec)
			ctx := context.Background()

			mockRepo.On("FindByDisplayID", ctx, mockExec, tt.displayID).Return(tt.repoReturn, tt.repoErr)

			// When
			result, err := uc.GetWorldsendSongByDisplayID(ctx, tt.displayID, tt.requesterAccountTypeID)

			// Then
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantResult, result)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestDeleteWorldsendSong_SavesDeletedState(t *testing.T) {
	// Given
	mockRepo := new(MockWorldsendChartRepository)
	mockExec := new(MockExecutor)
	tm := &passthroughTransactionManager{tx: mockExec}
	uc := NewWorldsendUsecase(mockRepo, tm, mockExec)
	ctx := context.Background()

	songWithChart := &repository.WorldsendSongWithChart{
		Song: &entity.Song{
			ID:          21,
			DisplayID:   "WE021",
			IsWorldsend: true,
			IsDeleted:   false,
			Charts:      []*entity.Chart{},
		},
		Chart: &entity.WorldsendChart{
			ID:     210,
			SongID: 21,
		},
	}
	mockRepo.On("FindByDisplayID", ctx, mockExec, "WE021").Return(songWithChart, nil).Once()
	mockRepo.On("SaveSong", ctx, mockExec, mock.MatchedBy(func(song *entity.Song) bool {
		return song == songWithChart.Song && song.IsDeleted
	})).Return(nil).Once()

	// When
	err := uc.DeleteWorldsendSong(ctx, "WE021")

	// Then
	assert.NoError(t, err)
	assert.True(t, songWithChart.Song.IsDeleted)
	mockRepo.AssertExpectations(t)
}

func TestRestoreWorldsendSong_SavesRestoredState(t *testing.T) {
	// Given
	mockRepo := new(MockWorldsendChartRepository)
	mockExec := new(MockExecutor)
	tm := &passthroughTransactionManager{tx: mockExec}
	uc := NewWorldsendUsecase(mockRepo, tm, mockExec)
	ctx := context.Background()

	songWithChart := &repository.WorldsendSongWithChart{
		Song: &entity.Song{
			ID:          22,
			DisplayID:   "WE022",
			IsWorldsend: true,
			IsDeleted:   true,
			Charts:      []*entity.Chart{},
		},
		Chart: &entity.WorldsendChart{
			ID:     220,
			SongID: 22,
		},
	}
	mockRepo.On("FindByDisplayID", ctx, mockExec, "WE022").Return(songWithChart, nil).Once()
	mockRepo.On("SaveSong", ctx, mockExec, mock.MatchedBy(func(song *entity.Song) bool {
		return song == songWithChart.Song && !song.IsDeleted
	})).Return(nil).Once()

	// When
	err := uc.RestoreWorldsendSong(ctx, "WE022")

	// Then
	assert.NoError(t, err)
	assert.False(t, songWithChart.Song.IsDeleted)
	mockRepo.AssertExpectations(t)
}

// intPtr はint値へのポインタを返すヘルパー関数です。
func intPtr(i int) *int {
	return &i
}
