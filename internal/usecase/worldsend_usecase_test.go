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

func newWorldsendUsecaseForTest(repo repository.WorldsendChartRepository, tm TransactionManager, exec repository.Executor) WorldsendUsecase {
	return NewWorldsendUsecase(repo, tm, exec)
}

func TestGetAllWorldsendSongs_WithDeletedSongs_RequiresEditorPermission(t *testing.T) {
	tests := []struct {
		name                   string
		includeDeleted         bool
		requesterAccountTypeID *int
		expectedIncludeDeleted bool
	}{
		{
			name:                   "EDITOR権限あり_includeDeleted=true_削除済みを含む",
			includeDeleted:         true,
			requesterAccountTypeID: intPtr(info.AccountTypeEditor),
			expectedIncludeDeleted: true,
		},
		{
			name:                   "ADMIN権限あり_includeDeleted=true_削除済みを含む",
			includeDeleted:         true,
			requesterAccountTypeID: intPtr(info.AccountTypeAdmin),
			expectedIncludeDeleted: true,
		},
		{
			name:                   "PLAYER権限のみ_includeDeleted=true_削除済みを除外",
			includeDeleted:         true,
			requesterAccountTypeID: intPtr(info.AccountTypePlayer),
			expectedIncludeDeleted: false,
		},
		{
			name:                   "権限なし_includeDeleted=true_削除済みを除外",
			includeDeleted:         true,
			requesterAccountTypeID: nil,
			expectedIncludeDeleted: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockRepo := new(MockWorldsendChartRepository)
			mockTM := new(MockTransactionManager)
			mockExec := new(MockExecutor)
			uc := newWorldsendUsecaseForTest(mockRepo, mockTM, mockExec)
			ctx := context.Background()

			expectedSongs := []*repository.WorldsendSongWithChart{{
				Song:  &entity.Song{ID: 1, DisplayID: "WE001", IsWorldsend: true, IsDeleted: false},
				Chart: &entity.WorldsendChart{ID: 1, SongID: 1},
			}}
			mockRepo.On("FindAll", ctx, mockExec, tt.expectedIncludeDeleted).Return(expectedSongs, nil)

			// When
			result, err := uc.GetAllWorldsendSongs(ctx, tt.includeDeleted, tt.requesterAccountTypeID)

			// Then
			assert.NoError(t, err)
			assert.Equal(t, expectedSongs, result)
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestGetWorldsendSongByDisplayID_DeletedSongPermission(t *testing.T) {
	activeSong := &repository.WorldsendSongWithChart{
		Song:  &entity.Song{ID: 1, DisplayID: "WE001", IsWorldsend: true, IsDeleted: false},
		Chart: &entity.WorldsendChart{ID: 1, SongID: 1},
	}
	deletedSong := &repository.WorldsendSongWithChart{
		Song:  &entity.Song{ID: 2, DisplayID: "WE002", IsWorldsend: true, IsDeleted: true},
		Chart: &entity.WorldsendChart{ID: 2, SongID: 2},
	}

	tests := []struct {
		name                   string
		displayID              string
		requesterAccountTypeID *int
		repoReturn             *repository.WorldsendSongWithChart
		repoErr                error
		wantResult             *repository.WorldsendSongWithChart
		wantErr                error
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
			uc := newWorldsendUsecaseForTest(mockRepo, mockTM, mockExec)
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
	uc := newWorldsendUsecaseForTest(mockRepo, tm, mockExec)
	ctx := context.Background()

	songWithChart := &repository.WorldsendSongWithChart{
		Song:  &entity.Song{ID: 21, DisplayID: "WE021", IsWorldsend: true, IsDeleted: false, Charts: []*entity.Chart{}},
		Chart: &entity.WorldsendChart{ID: 210, SongID: 21},
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
	uc := newWorldsendUsecaseForTest(mockRepo, tm, mockExec)
	ctx := context.Background()

	songWithChart := &repository.WorldsendSongWithChart{
		Song:  &entity.Song{ID: 22, DisplayID: "WE022", IsWorldsend: true, IsDeleted: true, Charts: []*entity.Chart{}},
		Chart: &entity.WorldsendChart{ID: 220, SongID: 22},
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

func TestUpdateWorldsendSongs_SavesEntities(t *testing.T) {
	// Given
	mockRepo := new(MockWorldsendChartRepository)
	mockExec := new(MockExecutor)
	tm := &passthroughTransactionManager{tx: mockExec}
	uc := newWorldsendUsecaseForTest(mockRepo, tm, mockExec)
	ctx := context.Background()

	songs := []*entity.Song{
		{DisplayID: "1234567890abcdef", Title: "A", Artist: "AR", IsWorldsend: true},
		{DisplayID: "abcdef1234567890", Title: "B", Artist: "BR", IsWorldsend: true},
	}
	charts := []*entity.WorldsendChart{
		nil,
		{},
	}

	mockRepo.On("UpdateSongs", ctx, mockExec, songs, charts).Return(nil).Once()

	// When
	err := uc.UpdateWorldsendSongs(ctx, songs, charts)

	// Then
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestUpdateWorldsendSongs_EmptySongsIsNoOp(t *testing.T) {
	// Given
	mockRepo := new(MockWorldsendChartRepository)
	mockExec := new(MockExecutor)
	tm := &passthroughTransactionManager{tx: mockExec}
	uc := newWorldsendUsecaseForTest(mockRepo, tm, mockExec)

	// When
	err := uc.UpdateWorldsendSongs(context.Background(), []*entity.Song{}, []*entity.WorldsendChart{})

	// Then
	assert.NoError(t, err)
	mockRepo.AssertNotCalled(t, "UpdateSongs", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestUpdateWorldsendSongs_LengthMismatchReturnsError(t *testing.T) {
	// Given
	mockRepo := new(MockWorldsendChartRepository)
	mockExec := new(MockExecutor)
	tm := &passthroughTransactionManager{tx: mockExec}
	uc := newWorldsendUsecaseForTest(mockRepo, tm, mockExec)
	songs := []*entity.Song{{DisplayID: "1234567890abcdef", IsWorldsend: true}}
	charts := []*entity.WorldsendChart{}

	// When
	err := uc.UpdateWorldsendSongs(context.Background(), songs, charts)

	// Then
	assert.ErrorIs(t, err, ErrInvalidWorldsendInput)
	assert.ErrorContains(t, err, "songs and charts length mismatch")
	mockRepo.AssertNotCalled(t, "UpdateSongs", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestUpdateWorldsendSongs_NilSongReturnsError(t *testing.T) {
	// Given
	mockRepo := new(MockWorldsendChartRepository)
	mockExec := new(MockExecutor)
	tm := &passthroughTransactionManager{tx: mockExec}
	uc := newWorldsendUsecaseForTest(mockRepo, tm, mockExec)
	songs := []*entity.Song{nil}
	charts := []*entity.WorldsendChart{nil}

	// When
	err := uc.UpdateWorldsendSongs(context.Background(), songs, charts)

	// Then
	assert.ErrorIs(t, err, ErrInvalidWorldsendInput)
	assert.ErrorContains(t, err, "songs[0] is null")
	mockRepo.AssertNotCalled(t, "UpdateSongs", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// TestUpdateWorldsendSongs_DuplicateDisplayIDIsMappedToValidationError はリポジトリ防衛チェックが
// ErrDuplicateDisplayID を返した際に ErrInvalidWorldsendInput へ変換されることを検証します。
func TestUpdateWorldsendSongs_DuplicateDisplayIDIsMappedToValidationError(t *testing.T) {
	// Given
	mockRepo := new(MockWorldsendChartRepository)
	mockExec := new(MockExecutor)
	tm := &passthroughTransactionManager{tx: mockExec}
	uc := newWorldsendUsecaseForTest(mockRepo, tm, mockExec)
	songs := []*entity.Song{
		{
			DisplayID:   "1234567890abcdef",
			Title:       "A",
			Artist:      "AR",
			IsWorldsend: true,
		},
		{
			DisplayID:   "1234567890abcdef",
			Title:       "B",
			Artist:      "BR",
			IsWorldsend: true,
		},
	}
	charts := []*entity.WorldsendChart{nil, nil}

	mockRepo.On("UpdateSongs", mock.Anything, mockExec,
		mock.MatchedBy(func(songs []*entity.Song) bool {
			return len(songs) == 2 && songs[0] != nil && songs[1] != nil && songs[0].DisplayID == songs[1].DisplayID
		}),
		charts,
	).
		Return(repository.ErrDuplicateDisplayID).
		Once()

	// When
	err := uc.UpdateWorldsendSongs(context.Background(), songs, charts)

	// Then
	assert.ErrorIs(t, err, ErrInvalidWorldsendInput)
	assert.ErrorIs(t, err, repository.ErrDuplicateDisplayID)
	mockRepo.AssertExpectations(t)
}

// intPtr はint値へのポインタを返すヘルパー関数です。
func intPtr(i int) *int {
	return &i
}
